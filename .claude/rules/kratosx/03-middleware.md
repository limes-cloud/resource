# 中间件 (middleware)

> 包路径：`github.com/limes-cloud/kratosx/middleware`

## 概述

kratosx 内置了一套中间件链，按顺序自动组装。未启用的中间件返回 `nil` 自动跳过。

## 中间件执行顺序

```
请求 → Recovery → Timeout → RateLimit → Metrics → Metadata → Signature → Tracing → Logging → Validator → IP → JWT → JwtToken → JwtBlack → JwtUnique → Handler
```

## 各中间件说明

### 1. Recovery（始终启用）

捕获 panic，返回结构化错误。支持通过 `ctx.Exit(err)` 主动中断请求。

```go
// 在 handler 中主动中断
ctx.Exit(errors.New(400, "BAD_REQUEST", "invalid input"))
```

### 2. Timeout（始终启用）

按路由粒度控制超时，支持全局超时和特殊路由超时：

```yaml
server:
  http:
    timeout: 30s
    timeoutSpecial:
      "POST:/api/v1/example": 120s
  grpc:
    timeout: 30s
    timeoutSpecial:
      "GRPC:/example.ExampleService/Create": 60s
```

### 3. RateLimit（按配置启用）

基于 kratos 内置 BBR 限流算法的自适应限流：

```yaml
rateLimit: true
```

### 4. Metrics（按配置启用）

OpenTelemetry 指标采集（请求计数 + 延迟直方图）：

```yaml
metrics: true
```

### 5. Metadata（始终启用）

kratos 元数据传播中间件，自动透传 `x-md-*` 前缀的 Header。

### 6. Signature（按配置启用）

API 签名验证中间件，支持白名单：

```yaml
signature:
  enable: true
  ak: "app-key"
  sk: "secret-key"
  expire: 10s
  whitelist:
    /api/v1/health: true
```

### 7. Tracing（始终启用）

OpenTelemetry 分布式链路追踪：

```yaml
tracing:
  httpEndpoint: "localhost:4318"
  sampleRatio: 1.0
  timeout: 10s
  insecure: true
```

不配置 `httpEndpoint` 时仅生成 trace/span ID，不上报。

### 8. Logging（按配置启用）

请求日志中间件，记录请求/响应信息：

```yaml
logging:
  enable: true
  whitelist:
    /api/v1/health: true          # 按路径跳过
    GET:/api/v1/health: true      # 按方法+路径跳过
```

### 9. Validator（始终启用）

自动调用 proto 生成的 `Validate()` 方法校验请求参数。

自定义错误处理：

```go
kratosx.New(
    kratosx.WithValidateErrHook(func(ctx context.Context, err error) error {
        c := kratosx.MustContext(ctx)
        c.Logger().Warnw("msg", "params validate error", "err", err)
        return errors.ParamsError()
    }),
)
```

### 10. IP（始终启用）

从请求中提取客户端 IP（优先 `x-real-ip` Header），存入 context。

### 11. JWT 系列（按配置启用）

分为三个子中间件：

| 中间件 | 功能 |
|--------|------|
| `Jwt` | JWT Token 签名验证（白名单路由跳过） |
| `JwtBlack` | Token 黑名单检查 |
| `JwtUnique` | 唯一设备登录校验 |

```yaml
jwt:
  header: Authorization
  secret: "your-jwt-secret"
  expire: 24h
  renewal: 12h
  unique: true
  uniqueKey: "uid"
  redis: "cache"
  whitelist:
    /api/v1/login: true
    /api/v1/register: true
```

## 追加自定义中间件

```go
kratosx.New(
    kratosx.WithMiddleware(
        myCustomMiddleware(),
        anotherMiddleware(),
    ),
)
```

自定义中间件会追加在内置中间件之后执行。
