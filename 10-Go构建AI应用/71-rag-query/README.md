# PostgreSQL + pgvector RAG 查询

示例提供一个 CLI 查询链路：将问题编码成 embedding，在 pgvector 中按租户和知识集合检索片段，再请求模型根据命中内容生成回答。需要 PostgreSQL 16、pgvector，以及支持 OpenAI 兼容 embeddings 和 chat completions 的端点。

## 准备数据库

```bash
docker run --name go-book-pgvector -e POSTGRES_PASSWORD=dev \
  -p 5432:5432 -d pgvector/pgvector:pg16
psql 'postgres://postgres:dev@localhost:5432/postgres?sslmode=disable' \
  -f schema.sql
```

## 导入和查询

示例提供经过人工审核的 Markdown 文档导入入口。租户和集合从环境变量读取，文档版本由导入任务显式指定：

```bash
TENANT_ID=demo-tenant COLLECTION_ID=support-policy \
go run . -mode ingest -file ./policies/refund.md -key refund-policy \
  -title '退款与优惠券规则' -source-uri 'support://policy/refund' -version '2026-10-01'
```

导入会先在事务中将该文档的旧片段标为 inactive，写入全部新版本后再提交。若读取、embedding 或入库失败，事务回滚，旧版本仍可查询。查询命令只会读取 active 片段：

使用本地 Ollama 时，先启动 Ollama 并下载两个模型：

```bash
ollama pull nomic-embed-text
ollama pull qwen2.5:3b
```

embedding 模型须输出 768 维向量，与 `schema.sql` 一致。导入和查询都使用同一个 embedding 模型。

```bash
TENANT_ID=demo-tenant COLLECTION_ID=support-policy \
go run . -mode query -question '已支付订单取消后，优惠券如何处理？'
```

环境变量可覆盖 `DATABASE_URL`、`TENANT_ID`、`COLLECTION_ID`、`EMBEDDING_ENDPOINT`、`EMBEDDING_MODEL`、`CHAT_ENDPOINT`、`CHAT_MODEL` 和 `LLM_API_KEY`。不同模型的维度不同时，必须同步调整 schema、模型配置和维度校验。

单次导入限制为 256 KiB 和 100 个片段，避免 CLI 同步处理大型知识库。生产系统应将解析和 embedding 放进可重试的后台任务，并在新版本完整构建后切换。

## 表结构

`schema.sql` 使用 HNSW cosine 索引。示例查询显式带 tenant ID 与 collection ID；生产环境应由认证中间件提供租户，并可叠加 PostgreSQL RLS。应用运行账号不应是表所有者或超级用户，否则可能绕过 RLS。
