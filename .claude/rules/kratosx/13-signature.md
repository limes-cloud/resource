# 签名验证 (library/signature)

> 包路径：`github.com/limes-cloud/kratosx/library/signature`

## 概述

HMAC-SHA256 API 签名验证，用于服务间通信的身份验证和防篡改。支持服务端和客户端双向签名。

## 配置

```yaml
signature:
  enable: true
  ak: "app-key"          # Access Key
  sk: "secret-key"       # Secret Key
  expire: 10s            # 签名有效期
  whitelist:             # 免签名白名单
    /api/v1/health: true
    GET:/api/v1/version: true
```

## 签名算法

```
签名内容 = request_body + "|" + timestamp + "|" + ak
signature = HMAC-SHA256(签名内容, sk)
```

请求时通过 Header 传递：
- `x-md-sign-time`: 时间戳
- `x-md-sign-token`: 签名值

## Signature 接口

```go
type Signature interface {
    // Generate 生成签名
    Generate(content []byte) (timestamp int64, sign string, err error)

    // Verify 验证签名
    Verify(content []byte, sign string, ts int64) error

    // IsWhitelist 是否为白名单路由
    IsWhitelist(name string) bool

    // Server 服务端验签中间件
    Server() middleware.Middleware

    // Client 客户端签名中间件
    Client(conf *config.Signature) middleware.Middleware
}
```

## 使用方式

### 手动签名/验签

```go
sig := signature.Instance()

// 生成签名
ts, sign, err := sig.Generate([]byte(requestBody))

// 验证签名
err := sig.Verify([]byte(requestBody), sign, ts)
```

### 中间件自动签名

服务端：在 kratosx 启动时自动注册（配置 `signature.enable: true`）。

客户端：在 gRPC 客户端配置中设置：

```yaml
client:
  - server: target-service
    signature:
      enable: true
      ak: "my-app"
      sk: "my-secret"
```
