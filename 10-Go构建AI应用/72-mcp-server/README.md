# MCP 知识库只读工具

这个 stdio Server 使用官方 Go SDK 暴露 `search_support_policy` 工具，查询第 71 章的 PostgreSQL/pgvector 知识库。工具固定在启动时配置的租户和集合中工作，不接收任意 SQL、表名或租户 ID。

```bash
go mod tidy
go build -o ./mcp-policy-server .
```

Host 启动该二进制时需要提供：

- `DATABASE_URL`：只读数据库账号的连接串
- `TENANT_ID`：该进程允许访问的租户
- `COLLECTION_ID`：允许搜索的知识集合
- `EMBEDDING_ENDPOINT`、`EMBEDDING_MODEL`：与知识库入库时一致的 embedding 配置
- `LLM_API_KEY`：只有 embedding 服务要求鉴权时才设置

本机开发可由 MCP Host 的子进程环境注入这些配置。stdio 协议占用 stdin/stdout，程序日志只输出到 stderr。不要将数据库密码提交到 Host 配置仓库。

Go SDK v1.8.0 支持 MCP 规范 2026-07-28，要求 Go 1.25 或以上。远程多租户场景不能复用本示例的进程级 `TENANT_ID`；应改为请求级身份认证，并在 Server 和数据库层校验授权。
