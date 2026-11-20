# 应用启动与初始化规则

## main.go 入口

参考 limes-cloud/manager 的入口模式，使用 kratos 框架启动：

```go
package main

import (
    "context"

    "github.com/go-kratos/kratos/v2/log"
    "github.com/limes-cloud/resource/internal/app"
    "github.com/limes-cloud/resource/internal/core"
    // 其他必要 import
)

func main() {
    srv := core.InitApp(
        core.WithRegistrarServer(app.Register),
        core.WithValidateErrHook(func(ctx context.Context, err error) error {
            c := core.MustContext(ctx)
            c.Logger().Warnw("msg", "params validate error", "err", err)
            return errors.ParamsError()
        }),
    )

    if err := srv.Run(); err != nil {
        log.Fatal(err)
    }
}
```

**关键选项：**

| 选项 | 用途 |
|---|---|
| `WithRegistrarServer(app.Register)` | 注册所有 gRPC/HTTP handler |
| `WithValidateErrHook()` | 参数校验错误的统一处理 |
| `WithMiddleware()` | 全局中间件注册 |

## core/ 初始化

### InitApp

封装 kratos 应用初始化，设置默认配置源和配置热更新：

```go
// core/app.go
func InitApp(opts ...kratos.Option) *kratos.App {
    defOpts := []kratos.Option{
        kratos.WithConfigSource(configSource()),
        kratos.WithConfigWatch(configScanWatch),
    }
    return kratos.New(append(defOpts, opts...)...)
}
```

### 配置源

支持两种配置源，优先使用远程配置中心：

```go
// core/conf.go
func configSource() config.Source {
    host := os.Getenv("CONF_HOST")
    token := os.Getenv("CONF_TOKEN")
    name := os.Getenv("APP_NAME")
    if host != "" && token != "" && name != "" {
        return remoteConfig.New(host, name, token)  // 远程配置中心
    }
    return file.NewSource("conf/")  // 本地配置文件
}
```

### 配置热更新

通过 `ScanWatch` 监听配置变更，自动刷新业务配置：

```go
func configScanWatch(watch config.Watcher) {
    watch("business", func(value config.Value) {
        if err := value.Scan(&conf); err != nil {
            panic(err)
        }
    })
}
```

**规则：** 业务配置结构体 `Conf` 必须是全局单例，通过 `core.Context.Config()` 访问，禁止在 service 中直接读取配置文件。

## core.Context 扩展

在 kratosx.Context 基础上扩展业务方法：

```go
type Context struct {
    kratosx.Context
}

// Transaction 包装事务，将内部 kratosx.Context 自动转换为 core.Context 传入闭包
func (c Context) Transaction(fn func(ctx Context) error, name ...string) error

// Config 获取业务配置单例
func (Context) Config() *Conf

// Clone 克隆上下文（异步任务用，不受原 context cancel/timeout 影响）
func (c Context) Clone() Context {
    return MustContext(context.WithoutCancel(c.Context))
}

// Token 从 metadata 中读取请求 token（x-md-global-token）
func (c Context) Token() string

// MustContext 将标准 context 转为 core.Context，支持 kratosx 选项
// 例：core.MustContext(ctx, kratosx.WithSkipDBHook())
func MustContext(ctx context.Context, opts ...kratosx.ContextOptionFunc) Context
```

**常用场景：**

| 方法 | 场景 |
|---|---|
| `ctx.Clone()` | 启动异步任务（后台清理等），脱离请求超时 |
| `ctx.Token()` | 在 service 层透传 token 给下游服务 |
| `MustContext(ctx, kratosx.WithSkipDBHook())` | 缓存预热等初始化场景，跳过租户/权限 DB Hook |

## 配置文件结构

`conf/config.yaml` 遵循 kratosx 的标准结构，业务配置放在 `business` 键下：

```yaml
server:
  http:
    host: 127.0.0.1
    port: 7020
    timeout: 10s
  grpc:
    host: 127.0.0.1
    port: 8020
    timeout: 10s

database:
  - enable: true
    name: resource
    drive: mysql
    connect:
      host: 127.0.0.1
      port: 3306
      dbName: resource
      username: root
      password: root
    config:
      maxLifetime: 2h
      maxOpenConn: 20
      maxIdleConn: 10

redis:
  - enable: true
    name: cache
    host: 127.0.0.1:6379

business:
  upload:
    gc_interval: 5m
    max_size: 100
```
