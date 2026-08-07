# 第25章 gRPC实战

## 场景

电商系统拆成微服务后,订单服务需要高频调用用户、库存、支付等服务。早期用 REST + JSON,问题逐渐暴露:

> "下单一次要调 6 个下游,每个调用光 JSON 序列化就占了 3ms,高峰期网关 P99 飙到 800ms。"

Leader 说:

> "内部服务间调用换成 gRPC。"

本章解决五个问题:

1. gRPC 和 REST 有什么区别,什么时候用?
2. 怎么定义服务、生成代码、写 server/client?
3. 四种 RPC 类型(一元、服务端流、客户端流、双向流)怎么选?
4. 拦截器(中间件)怎么做鉴权、日志、指标?
5. 超时/取消怎么在服务间传播,怎么避免雪崩?

> 代码:`25-grpc/`,包含 4 个可独立运行的 example。
>
> 前置:已安装 `protoc`(libprotoc 28)、`protoc-gen-go`(v1.36.x)、`protoc-gen-go-grpc`(v1.6.x):
>
> ```bash
> go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
> go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.2
> ```

---

## 问题

REST/JSON 在内部服务间调用的痛点:

- **序列化慢、体积大**:JSON 是文本协议,字段名重复传输,数字按字符串编码
- **没有强类型契约**:接口靠文档/Swagger 维护,字段改了调用方编译期发现不了
- **HTTP/1.1 队头阻塞**:同一连接上请求必须串行,只能靠多开连接扛并发
- **流式支持弱**:SSE/WebSocket 都是补丁式方案,没有统一的请求/响应流抽象
- **代码生成缺失**:客户端、服务端、模型都要手写

gRPC 的解法:

- **Protocol Buffers**:二进制编码 + 强类型 `.proto` 契约,一次定义多语言生成
- **HTTP/2**:多路复用、头部压缩、原生双向流
- **四种调用模式**:一元、服务端流、客户端流、双向流
- **拦截器链**:统一处理鉴权、日志、重试、指标

但 gRPC 不是银弹:浏览器不能直接调(要 grpc-web/网关),调试不如 curl 直观,对外 API 仍常以 REST 为主。**内部服务间通信用 gRPC,对外开放用 REST/网关**,是常见组合。

---

## 实现

### 25.1 第一个 gRPC 服务

> 代码:`25-grpc/example1-unary/`、`25-grpc/proto/user/user.proto`

#### 25.1.1 定义 proto

```proto
syntax = "proto3";

package user;

option go_package = "github.com/go-book/grpc/proto/user;userpb";

service UserService {
  rpc GetUser(GetUserReq) returns (User);
}

message GetUserReq {
  int64 id = 1;
}

message User {
  int64 id = 1;
  string username = 2;
  string email = 3;
  int32 status = 4; // 0:禁用 1:启用
}
```

关键点:

- `option go_package` **必须写**,指定生成的 Go 导入路径和包名(分号后是包别名)
- 字段后的 `= 1`、`= 2` 是 **字段编号(tag)**,用于二进制编码,**上线后绝不能改、不能复用**
- `proto3` 没有 `required`/`optional`(标量字段有零值语义),要表达"未设置"用 `optional` 或包装类型

生成代码:

```bash
protoc \
  --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  proto/user/user.proto
```

会生成两个文件:

- `user.pb.go`:消息结构体、序列化代码
- `user_grpc.pb.go`:服务端接口(`UserServiceServer`)和客户端(`UserServiceClient`)

**生成的代码要提交到仓库**,不要要求每个构建环境都装 protoc。

#### 25.1.2 服务端

```go
type UserServer struct {
	userpb.UnimplementedUserServiceServer // 前向兼容:新增 RPC 时旧代码能编译
}

func (s *UserServer) GetUser(ctx context.Context, req *userpb.GetUserReq) (*userpb.User, error) {
	return &userpb.User{
		Id:       req.GetId(),
		Username: "alice",
		Email:    "alice@example.com",
		Status:   1,
	}, nil
}

func runServer(addr string) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("监听失败: %v", err)
	}
	// 明文仅用于本地演示;生产必须用 TLS / mTLS
	s := grpc.NewServer()
	userpb.RegisterUserServiceServer(s, &UserServer{})
	log.Printf("gRPC server 监听 %s", addr)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Serve 失败: %v", err)
	}
}
```

