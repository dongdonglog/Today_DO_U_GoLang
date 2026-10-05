FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/order-api .

FROM scratch
COPY --from=builder /out/order-api /order-api
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["/order-api"]
