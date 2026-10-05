# Ollama 本地 Chat 示例

先安装并启动 Ollama，再拉取示例模型：

```bash
ollama pull qwen2.5:3b
```

运行：

```bash
go run . -question '为什么客服答案需要附上知识来源？'
```

`OLLAMA_MODEL` 可覆盖模型，`OLLAMA_BASE_URL` 默认为 `http://localhost:11434`。示例使用 `/api/chat` 并设置 `stream: false`。不要将无认证的 Ollama 端口暴露给不可信网络。
