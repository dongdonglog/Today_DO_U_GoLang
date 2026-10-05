# AI 知识库平台

这个目录是毕业项目四的阅读入口。实际可运行工程放在第 76 章目录：../../10-Go构建AI应用/76-ai-knowledge-base/README.md。这样做可以让 AI 知识库既作为第 76 章的完整收束，也作为毕业项目进入统一导航。

![AI 知识库工程的导入、查询与 MCP 只读访问边界](./images/knowledge-base-query.svg)

> **图解**：管理员导入资料时，系统先校验角色，再解析、分块、生成 embedding，并按租户、集合、版本和来源写入 pgvector；用户查询时，JWT 决定租户范围，检索服务只返回当前租户和集合的命中片段。MCP stdio Server 使用只读账号复用检索服务，不能写数据，也不能接收任意 SQL。

## 工程位置

- 可运行工程：../../10-Go构建AI应用/76-ai-knowledge-base/README.md
- 章节说明：../../10-Go构建AI应用/76-AI知识库系统.md

## 验收重点

- 管理员能导入文档，普通读者不能导入。
- 查询只返回当前租户和集合的资料。
- 没有证据时拒答或说明证据不足。
- citations 只来自本次真实检索命中。
- MCP 工具只读，不能写入数据库或执行任意 SQL。

本项目依赖 Ollama、PostgreSQL/pgvector、JWT、Eino 和 MCP。启动、导入、查询、MCP 验证、故障演练和上线检查都写在实际工程 README 中。
