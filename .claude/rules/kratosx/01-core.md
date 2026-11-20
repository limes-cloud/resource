# Kratosx 核心模块

> 包路径：`github.com/limes-cloud/kratosx`

## 概述

kratosx 是基于 [go-kratos/kratos](https://github.com/go-kratos/kratos) 的增强框架，提供了开箱即用的应用初始化、上下文封装、中间件编排、组件库等能力。

## 快速开始

```go
package main

import (
    "github.com/limes-cloud/kratosx"
    "github.com/limes-cloud/kratosx/config"
)

func main() {
    app := kratosx.New(
        kratosx.WithRegistrarServer(Register),
        kratosx.WithConfigWatch(configWatch),
    )

    if err := app.App().Run(); err != nil {
        panic(err)
    }
}

func Register(hs *http.Server, gs *grpc.Server) {
    // 注册 gRPC/HTTP 服务
}

func configWatch(w config.Watcher) {
    w("business", func(value config.Value) {
        // 监听业务配置变更
    })
}
```

## App 初始化流程

`kratosx.New()` 执行以下步骤：

1. **加载环境变量** — 调用 `env.Load()` 加载 `.env` 文件
2. **初始化默认配置** — 默认从环境变量（`APP_` 前缀）和 `conf/` 目录加载配置
3. **应用用户选项** — 执行所有 `Option` 函数
4. **加载配置** — 调用 `config.Load()` 解析配置到 `App` 结构体
5. **执行配置监听** — 若配置了 `WithConfigWatch`，注册配置变更监听
6. **初始化组件库** — 调用 `library.Init()` 初始化所有基础组件（DB、Redis、Logger 等）

调用 `app.App()` 后：

7. **组装 kratos 默认参数** — ID、Name、Version、生命周期钩子
8. **创建中间件链** — Recovery → Timeout → RateLimit → Metrics → Metadata → Signature → Tracing → Logging → Validator → IP → JWT
9. **创建 gRPC/HTTP Server** — 根据配置生成服务实例
10. **注册用户服务** — 调用 `regSrvFn(hs, gs)`
11. **注册服务发现**（可选）— 若配置了 Registry

## Option 列表

| Option | 说明 |
|--------|------|
| `WithRegistrarServer(fn)` | **必须** 注册 gRPC/HTTP handler 的函数 |
| `WithConfigSource(source...)` | 自定义配置源（替换默认的 env + file） |
| `WithConfigWatch(watch)` | 注册配置监听函数 |
| `WithMiddleware(mds...)` | 追加自定义中间件（在内置中间件之后） |
| `WithValidateErrHook(hook)` | 参数校验错误的自定义处理函数 |
| `WithHttpServerOptions(opts...)` | 追加 HTTP Server 选项 |
| `WithGrpcServerOptions(opts...)` | 追加 gRPC Server 选项 |
| `WithLibraryOptions(opts...)` | 组件库初始化选项 |
| `WithKratosOptions(opts...)` | 底层 kratos 选项 |
| `WithUnitTest()` | 标记为单元测试模式 |
| `WithHookSystemConfig(fn)` | 劫持修改系统配置 |

## Context 接口

`kratosx.Context` 是框架的核心上下文，通过 `kratosx.MustContext(ctx)` 从标准 `context.Context` 转换获得。

### 创建方式

```go
// handler 入口处转换
ctx := kratosx.MustContext(c)

// 带 trace 信息
ctx := kratosx.MustContext(c, kratosx.WithTrace(traceID, spanID))

// 跳过 DB 钩子
ctx := kratosx.MustContext(c, kratosx.WithSkipDBHook())
```

### 主要方法

| 方法 | 返回值 | 说明 |
|------|--------|------|
| `DB(name...)` | `*gorm.DB` | 获取数据库实例（支持事务传播） |
| `Redis(name...)` | `*redis.Client` | 获取 Redis 客户端 |
| `Transaction(fn, name...)` | `error` | 开启嵌套事务 |
| `Logger()` | `logger.Logger` | 获取结构化日志器 |
| `Config()` | `config.Config` | 获取配置对象 |
| `Pool()` | `pool.Pool` | 获取全局协程池 |
| `JWT()` | `jwt.Jwt` | 获取 JWT 服务 |
| `Token()` | `string` | 获取当前请求的 JWT Token |
| `ClientIP()` | `string` | 获取客户端 IP |
| `Email()` | `email.Email` | 获取邮件服务 |
| `Captcha()` | `captcha.PCaptcha` | 获取验证码服务 |
| `Loader(name)` | `[]byte` | 获取文件加载器内容 |
| `Request(opts...)` | `request.Request` | 创建 HTTP 请求客户端 |
| `GrpcConn(srvName)` | `(*grpc.ClientConn, error)` | 获取 gRPC 连接 |
| `GetMetadata(key)` | `string` | 获取元数据 |
| `SetMetadata(key, value)` | - | 设置元数据 |
| `UserAgent()` | `ua.UserAgent` | 解析 User-Agent |
| `Env()` | `env.Env` | 获取环境信息 |
| `Clone()` | `Context` | 克隆上下文（异步任务用） |
| `Exit(p)` | - | 中断当前请求 |

### 生命周期钩子

```go
ctx.BeforeStart("task-name", func() { /* 服务启动前执行 */ })
ctx.AfterStart("task-name", func() { /* 服务启动后执行 */ })
ctx.BeforeStop("task-name", func() { /* 服务停止前执行 */ })
ctx.AfterStop("task-name", func() { /* 服务停止后执行 */ })
```

### 事务使用

```go
err := ctx.Transaction(func(ctx kratosx.Context) error {
    // ctx.DB() 在此闭包内自动使用事务连接
    if err := ctx.DB().Create(&entity).Error; err != nil {
        return err
    }
    return ctx.DB().Delete(&other).Error
})
```

### Clone 用于异步任务

```go
// 异步任务不受原始请求 context 的 cancel/timeout 影响
asyncCtx := ctx.Clone()
go func() {
    // 使用 asyncCtx 而非原始 ctx
    asyncCtx.DB().Create(...)
}()
```
