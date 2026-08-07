# 第 13 章 JWT 认证

## 场景

第 12 章我们把日志体系做好了。现在后台管理系统要上线，产品经理说："用户管理接口不能裸奔，得加上认证。"

你打开代码，发现所有接口都是公开的：

```go
// 任何人都能访问
r.GET("/api/v1/users", listUsers)
r.POST("/api/v1/users", createUser)
r.DELETE("/api/v1/users/:id", deleteUser)
```

问题很明显：
- 没有登录功能，用户怎么认证？
- 接口没有权限控制，谁都能删除用户
- 用户信息（密码）怎么安全存储？

Leader 说："用 JWT 做无状态认证，加上 RBAC 权限控制。"

本章解决四个问题：
1. 为什么用 JWT 而不是 Session？
2. 如何安全存储密码？
3. 如何实现登录和认证？
4. 如何做权限控制？

---

## 问题：当前 API 的 4 个安全痛点

1. **没有认证机制**
   - 所有接口都是公开的，任何人都能访问
   - 不知道请求是谁发的

2. **密码明文存储**
   - 用户密码直接存数据库
   - 数据库泄露 = 所有密码泄露

3. **没有权限控制**
   - 普通用户能删除其他用户
   - 没有角色区分

4. **Session 的问题**
   - 分布式环境下 Session 共享困难
   - 服务端要存储所有 Session
   - 跨域问题

---

## 13.1 认证方案对比

### 13.1.1 Session vs JWT

| 特性 | Session | JWT |
|------|---------|-----|
| 存储位置 | 服务端 | 客户端 |
| 状态 | 有状态 | 无状态 |
| 分布式 | 需要共享 Session | 天然支持 |
| 跨域 | 需要配置 CORS | 仍需配置 CORS，适合放在 Authorization Header |
| 性能 | 需要查 Session 存储 | 本地验证 |
| 注销 | 直接删除 Session | 需要黑名单 |

### 13.1.2 为什么选 JWT

- 无状态，天然支持分布式
- 客户端存储，服务端无压力
- 适合前后端分离和移动端调用
- 性能好（本地验证）

### 13.1.3 JWT 的缺点

- 无法主动注销（需要黑名单）
- Token 泄露风险
- 需要 HTTPS

---

## 13.2 JWT 原理

### 13.2.1 JWT 结构

```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJyb2xlIjoiYWRtaW4iLCJleHAiOjE3MDUzMjE2MDB9.signature
```

三部分：
- Header：算法和类型
- Payload：用户信息和过期时间，只是 Base64URL 编码，不是加密
- Signature：签名验证

![JWT 结构](./images/ch13-jwt-structure.svg)

### 13.2.2 签名算法

- HS256：HMAC 对称签名（简单，适合内部服务）
- RS256：RSA 非对称签名（适合跨服务或多方验签）

JWT 默认只做签名，不做加密。不要把密码、手机号、身份证号、内部权限明细等敏感信息放进 Payload。

### 13.2.3 验证流程

![JWT 认证流程](./images/ch13-jwt-flow.svg)

```
1. 客户端发送 Token（Authorization: Bearer xxx）
2. 服务端解析 Token
3. 验证签名
4. 验证过期时间
5. 提取用户信息
```

---

## 13.3 密码安全存储

> 代码：`example1-password/`

### 13.3.1 为什么不能明文存储

```go
// 错误：明文存储
user := &User{
    Name:     "Alice",
    Password: "123456",  // 危险！
}
```

### 13.3.2 bcrypt 加密

```go
import "golang.org/x/crypto/bcrypt"

// 加密密码
hashedPassword, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)

// 验证密码
err := bcrypt.CompareHashAndPassword(hashedPassword, []byte("123456"))
```

### 13.3.3 为什么用 bcrypt

- 慢速哈希（防止暴力破解）
- 自动加盐
- 可调整成本

### 13.3.4 运行示例