> **必须内嵌 `UnimplementedUserServiceServer`**。它给所有 RPC 一个默认的"未实现"返回。以后 `.proto` 加新方法,没实现的服务端不会因为缺方法而编译失败,而是返回 `codes.Unimplemented`。这是 gRPC 的前向兼容约定。

#### 25.1.3 客户端

```go
func runClient(addr string) {
	// grpc.NewClient 是当前推荐写法;旧的 grpc.Dial 已废弃。
	// 默认异步重连、连接就绪探测;本地演示用 insecure 凭证
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer conn.Close()

	c := userpb.NewUserServiceClient(conn)

	// 每次 RPC 都要带超时:超时由客户端说了算,到点服务端 ctx 会被取消
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := c.GetUser(ctx, &userpb.GetUserReq{Id: 1})
	if err != nil {
		log.Fatalf("GetUser 失败: %v", err)
	}
	fmt.Printf("用户: id=%d username=%s email=%s status=%d\n",
		resp.GetId(), resp.GetUsername(), resp.GetEmail(), resp.GetStatus())
}
```

运行:

```bash
# 终端 A
go run ./example1-unary -role=server
# 终端 B
go run ./example1-unary -role=client
# 用户: id=1 username=alice email=alice@example.com status=1
```

> 注意客户端用的是 `GetId()`、`GetUsername()` 这类 getter,而不是直接访问字段。getter 会处理 nil receiver,**链式访问深层嵌套消息时不会因为中间层是 nil 而 panic**,这是 protobuf Go 代码生成的约定。

### 25.2 服务端流式 RPC

> 代码:`25-grpc/example2-stream/`、`proto/order/order.proto`

一元 RPC 是一次请求一次响应。当一个请求对应一批数据(比如查某用户的全部订单、实时增量推送),用**服务端流式**:客户端发一个请求,服务端 `Send` 多条消息,最后关闭流。

```proto
service OrderService {
  // stream 关键字在 returns 一侧 => 服务端流
  rpc ListOrders(ListOrdersReq) returns (stream Order);
}
```

服务端:

```go
func (s *OrderServer) ListOrders(req *orderpb.ListOrdersReq, stream orderpb.OrderService_ListOrdersServer) error {
	orders := []*orderpb.Order{ /* ... */ }
	for _, o := range orders {
		// 每条消息 Send 一次;真实场景里可以边查边发,不必一次性把结果攒进内存
		if err := stream.Send(o); err != nil {
			return err // 客户端已取消,Send 会返回错误
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil // 返回 nil 表示流正常结束
}
```

客户端:

```go
stream, err := c.ListOrders(ctx, &orderpb.ListOrdersReq{UserId: 1})
if err != nil {
	log.Fatalf("ListOrders 失败: %v", err)
}

for {
	o, err := stream.Recv()
	if err == io.EOF {
		break // 服务端关闭流,读取结束
	}
	if err != nil {
		log.Fatalf("Recv 失败: %v", err)
	}
	fmt.Printf("收到订单: %s 金额=%d分\n", o.GetOrderNo(), o.GetAmount())
}
```

运行:

```bash
$ go run ./example2-stream -role=client
收到订单: NO-20241001-001  金额=19900分  状态=paid
收到订单: NO-20241002-002  金额=25900分  状态=shipped
收到订单: NO-20241003-003  金额=6900分   状态=created
=== 共 3 个订单,合计 52700 分
```

**四种调用模式对比**:

| 模式 | proto 写法 | 适用场景 |
|---|---|---|
| 一元 |`rpc F(Req) returns (Resp)`| 普通请求-响应,最常用 |
| 服务端流 |`rpc F(Req) returns (stream Resp)`| 一批结果、实时推送、大结果集分页 |
| 客户端流 |`rpc F(stream Req) returns (Resp)`| 客户端上传一批数据后拿一个汇总(如批量统计) |
| 双向流 |`rpc F(stream Req) returns (stream Resp)`| 全双工:聊天、实时协作、多路事件 |

