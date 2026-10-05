# Agent 控制循环回放

示例使用记录式模型轨迹，让开发者在不请求真实模型的情况下重复调试 Agent 调度器。订单查询会调用配置的内部只读接口。

```bash
ORDER_SERVICE_URL=http://localhost:8081/internal/orders \
CALLER_USER_ID=staff-42 \
go run . -question 'ORD-104288 目前是什么状态？'
```

真实模型的工具调用协议见同阶段第 69 章。生产使用仍需限制轮数、调用预算和后端权限。
