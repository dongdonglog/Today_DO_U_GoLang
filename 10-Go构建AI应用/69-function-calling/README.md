# Function Calling 查单示例

程序将一个只读查单工具声明给 OpenAI 兼容模型端点。模型提出调用后，Go 服务校验订单号，并调用内部订单 API；客服身份从环境变量模拟已认证请求上下文。

```bash
ORDER_SERVICE_URL=http://localhost:8081/internal/orders \
CALLER_USER_ID=staff-42 \
LLM_MODEL=qwen2.5:3b \
go run . -question 'ORD-104288 现在是什么状态？'
```

示例不会自动执行退款等写操作。生产接入时，订单服务应基于可信身份重新校验资源权限。
