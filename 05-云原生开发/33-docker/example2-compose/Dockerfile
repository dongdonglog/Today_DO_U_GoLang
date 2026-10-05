# 多阶段构建:构建产物只有 6MB 左右的静态二进制(原理见 example1)
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
