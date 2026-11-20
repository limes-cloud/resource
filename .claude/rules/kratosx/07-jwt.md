# JWT 认证 (library/jwt)

> 包路径：`github.com/limes-cloud/kratosx/library/jwt`

## 概述

JWT Token 管理，支持 Token 签发、刷新、黑名单、唯一设备登录。基于 Redis 存储 Token 状态。

## 配置

```yaml
jwt:
  enableGrpc: false       # gRPC 是否启用 JWT
  header: Authorization   # Token Header 名称
  secret: "your-secret"   # 签名密钥
  expire: 24h             # Token 过期时间
  renewal: 12h            # Token 续期时间
  unique: false           # 是否启用唯一设备登录
  uniqueKey: "uid"        # Claims 中用于标识用户的字段
  redis: "cache"          # 使用的 Redis 实例名称
  whitelist:              # 免验证白名单
    /api/v1/login: true
    /api/v1/register: true
    POST:/api/v1/captcha: true
```

## Jwt 接口

```go
type Jwt interface {
    // NewToken 签发 Token
    NewToken(claims jwtv5.Claims) (string, error)

    // ParseToken 解析 Token
    ParseToken(token string) (jwtv5.Claims, error)

    // IsWhitelist 是否为白名单路由
    IsWhitelist(path string, method string) bool

    // IsBlacklist Token 是否在黑名单
    IsBlacklist(token string) bool

    // AddBlacklist 将 Token 加入黑名单
    AddBlacklist(token string) error

    // SetToken 设置 Token 到上下文
    SetToken(ctx context.Context, token string) context.Context

    // GetToken 从上下文获取 Token
    GetToken(ctx context.Context) string

    // SetUniqueToken 设置唯一设备 Token
    SetUniqueToken(uniqueKey, token string) error

    // CompareUniqueToken 比较唯一设备 Token
    CompareUniqueToken(uniqueKey, token string) bool
}
```

## 使用方式

### 签发 Token

```go
jwt := ctx.JWT()
token, err := jwt.NewToken(jwtv5.MapClaims{
    "uid":  user.ID,
    "role": user.Role,
    "exp":  time.Now().Add(24 * time.Hour).Unix(),
})
```

### 获取当前 Token

```go
// 获取当前请求的 Token 字符串
token := ctx.Token()

// 或通过 JWT 实例获取
token := ctx.JWT().GetToken(ctx.Ctx())
```

### Token 黑名单（退出登录）

```go
// 将当前 Token 加入黑名单
err := ctx.JWT().AddBlacklist(ctx.Token())
```

### 唯一设备登录

```go
// 登录时设置唯一 Token（会使旧设备的 Token 失效）
err := ctx.JWT().SetUniqueToken(userID, newToken)
```

## 中间件行为

1. **白名单路由**：直接放行，不做 Token 验证
2. **Token 验证**：验证签名和过期时间
3. **黑名单检查**：Token 在黑名单中则拒绝
4. **唯一设备检查**（启用时）：Token 与 Redis 中存储的不一致则拒绝
