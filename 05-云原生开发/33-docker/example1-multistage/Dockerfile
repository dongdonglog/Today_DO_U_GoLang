# ============ 阶段1:构建 ============
# golang:1.25-alpine 体积小(约 250MB),自带 Go 工具链
FROM golang:1.25-alpine AS builder

WORKDIR /src

# 先拷贝 go.mod/go.sum 再拷源码:依赖不变时,这一层走缓存,docker build 秒级完成
# (无第三方依赖时只有 go.mod;go.sum 一旦生成必须一起拷)
COPY go.mod go.sum* ./
RUN go mod download

COPY . .

# CGO_ENABLED=0:纯静态编译,不依赖 glibc,可跑在 scratch
# -trimpath:去掉本机路径,产物可复现
# -ldflags "-s -w":去掉符号表与调试信息,体积再减 30%
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/app .

# ============ 阶段2:运行 ============
# scratch 是空镜像:最终镜像里只有我们的二进制,约 5~8MB
FROM scratch

# 静态二进制不需要 libc,但 HTTPS/时间解析需要 CA 证书;纯 HTTP 服务可不拷
# COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /out/app /app

# 声明端口(文档作用;真正限制要靠 -p 映射或 K8s)
EXPOSE 8080

# 容器内以非 root 运行需要 USER 指令;scratch 无 /etc/passwd,直接写 UID
USER 10001:10001

ENTRYPOINT ["/app"]
