package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	requestCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Completed HTTP requests.",
	}, []string{"route", "method", "code"})

	requestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration in seconds.",
		Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5},
	}, []string{"route"})

	inFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "http_in_flight_requests",
		Help: "Requests currently being handled.",
	})
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serviceName := getenv("SERVICE_NAME", "order-api")
	otelEndpoint := getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector:4318")
	httpAddr := getenv("HTTP_ADDR", ":8080")

	provider, err := newTraceProvider(context.Background(), serviceName, otelEndpoint)
	if err != nil {
		logger.Error("initialize tracing", "error", err)
		os.Exit(1)
	}
	otel.SetTracerProvider(provider)

	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector(), requestCount, requestDuration, inFlight)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready\n"))
	})
	mux.Handle("/orders", observe("orders", logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		span := trace.SpanFromContext(r.Context())
		span.SetAttributes(attribute.String("order.result", "accepted"))
		logger.InfoContext(r.Context(), "order accepted",
			"route", "orders",
			"trace_id", span.SpanContext().TraceID().String(),
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
	})))

	server := &http.Server{
		Addr:              httpAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server stopped", "error", err)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("HTTP shutdown", "error", err)
		}
		if err := provider.Shutdown(shutdownCtx); err != nil {
			logger.Error("trace shutdown", "error", err)
		}
	}
}

func newTraceProvider(ctx context.Context, serviceName, endpoint string) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}
	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		)),
	), nil
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func observe(route string, logger *slog.Logger, next http.Handler) http.Handler {
	tracer := otel.Tracer("order-api")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inFlight.Inc()
		defer inFlight.Dec()
		started := time.Now()

		ctx, span := tracer.Start(r.Context(), "http "+route,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.route", "/orders"),
				attribute.String("http.request.method", r.Method),
			),
		)
		defer span.End()

		response := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(response, r.WithContext(ctx))
		if response.status == 0 {
			response.status = http.StatusOK
		}

		requestCount.WithLabelValues(route, r.Method, strconv.Itoa(response.status)).Inc()
		requestDuration.WithLabelValues(route).Observe(time.Since(started).Seconds())
		span.SetAttributes(attribute.Int("http.response.status_code", response.status))
		if response.status >= http.StatusInternalServerError {
			span.SetStatus(codes.Error, "server error")
			logger.ErrorContext(ctx, "request failed", "route", route, "status", response.status)
		}
	})
}
