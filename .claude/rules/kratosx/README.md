# Kratosx 框架使用手册

> 源码路径：`GolandProjects/go-platform/framework/kratosx`
> 模块：`github.com/limes-cloud/kratosx`
> 版本：Go 1.24.6 | kratos v2.8.4

## 框架简介

kratosx 是基于 [go-kratos/kratos](https://github.com/go-kratos/kratos) 的增强型微服务框架，提供：

- 开箱即用的应用初始化（一行代码启动 gRPC + HTTP 服务）
- 丰富的上下文封装（DB、Redis、JWT、Logger、Pool 等一站式访问）
- 内置中间件链（Recovery → Timeout → 限流 → 链路 → 日志 → 校验 → JWT）
- 配置热更新 + 多配置源
- proto-first 开发流程 + 自定义代码生成插件

---

## 目录

### 核心模块

| 文档 | 说明 |
|------|------|
| [01-core.md](01-core.md) | **核心入口** — App 初始化、Context 接口、Option 配置 |
| [02-config.md](02-config.md) | **配置管理** — 配置加载、热更新、类型定义 |
| [03-middleware.md](03-middleware.md) | **中间件** — 内置中间件链及各中间件使用 |

### 数据层

| 文档 | 说明 |
|------|------|
| [04-database.md](04-database.md) | **数据库** — GORM 封装、多实例、事务传播 |
| [05-redis.md](05-redis.md) | **Redis** — 多实例管理、连接配置 |

### 基础服务

| 文档 | 说明 |
|------|------|
| [06-logger.md](06-logger.md) | **日志** — zap 结构化日志、多输出、日志切割 |
| [07-jwt.md](07-jwt.md) | **JWT 认证** — Token 签发/验证/黑名单/唯一设备 |
| [08-pool.md](08-pool.md) | **协程池** — 异步任务、WaitRunner 批量编排 |
| [09-request.md](09-request.md) | **HTTP 客户端** — resty 封装、重试、日志 |
| [10-client.md](10-client.md) | **gRPC 客户端** — 服务发现、负载均衡、签名 |
| [11-email.md](11-email.md) | **邮件** — SMTP 发送、模板渲染 |
| [12-captcha.md](12-captcha.md) | **验证码** — 图形验证码生成/校验 |
| [13-signature.md](13-signature.md) | **签名验证** — HMAC-SHA256 API 签名 |
| [14-lock.md](14-lock.md) | **分布式锁** — Redis 锁实现 |
| [15-env.md](15-env.md) | **环境管理** — .env 加载、环境判断 |
| [16-loader.md](16-loader.md) | **文件加载器** — 启动时预加载文件到内存 |
| [17-prometheus.md](17-prometheus.md) | **Prometheus** — 指标定义与采集 |
| [18-tasker.md](18-tasker.md) | **生命周期** — 启动/停止回调管理 |
| [19-registry.md](19-registry.md) | **服务注册** — Consul 服务注册/发现 |

### 工具包

| 文档 | 说明 |
|------|------|
| [20-pkg.md](20-pkg.md) | **工具包** — value/Transform、crypto、tree、ua |
| [21-model.md](21-model.md) | **模型** — 分页查询、模型钩子 |
| [22-server.md](22-server.md) | **服务器工具** — CORS、响应格式化、静态文件 |
| [23-cmd.md](23-cmd.md) | **CLI 工具** — kratosx 脚手架、Proto 插件 |
| [24-pprof.md](24-pprof.md) | **Pprof** — 性能分析端点 |
| [25-ip.md](25-ip.md) | **IP 获取** — 客户端真实 IP 提取 |

---

## 快速上手示例

### main.go

```go
package main

import (
    "context"

    "github.com/limes-cloud/kratosx"
    "github.com/limes-cloud/kratosx/config"
    "github.com/limes-cloud/resource/api/errors"
    "github.com/limes-cloud/resource/internal/app"
)

func main() {
    application := kratosx.New(
        kratosx.WithRegistrarServer(app.Register),
        kratosx.WithConfigWatch(configWatch),
        kratosx.WithValidateErrHook(func(ctx context.Context, err error) error {
            c := kratosx.MustContext(ctx)
            c.Logger().Warnw("msg", "params validate error", "err", err)
            return errors.ParamsError()
        }),
    )

    if err := application.App().Run(); err != nil {
        panic(err)
    }
}

func configWatch(w config.Watcher) {
    w("business", func(value config.Value) {
        // 热更新业务配置
    })
}
```

### Handler（App 层）

```go
package app

import (
    "context"

    "github.com/go-kratos/kratos/v2/transport/grpc"
    "github.com/go-kratos/kratos/v2/transport/http"
    "github.com/limes-cloud/kratosx"
    "github.com/limes-cloud/kratosx/pkg/value"

    pb "github.com/limes-cloud/resource/api/example"
    "github.com/limes-cloud/resource/api/errors"
    "github.com/limes-cloud/resource/internal/domain/service"
    "github.com/limes-cloud/resource/internal/infra/dbs"
    "github.com/limes-cloud/resource/internal/types"
)

type Example struct {
    pb.UnimplementedExampleServiceServer
    srv *service.Example
}

func init() {
    register(func(hs *http.Server, gs *grpc.Server) {
        srv := &Example{
            srv: service.NewExample(dbs.NewExample()),
        }
        pb.RegisterExampleServiceHTTPServer(hs, srv)
        pb.RegisterExampleServiceServer(gs, srv)
    })
}

func (h *Example) CreateExample(c context.Context, req *pb.CreateExampleRequest) (*pb.CreateExampleResponse, error) {
    ctx := kratosx.MustContext(c)

    var in types.CreateExampleRequest
    if err := value.Transform(req, &in); err != nil {
        return nil, errors.TransformError()
    }

    id, err := h.srv.Create(ctx, &in)
    if err != nil {
        return nil, err
    }
    return &pb.CreateExampleResponse{Id: id}, nil
}
```

---

## 与本项目的适配说明

本项目（resource）使用 kratosx 封装的 **GORM**：`ctx.DB()` 返回 `*gorm.DB`，由 `library/db` 自动初始化。数据库为 **MySQL**（`conf.yaml` 配 `drive: mysql`）。

| kratosx 能力 | 本项目用法 |
|---|---|
| `ctx.DB()` → `*gorm.DB` | 直接使用，CRUD 走 `First()` / `Find()` / `Create()` / `Updates()` / `Delete()`，Upsert 用 `clause.OnConflict`，表达式更新用 `gorm.Expr`（**禁用 `Save`**，见 infra-layer.md） |
| `library/db` 自动初始化 | 使用 |
| `gormtranserror` | 使用（`internal/domain/service/errors.go` 封装 `isNotFound` / `isDuplicate`） |
| 主键与公共字段 | entity 嵌入 kratosx 的 `model.BaseModel`（`github.com/limes-cloud/kratosx/model`，与 manager 一致）：`Id uint32 primaryKey;autoIncrement`，`CreatedAt` / `UpdatedAt` 为 `int64`（Unix 秒），GORM 自动填充 |
| `model/page` 分页 | 不使用，列表分页在 repository 中手动封装 |

保留使用的 kratosx 能力：
- Context 封装（Logger、Config、Redis、Pool、Clone、Transaction 自定义实现）
- 中间件链（Recovery、Timeout、JWT、Validator、Tracing）
- 配置管理（热更新、ScanWatch）
- Proto 代码生成（kratosx proto client）
- 协程池（Pool + WaitRunner）
- HTTP 请求客户端（Request）
- 生命周期管理（Tasker）