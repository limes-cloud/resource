# 配置管理 (config)

> 包路径：`github.com/limes-cloud/kratosx/config`

## 概述

封装 kratos 配置管理，支持多配置源、配置热更新、mapstructure 自动解码。

## Config 接口

```go
type Config interface {
    SetAppInfo(id, name, version string)
    Load() error
    Scan(v any) error
    Value(key string) Value
    Watch(key string, o WatchHandleFunc)
    ScanWatch(key string, o WatchHandleFunc)
    Close() error
    App() *App
}
```

## 配置文件结构

默认从 `conf/` 目录加载 YAML 配置：

```yaml
# conf/config.yaml
name: "resource"
version: "v1.0.0"

server:
  count: 1
  http:
    host: 0.0.0.0
    port: 7010
    timeout: 30s
    formatResponse: true
    cors:
      allowOrigins: ["*"]
      allowMethods: ["GET", "POST", "PUT", "DELETE"]
      allowHeaders: ["*"]
    marshal:
      emitUnpopulated: true
      useProtoNames: true
  grpc:
    host: 0.0.0.0
    port: 8010
    timeout: 30s

database:
  - enable: true
    name: resource
    drive: postgres
    connect:
      host: 127.0.0.1
      port: 5432
      dbName: resource
      username: postgres
      password: postgres
    config:
      maxLifetime: 2h
      maxOpenConn: 20
      maxIdleConn: 10

redis:
  - enable: true
    name: cache
    host: 127.0.0.1:6379

logger:
  level: 0
  output: ["stdout"]
  enCoder: json

jwt:
  header: Authorization
  secret: your-secret
  expire: 24h
  renewal: 12h
  whitelist:
    /api/v1/login: true

business:
  cache:
    ttl: 10m
```

## 配置热更新

### 方式一：Watch（仅监听变更）

```go
kratosx.New(
    kratosx.WithConfigWatch(func(w config.Watcher) {
        w("business", func(value config.Value) {
            var conf BusinessConf
            if err := value.Scan(&conf); err != nil {
                // 处理错误
                return
            }
            // 应用新配置
        })
    }),
)
```

### 方式二：ScanWatch（初始加载 + 监听变更）

```go
kratosx.New(
    kratosx.WithConfigWatch(func(w config.Watcher) {
        // ScanWatch = 立即 Scan 一次 + Watch
        w.ScanWatch("business", func(value config.Value) {
            value.Scan(&conf)
        })
    }),
)
```

## 配置类型结构

### App 主结构

| 字段 | 类型 | 说明 |
|------|------|------|
| `Name` | `string` | 应用名称 |
| `Version` | `string` | 版本号 |
| `Server` | `struct` | 服务配置（HTTP/gRPC/TLS/Registry） |
| `Database` | `[]*Database` | 数据库配置列表 |
| `Redis` | `[]*Redis` | Redis 配置列表 |
| `Logger` | `*Logger` | 日志配置 |
| `JWT` | `*JWT` | JWT 配置 |
| `Pool` | `*Pool` | 协程池配置 |
| `Email` | `*Email` | 邮件配置 |
| `Tracing` | `*Tracing` | 链路追踪配置 |
| `Client` | `[]*Client` | gRPC 客户端配置 |
| `Logging` | `*Logging` | 请求日志配置 |
| `Signature` | `*Signature` | 签名配置 |
| `Captcha` | `map[string]*Captcha` | 验证码配置 |
| `Loader` | `map[string]string` | 文件加载器配置 |
| `Prometheus` | `[]*Prometheus` | Prometheus 指标配置 |
| `RateLimit` | `bool` | 是否开启限流 |
| `Metrics` | `bool` | 是否开启指标 |

### Database 配置

```yaml
database:
  - enable: true
    name: default       # 实例名称
    drive: postgres     # 驱动：postgres / mysql / sqlserver
    autoCreate: false   # 是否自动创建数据库
    connect:
      host: 127.0.0.1
      port: 5432
      dbName: resource
      username: postgres
      password: postgres
    config:
      logLevel: 1           # 0=Silent 1=Error 2=Warn 3=Info
      slowThreshold: 200ms  # 慢查询阈值
      maxLifetime: 2h
      maxOpenConn: 20
      maxIdleConn: 10
      prepareStmt: false
      transformError:       # GORM 错误转换
        enable: true
      initializer:          # 数据初始化
        enable: false
        path: deploy/data.sql
```

### Redis 配置

```yaml
redis:
  - enable: true
    name: cache
    host: 127.0.0.1:6379
    username: ""
    password: ""
    db: 0
    poolSize: 10
    minIdleConns: 5
    dialTimeout: 5s
    readTimeout: 3s
    writeTimeout: 3s
```

### HTTP/gRPC 服务配置

```yaml
server:
  count: 1              # 服务实例数（多实例自动分配端口）
  http:
    host: 0.0.0.0
    port: 7010
    timeout: 30s
    timeoutSpecial:     # 按路由设置特殊超时
      "POST:/api/v1/example": 120s
    formatResponse: true  # 是否统一格式化响应
    cors:
      allowOrigins: ["*"]
    marshal:
      emitUnpopulated: true
      useProtoNames: true
    pprof:              # pprof 调试
      query: token
      secret: your-secret
    webServerDir: ""    # 静态文件目录
  grpc:
    host: 0.0.0.0
    port: 8010
    timeout: 30s
    timeoutSpecial:
      "GRPC:/example.ExampleService/Create": 60s
  registry: "consul://127.0.0.1:8500"  # 服务注册（可选）
  tls:
    pem: "..."
    key: "..."
```

## Value 接口

`config.Value` 封装了 kratos 的 Value，增强了 Scan 能力：

```go
type Value interface {
    kratosConfig.Value
    Scan(dst any) error  // 支持 mapstructure 自动解码
}
```

支持 `time.Duration` 字符串自动转换和逗号分隔的字符串自动转为切片。

## 获取配置

```go
// 通过 Context 获取
conf := ctx.Config()
app := conf.App()

// 获取任意配置值
val := conf.Value("business.cache.ttl")

// 全局单例
conf := config.Instance()
```