选型原则:**先用一元,确有批量/流式需求再升级**。流式不是免费的,错误处理、流控、重试都比一元复杂。

### 25.3 拦截器:鉴权与日志

> 代码:`25-grpc/example3-interceptor/`

gRPC 的拦截器(interceptor)就是它的"中间件"。一元调用走 `UnaryServerInterceptor`,流式调用走 `StreamServerInterceptor`。多个拦截器用 `grpc.ChainUnaryInterceptor` 串起来,按注册顺序执行(类似洋葱模型)。

下面这个例子把**日志**和**鉴权**拆成两个拦截器:

```go
const tokenKey = "authorization"

// 1) 鉴权:从 metadata(gRPC 的 header)里取 token 校验
func authUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok || len(md.Get(tokenKey)) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing token")
		}
		if md.Get(tokenKey)[0] != "Bearer valid-token" {
			return nil, status.Errorf(codes.PermissionDenied, "invalid token")
		}
		ctx = context.WithValue(ctx, ctxUserKey{}, "alice") // 把身份塞进 ctx
		return handler(ctx, req)
	}
}

// 2) 日志:记录方法、状态码、耗时
func loggingUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		log.Printf("method=%s code=%s duration=%s", info.FullMethod, status.Code(err), time.Since(start))
		return resp, err
	}
}

// 注册:注意顺序,日志在外层先执行后返回,鉴权在内层
s := server.NewAllServer(
	grpc.ChainUnaryInterceptor(
		loggingUnaryServerInterceptor(),
		authUnaryServerInterceptor(),
	),
)
```

客户端通过 `metadata` 透传认证信息:

```go
md := metadata.Pairs(tokenKey, tok)
ctx = metadata.NewOutgoingContext(ctx, md)
resp, err := c.GetUser(ctx, &userpb.GetUserReq{Id: 1})
```

运行(用不同 token 演示):

```bash
$ go run ./example3-interceptor -role=client
token="Bearer valid-token"   -> 用户=alice
token="Bearer wrong-token"   -> code=PermissionDenied msg=invalid token
token=""                     -> code=PermissionDenied ...
# 服务端日志:
# method=/user.UserService/GetUser code=OK duration=2.4µs
# method=/user.UserService/GetUser code=PermissionDenied duration=3.5µs
```

**要点**:

- 错误必须用 `status.New(codes.Xxx, msg)` 构造,客户端才能通过 `status.Code(err)` 拿到标准化状态码。不要直接返回 `errors.New`,那样客户端只会收到 `codes.Unknown`
- 常用状态码:`Unauthenticated`(未认证)、`PermissionDenied`(无权限)、`NotFound`、`InvalidArgument`、`DeadlineExceeded`、`Internal`、`Unavailable`(可重试)
- 生产里鉴权逻辑一般抽成独立 interceptor,日志/指标/recover 同理;开源的有 `grpc-ecosystem/go-grpc-middleware`

### 25.4 超时与取消传播

> 代码:`25-grpc/example4-deadline/`

这是微服务稳定性最关键的一节。gRPC 基于 HTTP/2,**客户端设的 deadline 会随请求自动传播给服务端**,服务端的 `ctx.Done()` 会在超时时被触发。

```go
func (s *InventoryServer) GetStock(ctx context.Context, req *invpb.GetStockReq) (*invpb.Stock, error) {
	if req.GetSlow() {
		select {
		case <-time.After(500 * time.Millisecond): // 模拟慢查询
		case <-ctx.Done():
			return nil, ctx.Err() // 客户端已超时,立刻停下,别再白干活
		}
	}
	return &invpb.Stock{SkuId: req.GetSkuId(), Available: 42}, nil
}
```

客户端给 200ms 超时,而服务端要处理 500ms:

```bash
# 服务端即时返回
$ go run ./example4-deadline -client -slow=false -timeout=200ms
耗时=21ms 库存=42

# 服务端慢处理,客户端 200ms 就放弃
$ go run ./example4-deadline -client -slow=true -timeout=200ms
耗时=201ms code=DeadlineExceeded msg=context deadline exceeded
=> 触发 deadline,服务端会收到 ctx 取消并停止慢操作
```