```bash
cd example1-password
go run main.go

# 输出：
# 原始密码: 123456
# 加密后: $2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
# 密码验证成功
```

---

## 13.4 JWT 库使用

> 代码：`example2-jwt/`

### 13.4.1 golang-jwt/jwt

```go
import "github.com/golang-jwt/jwt/v5"

// 创建 Token
secret := []byte(cfg.JWT.AccessSecret)
claims := jwt.MapClaims{
    "user_id": 1,
    "role":    "admin",
    "exp":     time.Now().Add(24 * time.Hour).Unix(),
}
token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
tokenString, err := token.SignedString(secret)

// 验证 Token
parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
    if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
        return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
    }
    return secret, nil
})
claims := parsedToken.Claims.(jwt.MapClaims)
```

### 13.4.2 自定义 Claims

```go
type UserClaims struct {
    UserID int    `json:"user_id"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}

// 创建
claims := &UserClaims{
    UserID: 1,
    Role:   "admin",
    RegisteredClaims: jwt.RegisteredClaims{
        ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
    },
}

// 验证
parsedToken, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
    if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
        return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
    }
    return secret, nil
}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
claims := parsedToken.Claims.(*UserClaims)
```

### 13.4.3 运行示例

```bash
cd example2-jwt
go run main.go

# 输出：
# 生成的 Token:
# eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
# 
# 解析结果:
#   UserID: 1
#   Role: admin
#   Valid: true
```

---

## 13.5 登录接口

> 代码：`example3-login/`

### 13.5.1 登录流程

```
1. 客户端发送用户名和密码
2. 服务端验证用户名和密码
3. 生成 JWT Token
4. 返回 Token
```

### 13.5.2 登录接口实现

```go
type LoginReq struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}

type LoginResp struct {
    Token string `json:"token"`
}

func loginHandler(c *gin.Context) {
    var req LoginReq
    if err := c.ShouldBindJSON(&req); err != nil {
        response.ValidationError(c, err)
        return
    }

    // 查询用户
    user, err := store.GetUserByUsername(req.Username)
    if err != nil {
        response.Unauthorized(c, "invalid credentials")
        return
    }

    // 验证密码
    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
        response.Unauthorized(c, "invalid credentials")
        return
    }

    // 生成 Token
    token, err := jwt.GenerateToken(user.ID, user.Role)
    if err != nil {
        response.InternalError(c, "failed to generate token")
        return
    }

    c.JSON(200, LoginResp{Token: token})
}
```

### 13.5.3 运行示例

```bash
cd example3-login
go run main.go

# 登录
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"123456"}'
# {"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."}
```

生产环境登录接口还要加防暴力破解：

- 按 IP 限制请求频率
- 按用户名记录连续失败次数
- 失败过多时短时间锁定账号或要求二次验证
- 登录失败只返回 `invalid credentials`，不要暴露“用户不存在”或“密码错误”
- 记录登录失败审计日志，便于告警和排查

---

## 13.6 认证中间件

> 代码：`example4-middleware/`

### 13.6.1 JWT 认证中间件

```go
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 获取 Token
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            response.Unauthorized(c, "missing authorization header")
            c.Abort()
            return
        }

        // 解析 Bearer Token
        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        if tokenString == authHeader {
            response.Unauthorized(c, "invalid authorization format")
            c.Abort()
            return
        }

        // 验证 Token
        claims, err := jwt.ParseToken(tokenString)
        if err != nil {
            response.Unauthorized(c, "invalid or expired token")
            c.Abort()
            return
        }

        // 设置用户信息
        c.Set("user_id", claims.UserID)
        c.Set("role", claims.Role)
        c.Next()
    }
}
```

### 13.6.2 使用中间件

```go
// 公开接口
r.POST("/api/v1/login", loginHandler)

