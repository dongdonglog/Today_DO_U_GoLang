FROM golang:1.25-alpine AS build
ARG APP_NAME
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/app ./cmd/$APP_NAME

FROM alpine:3.20
RUN addgroup -S app && adduser -S app -G app
USER app
WORKDIR /app
COPY --from=build /out/app /app/app
ENTRYPOINT ["/app/app"]