**为什么这能防雪崩**:

- 上游请求已经超时返回用户了,下游如果还在傻乎乎查库,就是在浪费连接、CPU、数据库连接
- deadline 一路透传:网关(2s) → 订单(1.5s) → 库存(1s),每层比上游更短,到点全部一起停
- 服务端监听 `ctx.Done()`,及时中断慢操作、释放资源

实战中务必:

- **每个客户端调用都设超时**,不要用永远不超时的 `context.Background()`
- 服务端所有可能阻塞的操作(DB、RPC、IO)都要接收并响应 `ctx`
- 不要把已取消的 ctx 传到另一个独立的异步任务里(那个任务该用 `context.WithoutCancel` 或新建 ctx)

---

## 原理

### 25.5.1 gRPC 为什么快

gRPC 的性能优势来自三层:

1. **Protobuf 二进制编码**:字段编号(tag)代替字段名,整数用 varint 变长编码,数字按二进制存储。同样的数据,Protobuf 体积通常是 JSON 的 1/3 ~ 1/10,解析也更快(无需词法分析)
2. **HTTP/2 多路复用**:一个 TCP 连接上可以并发多个 stream,通过 stream ID 区分,彻底解决 HTTP/1.1 的队头阻塞;同时支持头部压缩(HPACK)
3. **生成的代码是手写级别的**:序列化/反序列化代码由 protoc 生成,直接操作字节,没有反射开销

但不要神化。对大多数内部服务,网络延迟和业务逻辑才是大头,序列化的差异在 benchmark 里明显,在端到端延迟里往往只占零点几毫秒。真正让团队选 gRPC 的通常是**强类型契约和代码生成**,而不是那点性能。

### 25.5.2 HTTP/2 上的帧与流

一次 gRPC 调用在 HTTP/2 上的结构:

- 请求是一个 HTTP/2 POST,`Content-Type: application/grpc`,path 是 `/包.服务/方法`(如 `/user.UserService/GetUser`)
- 请求体和响应体都被切成带 gRPC 前缀的 frame(5 字节头:1 字节是否压缩 + 4 字节长度,后跟 protobuf 编码的消息)
- 一元调用:各一帧;流式调用:多帧 + 最后 trailers 传状态码
- 状态码不在 HTTP body 里,而在响应 trailers(`grpc-status`、`grpc-message`)

理解这个有什么用?排查问题时你会知道:

- gRPC 的 deadline 怎么传到服务端 → HTTP/2 的 `grpc-timeout` 头
- 为什么代理/网关要支持 HTTP/2 → 不支持就没法多路复用、没法流式
- 为什么 gRPC 不能直接用浏览器调 → 浏览器 fetch/XHR 不暴露 HTTP/2 trailer,需要 grpc-web 转译

### 25.5.3 拦截器执行顺序

```
请求进入
  │
  ▼
logging interceptor (前置:记 start)
  │
  ▼
auth interceptor (前置:校验 token)
  │
  ▼
真正的 handler (业务逻辑)
  │
  ▼
auth interceptor (后置:通常没有)
  │
  ▼
logging interceptor (后置:记耗时、状态码)
  │
  ▼
响应返回
```

`ChainUnaryInterceptor` 按注册顺序形成调用链,和 HTTP 中间件、gRPC 拦截器的洋葱模型一致。**放最外层的最先开始、最后结束**,所以日志、recover、panic 捕获一般放最外层。

### 25.5.4 服务端如何处理取消

客户端超时或主动取消时,HTTP/2 会发 RST_STREAM。gRPC 服务端的 `ctx` 随之被 cancel,所有监听 `ctx.Done()` 的操作(DB 查询、下游 RPC)都会收到信号。这就是为什么**业务代码必须把 ctx 传到底**——ctx 不传,取消就断在某一层,下游还在白干。

---

## 最佳实践

### 25.6.1 契约设计

- **字段编号一旦上线永不复用**。删字段时保留编号 `reserved 5;`,防止后人误用
- **字段只加不删、只放宽不收紧**。新增字段用新编号,旧客户端会忽略不认识的字段(前向兼容);枚举不要删值、不要改数字
- **给 message 和字段加注释**。`.proto` 是团队契约,注释就是接口文档
- **包名用域名倒序或团队约定**,如 `order.v1`,为以后多版本共存留空间
- 时间用 `google.protobuf.Timestamp`,金额用整数(分)或 `string`,**不要用 float 传钱**

