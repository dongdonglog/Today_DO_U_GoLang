FROM golang:1.25-alpine AS build

WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/knowledge-api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/knowledge-api /knowledge-api
EXPOSE 8080
ENTRYPOINT ["/knowledge-api"]
