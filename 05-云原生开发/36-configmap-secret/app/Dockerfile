# ConfigMap/Secret 演示镜像(复用第33章多阶段模式)
FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/app .

FROM scratch
COPY --from=builder /out/app /app
EXPOSE 8080
USER 10001:10001
ENTRYPOINT ["/app"]