### 25.6.2 错误处理

```go
// 推荐:带状态码,客户端可程序化判断
return nil, status.Error(codes.NotFound, "user not found")

// 带详情(可以传结构化错误信息)
st, _ := status.New(codes.InvalidArgument, "invalid request").
    WithDetails(&errdetails.BadRequest{...})
return nil, st.Err()
```

- 业务错误用合适的 gRPC code,不要统统返回 `Internal`
- `error` 不要直接透传数据库错误给客户端(会泄露表结构)
- 客户端统一用 `status.Code(err)` 判断,`codes.Unavailable`、`DeadlineExceeded` 这类可重试

### 25.6.3 客户端

- **连接是长连接,全局复用**。`grpc.NewClient` 返回的 `*grpc.ClientConn` 是并发安全的,不要每次请求都新建
- 配置连接参数:

```go
grpc.NewClient(addr,
    grpc.WithTransportCredentials(creds),
    grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
    grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(10*1024*1024)), // 默认 4MB,大消息要调
)
```

- 默认消息大小限制 **4MB**,传大文件要调 `MaxCallRecvMsgSize`/`MaxCallSendMsgSize`,但更好的做法是大对象走对象存储、gRPC 只传引用
- 生产环境**必须用 TLS**,内部用 mTLS 做服务间身份认证;不要把 `insecure` 带出本地开发

### 25.6.4 可观测性

拦截器是统一埋点的好地方:

- **日志**:记 method、code、duration(如 example3)
- **指标**:用 Prometheus 拦截器统计 QPS、错误率、延迟分布
- **链路追踪**:用 OpenTelemetry gRPC instrumentation,自动在 metadata 里传播 trace context
- **panic recover**:加一个 recover 拦截器,防止单个 handler panic 搞垮整个进程

---

## 排障

### 25.7.1 `connection refused` / 一直重连

**现象**:客户端报 `connection refused` 或日志反复 `connection error`。

**排查**:

1. 服务端是否真的在监听?`lsof -i :50051` 或 `ss -lntp`
2. 地址端口对不对?容器里 `localhost` 指的是容器自己,要用宿主地址或 service 名
3. 是不是用了 `grpc.Dial`(已废弃)且没加阻塞选项?用 `grpc.NewClient`,它默认异步连接,第一次调用时才真正建连;要确认连通用 `conn.Connect()` + 健康检查

### 25.7.2 `rpc error: code = Unavailable desc = ...`

`Unavailable` 通常是**临时性**的:连接断了、服务端重启、负载均衡后端不可用。客户端可以重试,但要配上限和退避。注意区分:

- `Unavailable`/`DeadlineExceeded`/`ResourceExhausted` → 通常可重试
- `InvalidArgument`/`NotFound`/`PermissionDenied` → 不可重试,重试也没用

### 25.7.3 `context deadline exceeded` 但服务端没收到

- 检查客户端超时是不是设太短(比如 10ms 连握手都不够)
- 检查中间是否有代理/LB 吞掉了 gRPC 的 HTTP/2 流量(某些默认配置只转发 HTTP/1.1)
- 开 gRPC debug:`GRPC_GO_LOG_VERBOSITY_LEVEL=99 GRPC_GO_LOG_SEVERITY_LEVEL=info ./client`

### 25.7.4 流式调用卡住不返回

- 服务端记得最后 `return nil` 才会发 EOF;客户端 `Recv()` 收到 `io.EOF` 才结束
- 客户端超时要覆盖整个流的生命周期,而不是只建连时
- 检查流控:消费者不 `Recv`,服务端 `Send` 最终会阻塞(HTTP/2 flow control)

### 25.7.5 `received message larger than max`

gRPC 默认单消息 4MB。要么调大 `MaxCallRecvMsgSize`,更推荐**把大消息拆分或改成对象存储引用**——把 50MB 的图片塞进 RPC 是反模式。

---

## 面试题