// 需要认证的接口
api := r.Group("/api/v1")
api.Use(AuthMiddleware())
{
    api.GET("/users", listUsers)
    api.POST("/users", createUser)
}
```

### 13.6.3 运行示例

```bash
cd example4-middleware
go run main.go

# 未认证访问
curl http://localhost:8080/api/v1/users
# {"code":10002,"message":"missing authorization header"}

# 认证访问
curl http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
# {"code":0,"data":[...]}
```

---

## 13.7 RBAC 权限控制

> 代码：`example5-rbac/`

### 13.7.1 角色定义

```go
const (
    RoleAdmin  = "admin"
    RoleUser   = "user"
    RoleGuest  = "guest"
)
```

### 13.7.2 权限中间件

```go
func RequireRole(roles ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userRole, exists := c.Get("role")
        if !exists {
            response.Unauthorized(c, "unauthorized")
            c.Abort()
            return
        }

        role := userRole.(string)
        for _, r := range roles {
            if role == r {
                c.Next()
                return
            }
        }

        response.Forbidden(c, "forbidden")
        c.Abort()
    }
}
```

### 13.7.3 使用权限控制

```go
// 只有 admin 能访问
admin := api.Group("/admin")
admin.Use(RequireRole("admin"))
{
    admin.DELETE("/users/:id", deleteUser)
}

// admin 和 user 能访问
users := api.Group("/users")
users.Use(RequireRole("admin", "user"))
{
    users.GET("", listUsers)
    users.GET("/:id", getUser)
}
```

### 13.7.4 运行示例

```bash
cd example5-rbac
go run main.go

# admin 登录
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'

# admin 删除用户
curl -X DELETE http://localhost:8080/api/v1/admin/users/1 \
  -H "Authorization: Bearer <admin_token>"
# 204 No Content

# user 登录
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"123456"}'

# user 删除用户（被拒绝）
curl -X DELETE http://localhost:8080/api/v1/admin/users/1 \
  -H "Authorization: Bearer <user_token>"
# {"code":10003,"message":"forbidden"}
```

---

## 13.8 Token 刷新

> 代码：`example6-refresh/`

### 13.8.1 为什么需要 Token 刷新

- Access Token 过期时间短（15 分钟）
- Refresh Token 过期时间长（7 天）
- 用户体验和安全性的平衡

### 13.8.2 双 Token 机制

```go
type TokenPair struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
}

type RefreshClaims struct {
    UserID int `json:"user_id"`
    jwt.RegisteredClaims // 包含 jti、exp、iat、iss
}

// 登录时返回两个 Token
func loginHandler(c *gin.Context) {
    // ... 验证用户名密码

    accessToken, _ := jwt.GenerateAccessToken(user.ID, user.Role)
    refreshToken, _ := jwt.GenerateRefreshToken(user.ID)
    store.SaveRefreshToken(refreshToken.TokenID, user.ID, refreshToken.ExpiresAt)

    c.JSON(200, TokenPair{
        AccessToken:  accessToken,
        RefreshToken: refreshToken.Token,
    })
}
```

Refresh Token 必须包含 `jti`（Token ID），并在服务端保存状态。这样才能在刷新、退出登录、风控封禁时吊销指定 refresh token。

### 13.8.3 刷新接口

```go
type RefreshReq struct {
    RefreshToken string `json:"refresh_token" binding:"required"`
}

func refreshHandler(c *gin.Context) {
    var req RefreshReq
    if err := c.ShouldBindJSON(&req); err != nil {
        response.ValidationError(c, err)
        return
    }

    // 验证 Refresh Token
    claims, err := jwt.ParseRefreshToken(req.RefreshToken)
    if err != nil {
        response.Unauthorized(c, "invalid refresh token")
        return
    }

    // 生成新的 Access Token 和 Refresh Token
    accessToken, _ := jwt.GenerateAccessToken(claims.UserID, claims.Role)
    refreshToken, _ := jwt.GenerateRefreshToken(claims.UserID)

    // 轮换 Refresh Token：旧 jti 吊销，新 jti 生效
    if err := store.RotateRefreshToken(claims.ID, claims.UserID, refreshToken.TokenID, refreshToken.ExpiresAt); err != nil {
        response.Unauthorized(c, "invalid refresh token")
        return
    }

    c.JSON(200, gin.H{
        "access_token":  accessToken,
        "refresh_token": refreshToken.Token,
    })
}
```

### 13.8.4 运行示例

```bash
cd example6-refresh
go run main.go

