# OpenAI Responses API Go 示例

使用 OpenAI 官方 Go SDK v3.70.0 发起一次文本请求。SDK 默认从 `OPENAI_API_KEY` 读取密钥，模型默认值可以通过 `OPENAI_MODEL` 调整。

```bash
export OPENAI_API_KEY='由密钥管理系统注入的 key'
go run . -question '解释订单服务中为什么要给下游请求设置 deadline。'
```

SDK v3.45.0 及之后版本要求 Go 1.25 或更新版本。升级时查看官方 changelog，并重新核对模型权限、输出类型和错误处理。