**Q1:gRPC 相比 REST/JSON 有什么优势?什么场景不适合?**

A:
- 优势:Protobuf 体积小解析快、HTTP/2 多路复用、强类型契约+多语言代码生成、原生流式和超时传播
- 不适合:对外开放给浏览器/第三方的 API(浏览器支持差、调试不直观)、需要人类可读缓存的边缘场景、团队没有 schema 管理能力时反而增加沟通成本

**Q2:Protobuf 为什么前向/后向兼容?**

A:序列化时只写**字段编号+类型+值**,解析时按编号匹配。新增字段旧客户端不认识就跳过(前向兼容);删掉的字段编号 `reserved` 住,旧客户端发来的旧字段新客户端忽略(后向兼容)。关键纪律是编号永不复用、类型不随意变更。

**Q3:gRPC 的四种调用模式分别什么时候用?**

A:
- 一元:绝大多数请求-响应
- 服务端流:一次请求对应一批数据/实时推送(订单列表、监控增量)
- 客户端流:上传一批数据拿汇总(批量上报、聚合统计)
- 双向流:全双工交互(聊天、实时协作)

**Q4:gRPC 的超时是怎么传播的?为什么能防雪崩?**

A:客户端 deadline 通过 HTTP/2 的 `grpc-timeout` 头传给服务端,服务端 ctx 到点自动 cancel,所有监听 `ctx.Done()` 的 DB/RPC 一起停。这样上游超时后下游不会继续白耗资源;每层设比上游更短的超时,能把故障限制在局部。关键是 ctx 要一路传到底。

**Q5:为什么必须内嵌 `UnimplementedXxxServer`?**

A:它给所有 RPC 提供默认的 `Unimplemented` 返回。这样以后 `.proto` 新增方法时,还没实现新方法的旧服务端依然能编译、能启动,只是新 RPC 返回未实现——保证了服务端的前向兼容和滚动升级期间不崩。

**Q6:gRPC 拦截器和 HTTP 中间件有什么异同?**

A:模型一样,都是洋葱模型的责任链。区别是 gRPC 分一元和流式两套拦截器;一元拦截器能直接拿到 req/resp 和方法信息,流式拦截器要包装 stream。用途相同:鉴权、日志、指标、recover、tracing。

---

## 小结

本章从 REST/JSON 的内部调用痛点出发,实现了 gRPC 的核心能力:

1. **基础**:用 `.proto` 定义服务、`protoc` 生成代码、实现 server/client
2. **流式**:服务端流式 RPC 处理一批数据,了解四种调用模式
3. **拦截器**:用链式拦截器统一做鉴权、日志,掌握标准错误码
4. **超时传播**:deadline 沿 HTTP/2 传播到服务端,是防雪崩的第一道关
5. **原理**:Protobuf 编码、HTTP/2 多路复用、拦截器执行顺序
6. **最佳实践与排障**:契约兼容性、错误码、长连接复用、常见故障排查

**核心原则:**

> gRPC 用强类型契约和 HTTP/2 让服务间通信更快、更可靠,但它的价值更多在工程(契约、代码生成、超时传播)而非单纯性能。生产里:ctx 传到底、错误用 status code、连接全局复用、拦截器统一横切逻辑。

下一章我们深入 Protocol Buffers,讲清字段编号、兼容规则、版本演进和高级类型。

---

## 参考资料

> 本章基于 **Go 1.25**、**google.golang.org/grpc v1.83.0**、**google.golang.org/protobuf v1.36.11**,`protoc` libprotoc 28。gRPC API 和生成代码在不同小版本间可能有差异,以官方文档为准。

- gRPC Go 官方文档:https://grpc.io/docs/languages/go/
- gRPC Go 示例:https://github.com/grpc/grpc-go/tree/master/examples
- Protocol Buffers 官方指南:https://protobuf.dev/programming-guides/proto3/
- gRPC 概念(HTTP/2、deadline、interceptor):https://grpc.io/docs/what-is-grpc/core-concepts/
- gRPC 状态码:https://grpc.github.io/grpc/core/md_doc_statuscodes.html
- grpc-ecosystem 中间件集:https://github.com/grpc-ecosystem/go-grpc-middleware
