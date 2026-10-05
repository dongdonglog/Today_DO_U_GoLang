package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/go-book/76-ai-knowledge-base/internal/answer"
	"example.com/go-book/76-ai-knowledge-base/internal/config"
	"example.com/go-book/76-ai-knowledge-base/internal/embedding"
	"example.com/go-book/76-ai-knowledge-base/internal/httpapi"
	"example.com/go-book/76-ai-knowledge-base/internal/knowledge"
	"example.com/go-book/76-ai-knowledge-base/internal/postgres"
)

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run() error {
	if err := config.LoadDotEnv(".env"); err != nil {
		return fmt.Errorf("load .env: %w", err)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := cfg.ValidateAPI(); err != nil {
		return err
	}
	startup, cancelStartup := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStartup()
	store, err := postgres.Open(startup, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer store.Close()
	embedder, err := embedding.NewClient(cfg.OllamaBaseURL, cfg.EmbeddingModel)
	if err != nil {
		return err
	}
	generator, err := answer.New(startup, cfg.OllamaBaseURL, cfg.OllamaAPIKey, cfg.ChatModel)
	if err != nil {
		return err
	}
	service, err := knowledge.NewService(store, embedder, generator, cfg.MaxDistance)
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           httpapi.New(service, cfg),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	shutdown, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serveError := make(chan error, 1)
	go func() { serveError <- server.ListenAndServe() }()
	log.Printf("knowledge API listening on %s", cfg.ListenAddr)
	select {
	case err := <-serveError:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
	case <-shutdown.Done():
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown HTTP: %w", err)
		}
	}
	return nil
}
