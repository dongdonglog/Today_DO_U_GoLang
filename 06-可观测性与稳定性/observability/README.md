# 可观测性示例工程

该工程把第 41–46 章的配置放在一起：Go 订单 API 输出 Prometheus 指标和结构化日志，通过 OTLP/HTTP 发送 Trace；Prometheus 负责抓取和规则计算，Grafana 预置指标看板并连接 Tempo 查询 Trace。

## 启动

需要 Docker Compose。首次启动会构建 Go 服务并拉取示例镜像：

```bash
docker compose up -d --build
docker compose ps
curl -i http://localhost:8080/healthz
curl -X POST http://localhost:8080/orders
```

本地入口：

| 组件 | 地址 | 用途 |
|---|---|---|
| Order API | `http://localhost:8080` | `/orders`、`/healthz`、`/metrics` |
| Prometheus | `http://localhost:9090` | PromQL、Targets、告警规则 |
| Grafana | `http://localhost:3000` | 指标看板与 Tempo Explore |
| Tempo API | `http://localhost:3200` | Grafana 查询所用的 Trace API |

Grafana 首次登录账号为 `admin` / `admin`，仅限本机学习环境；生产环境必须使用 Secret 注入强凭据并启用身份认证。本地配置将 Trace 保存在容器临时目录，删除容器后数据会丢失。

## 运行压测

```bash
docker compose --profile load run --rm k6
```

k6 会连接同一个 Compose 网络中的 `order-api`，运行约 9 分钟的分阶段负载。请勿把该脚本指向共享或生产环境；运行前确认本机 CPU、内存和其他服务不会受影响。

## 目录

| 路径 | 内容 |
|---|---|
| `app/` | Go HTTP 服务、Prometheus 指标、OTel Trace 和 JSON 日志 |
| `prometheus/` | 抓取配置与告警规则 |
| `grafana/` | Prometheus/Tempo datasource 与 dashboard provisioning |
| `otel/` | 接收 OTLP 并导出到 Tempo 的 Collector 配置 |
| `tempo/` | 本地 Trace 存储配置 |
| `load/` | k6 分阶段压测脚本 |

Compose 镜像和 Go module 都固定了示例版本以便复现。升级生产版本前，检查各组件兼容矩阵、镜像安全公告和配置变更，并在预发环境验证。
