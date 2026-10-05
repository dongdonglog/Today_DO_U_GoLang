# Eino ChatTemplate + ChatModel Chain

程序将客服问题填入 ChatTemplate，再交给 Eino OpenAI ChatModel。Chain 输入和输出类型固定，示例不开放任何业务工具。

```bash
export OPENAI_API_KEY='由密钥管理系统注入的 key'
export OPENAI_MODEL='账号已开通的模型名'
go run . -question '订单已经进入配送，还能直接取消吗？'
```

如需兼容服务，可设置 `OPENAI_BASE_URL`，但必须确认该服务支持扩展使用的 Chat Completions 请求格式。Eino v0.9.21 与 OpenAI 扩展 v0.1.13 固定在本示例的 `go.mod` 中。