# 登录
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"123456"}'
# {"access_token":"...","refresh_token":"..."}

# 刷新 Token
curl -X POST http://localhost:8080/api/v1/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"..."}'
# {"access_token":"...","refresh_token":"..."}
```

---

## 13.9 实战：给后台管理系统加认证

> 代码：`example7-admin-auth/`

### 13.9.1 项目结构

```
example7-admin-auth/
├── main.go
├── config/
│   ├── config.go
│   └── config.yaml
├── handler/
│   ├── user.go
│   └── auth.go          # 登录、刷新
├── model/
│   └── user.go
├── store/
│   └── memory.go
├── middleware/
│   └── auth.go          # JWT 认证和 RBAC 权限控制
├── jwt/
│   └── jwt.go           # JWT 工具
├── logger/
│   └── logger.go
└── response/
    └── response.go
```

### 13.9.2 配置文件

```yaml
app:
  name: admin-api
  env: dev
  version: 1.0.0

server:
  host: 0.0.0.0
  port: 8080

jwt:
  access_secret: ""      # 通过 JWT_ACCESS_SECRET 注入
  refresh_secret: ""     # 通过 JWT_REFRESH_SECRET 注入
  access_token_expire: 15m
  refresh_token_expire: 168h

log:
  level: info
  format: json
```

### 13.9.3 运行示例

```bash
cd example7-admin-auth
export JWT_ACCESS_SECRET="change-me-access-secret-32-bytes-min"
export JWT_REFRESH_SECRET="change-me-refresh-secret-32-bytes-min"
go run main.go

# 登录
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
# {"code":0,"message":"success","data":{"access_token":"...","refresh_token":"..."}}

# 创建用户（需要 admin 权限）
curl -X POST http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"username":"newuser","email":"newuser@example.com","password":"password123","role":"user"}'
# {"code":0,"message":"created","data":{"id":4,"username":"newuser","email":"newuser@example.com","role":"user"}}

# 查询用户列表（需要认证）
curl http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer <access_token>"
# {"code":0,"message":"success","data":{"users":[...],"total":4}}

# 删除用户（需要 admin 权限）
curl -X DELETE http://localhost:8080/api/v1/users/2 \
  -H "Authorization: Bearer <access_token>"
# 204 No Content

