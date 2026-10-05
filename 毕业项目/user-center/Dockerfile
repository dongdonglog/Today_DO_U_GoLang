FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/user-center ./cmd/api

FROM alpine:3.20
RUN addgroup -S app && adduser -S app -G app
USER app
WORKDIR /app
COPY --from=build /out/user-center /app/user-center
EXPOSE 8080
ENTRYPOINT ["/app/user-center"]
