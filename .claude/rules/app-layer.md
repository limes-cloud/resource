# App 层规则

> 参考 limes-cloud/manager/internal/app/ 的 handler + register 模式。

## 目录结构

```
internal/app/
├── register.go           # 自注册框架（全局变量 + register 函数）
├── example.go            # 示例资源 handler
└── store.go              # 存储配置 handler
```

## Handler 结构体

每个 handler 必须遵循以下模式：

```go
type Example struct {
    example.UnimplementedExampleServiceServer  // 嵌入 proto 生成的基类
    srv *service.Example                         // 注入 domain service
}
```

## 构造函数与依赖注入

在 `NewXxx()` 中完成 service → repo 的注入链：

```go
func NewExample() *Example {
    return &Example{
        srv: service.NewExample(
            dbs.NewExample(),       // infra/dbs/ 仓储实现
        ),
    }
}
```

## 自注册模式

每个 handler 通过 `init() + register()` 自动注册 gRPC 和 HTTP 服务：

```go
func init() {
    register(func(hs *http.Server, gs *grpc.Server) {
        srv := NewExample()
        example.RegisterExampleServiceHTTPServer(hs, srv)
        example.RegisterExampleServiceServer(gs, srv)
    })
}
```

**register.go 框架：**

```go
package app

import (
    "github.com/go-kratos/kratos/v2/transport/grpc"
    "github.com/go-kratos/kratos/v2/transport/http"
)

type registerFunc func(hs *http.Server, gs *grpc.Server)

var registerList []registerFunc

func register(fn registerFunc) {
    registerList = append(registerList, fn)
}

func Register(hs *http.Server, gs *grpc.Server) {
    for _, registry := range registerList {
        registry(hs, gs)
    }
}
```

## Handler 方法实现

每个方法遵循三步模式：

### 1. 上下文转换

```go
ctx := core.MustContext(c)   // c 是 context.Context，转为 core.Context
```

### 2. 请求转换（proto → types/entity）

```go
var in types.GetExampleRequest
if err := value.Transform(req, &in); err != nil {
    ctx.Logger().Errorw("msg", "req transform error", "err", err)
    return nil, errors.TransformError()
}
```

或直接转换为 entity：

```go
var in entity.Example
if err := value.Transform(req, &in); err != nil {
    ctx.Logger().Errorw("msg", "req transform error", "err", err)
    return nil, errors.TransformError()
}
```

### 3. 调用 service + 响应转换

```go
// 调用 service
result, err := h.srv.GetExample(ctx, &in)
if err != nil {
    return nil, err
}

// 响应转换（entity → proto）
reply := example.GetExampleResponse{}
if err := value.Transform(result, &reply); err != nil {
    ctx.Logger().Errorw("msg", "resp transform error", "err", err)
    return nil, errors.TransformError()
}
return &reply, nil
```

### 简单场景

对于只传 ID 的简单请求，可以跳过 Transform：

```go
func (h *Example) CreateExample(c context.Context, req *example.CreateExampleRequest) (*example.CreateExampleResponse, error) {
    ctx := core.MustContext(c)
    id, err := h.srv.Create(ctx, &types.CreateExampleRequest{
        App:    req.App,
        Name:   req.Name,
        Status: req.Status,
        Remark: req.Remark,
    })
    if err != nil {
        return nil, err
    }
    return &example.CreateExampleResponse{Id: id}, nil
}
```

### 字段访问与可选透传

- 直接用 `req.Xxx`，**不用** `req.GetXxx()`。
- `optional`（指针）字段把指针**原样透传**给 types/service，**禁止** `if v := req.GetX(); v != "" { ... }` / `if d := req.X; d != 0 { ... }` 这种用零值判断模拟"可选"——proto `optional` + 指针已经表达了存在性（见 proto-and-api.md 规则 11）。只有 proto 表达不了的深校验（如名称文法 `ValidateName`）才在 handler 里 `if req.X != nil` 后调用。
- 响应里的凭证字段脱敏在此层做（如 `toStore` 把 sk 回成占位符）；repo / service 不负责脱敏（见 infra-layer.md）。

## 新增 Handler 的步骤

1. 在 `api/xxx/proto/` 定义 proto 文件
2. 运行 protoc 生成代码
3. 在 `internal/app/` 新建 `xxx.go` 文件
4. 定义 handler 结构体，嵌入 `UnimplementedXxxServer`
5. 实现 `NewXxx()` 构造函数，注入 service 和 dbs
6. 编写 `init()` + `register()` 自注册
7. 实现各 RPC 方法（三步模式：上下文转换 → 请求转换 → 调用 + 响应转换）