# 退出登录（吊销 refresh token）
curl -X POST http://localhost:8080/api/v1/logout \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<refresh_token>"}'
# 204 No Content
```

---

## 13.10 原理：JWT 的安全问题

### 13.10.1 Token 泄露

**问题**：Token 被窃取，攻击者可以冒充用户

**解决方案**：
- 使用 HTTPS
- Access Token 过期时间短
- 敏感操作需要二次验证

### 13.10.2 无法主动注销

**问题**：JWT 无法主动注销（无状态）

**解决方案**：
- Access Token 黑名单（Redis）
- 短过期时间
- Refresh Token 使用 `jti`，服务端保存状态并轮换
- 退出登录时吊销 Refresh Token

### 13.10.3 签名算法攻击

**问题**：`alg: none` 攻击

**解决方案**：
```go
// 验证算法
if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
    return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
}
```

### 13.10.4 密钥管理

**问题**：密钥泄露 = 所有 Token 可伪造

**解决方案**：
- 密钥通过环境变量、K8s Secret 或 Vault 注入
- Access Token 和 Refresh Token 使用不同密钥
- 启动时拒绝空密钥、默认密钥和过短密钥
- 定期轮换密钥
- 使用非对称签名（RS256）

---

## 13.11 最佳实践

1. **密码用 bcrypt 加密**：永远不要明文存储
2. **Access Token 过期时间短**：15 分钟
3. **Refresh Token 过期时间长**：7 天
4. **使用 HTTPS**：防止 Token 被窃取
5. **验证签名算法**：防止 `alg: none` 攻击
6. **密钥不要硬编码**：用环境变量、K8s Secret 或 Vault
7. **敏感操作加权限控制**：RBAC
8. **Refresh Token 要轮换**：保存 `jti`，刷新后吊销旧 Token
9. **登录接口要限流**：按 IP 和账号维度限制失败次数
10. **浏览器跨域要配置 CORS**：允许 `Authorization` Header

---

## 13.12 排障

### 13.12.1 Token 无效

**问题**：`invalid token`

**原因**：
- Token 过期
- 签名错误
- Token 格式错误
- Access/Refresh 密钥用错
- Refresh Token 已被轮换或吊销

**解决**：
- 检查 Token 是否过期
- 检查密钥是否正确
- 检查 Token 格式（Bearer xxx）
- Refresh 失败时检查服务端 `jti` 状态

### 13.12.2 密码验证失败

**问题**：`invalid credentials`

**原因**：
- 密码错误
- 密码没有正确加密

**解决**：
- 检查密码是否正确
- 检查 bcrypt 加密是否正确

### 13.12.3 权限不足

**问题**：`forbidden`

**原因**：
- 用户角色不对
- 权限中间件配置错误

**解决**：
- 检查用户角色
- 检查 `RequireRole()` 参数

---

## 13.13 面试题

**Q1：JWT 和 Session 的区别？**

A：
- JWT：无状态，客户端存储，天然支持分布式
- Session：有状态，服务端存储，需要共享

**Q2：为什么密码要用 bcrypt？**

A：
- 慢速哈希，防止暴力破解
- 自动加盐，防止彩虹表攻击
- 可调整成本

**Q3：JWT 无法注销怎么办？**

A：
- Token 黑名单（Redis）
- 短过期时间
- Refresh Token 保存 `jti`，刷新时轮换，退出时吊销

**Q4：Access Token 和 Refresh Token 的区别？**

A：
- Access Token：过期时间短（15 分钟），用于访问资源
- Refresh Token：过期时间长（7 天），用于刷新 Access Token，需要服务端保存状态并支持轮换/吊销

**Q5：如何防止 `alg: none` 攻击？**

A：
- 验证签名算法
- 使用白名单（只允许 HS256/RS256）

---

## 13.14 小结

本章从 API 安全的痛点出发，用 JWT 实现了完整的认证和权限控制：

1. **认证方案对比**：Session vs JWT
2. **JWT 原理**：结构、签名、验证
3. **密码安全**：bcrypt 加密
4. **登录接口**：用户名密码验证
5. **认证中间件**：JWT 验证
6. **RBAC 权限**：角色权限控制
7. **Token 刷新**：双 Token 机制
8. **实战**：给后台管理系统加认证

**核心原则：**

> 认证是安全的第一道防线，JWT 让认证变得简单、高效、可扩展。

下一章我们将学习文件上传服务，让服务支持文件处理。

---

## 参考资料

> 本章基于 **Go 1.25**、Gin v1.12.0、golang-jwt/jwt/v5 v5.3.1。API 与默认行为随版本变化，以对应版本官方文档为准。

- JWT 规范 RFC 7519：https://www.rfc-editor.org/rfc/rfc7519
- JWT 最佳实践 RFC 8725：https://www.rfc-editor.org/rfc/rfc8725
- golang-jwt：https://pkg.go.dev/github.com/golang-jwt/jwt/v5
- Gin 官方文档：https://gin-gonic.com/docs/
- net/http：https://pkg.go.dev/net/http
