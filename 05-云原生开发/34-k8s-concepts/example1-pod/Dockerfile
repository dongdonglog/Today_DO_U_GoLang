# 版本化镜像:VERSION 构建参数注入,滚动更新用 v1/v2 区分
FROM golang:1.25-alpine AS builder
ARG VERSION=v1
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
# 版本号在编译期写进二进制,运行时 /info 返回
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/app .

FROM scratch
COPY --from=builder /out/app /app
EXPOSE 8080
USER 10001:10001
ENTRYPOINT ["/app"]
