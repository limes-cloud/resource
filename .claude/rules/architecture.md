# 架构分层规则

> 完全对齐 limes-cloud/manager 的 DDD 分层，使用 kratosx 封装的 GORM（`ctx.DB()` 返回 `*gorm.DB`）。

## 分层职责

### core/ — 核心上下文与配置

- `conf.go`：业务配置结构体 `Conf`，全局单例 `var conf = &Conf{}`
- `context.go`：封装 `core.Context`（内嵌 `kratosx.Context`），提供 `DB()` / `Redis()` / `Logger()` / `Transaction()` / `Config()` / `Clone()` / `Token()` 等方法
- `MustContext(ctx context.Context, opts ...kratosx.ContextOptionFunc) Context`：将标准 context 转为 core.Context，支持 `kratosx.WithSkipDBHook()` 等选项
- 配置监听：通过 `configScanWatch()` 热更新业务配置

### types/ — 请求/响应 DTO

- 内部转换用类型，不暴露给 proto 层
- 仅包含 JSON tag，不含 DB tag
- nullable 查询参数使用指针（如 `*string` / `*uint32`）

### domain/ — 领域层（纯业务逻辑）

- **entity/**：领域实体，纯结构体，对应 DB 表，使用 `gorm` tag
- **repository/**：依赖接口（interface），包含仓储和外部客户端，由 infra 层实现
- **service/**：业务逻辑，通过构造函数注入 repository 接口

### infra/ — 基础设施层（接口的具体实现）

- **dbs/**：数据仓储实现（MySQL + Redis），单例模式 `sync.Once`
- **store/**：对象存储客户端实现（实现 repository 中对应的存储接口）

### app/ — 应用层（handler + 注册）

- 每个 service 一个文件，实现 proto 生成的 gRPC 接口
- `register.go`：自注册框架（`init() + register()` 模式）
- 请求/响应转换：proto ↔ types/entity

### middleware/ — 中间件

- 认证、日志、链路追踪等横切关注点
- 通过 `kratosx.WithMiddleware()` 在 main.go 中统一注册

## 依赖方向约束

```
app → domain/service → domain/repository (interface)
                          ↑
infra (implements) ────────┘
```

**严格规则：**

1. **domain 层不 import infra 包**：service 和 repository 接口只依赖 core、entity、types
2. **app 层同时 import domain 和 infra**：在 `NewXxx()` 中完成依赖注入
3. **infra 层可以 import domain**：实现 repository 接口时引用 entity 类型
4. **entity 不依赖任何层**：纯结构体，只包含字段和基本方法
5. **types 不依赖 entity**：两者独立，通过 value.Transform 转换
6. **所有 infra 层组件的接口统一定义在 domain/repository/**：infra 层不定义接口，只做实现。domain service 通过 repository 接口依赖 infra 能力，app 层完成注入。

## 文件组织规范

- 每个业务领域在各层都有对应文件（如 store / file / directory / app_config）
- 新增业务模块时，在各层新增对应文件，app 层使用 init() 自注册，无需修改已有文件