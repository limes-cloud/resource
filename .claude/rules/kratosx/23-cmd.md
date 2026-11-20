# CLI 工具 (cmd)

> 包路径：`github.com/limes-cloud/kratosx/cmd`

## 概述

kratosx 提供了三个 CLI 工具：

1. **kratosx** — 项目脚手架和 Proto 代码生成
2. **protoc-gen-go-errorsx** — 自定义错误码 Proto 插件
3. **protoc-gen-go-httpx** — 增强版 HTTP 网关 Proto 插件

---

## kratosx CLI

### 安装

```bash
go install github.com/limes-cloud/kratosx/cmd/kratosx@latest
```

### 命令列表

| 命令 | 说明 |
|------|------|
| `kratosx new <project>` | 创建新项目（从模板仓库克隆） |
| `kratosx proto add <name>` | 新增 proto 文件 |
| `kratosx proto client <path>` | 生成 proto 客户端代码 |
| `kratosx run` | 运行项目 |
| `kratosx upgrade` | 升级 kratosx CLI |
| `kratosx change` | 查看版本变更日志 |

### 创建项目

```bash
kratosx new my-service
# 从模板仓库克隆项目骨架
```

### 生成 Proto 代码

```bash
# 生成服务代码（gRPC + HTTP + OpenAPI）
kratosx proto client api/example/proto/example_service.proto

# 生成错误码
kratosx proto client api/errors/proto/errors.proto
```

### 运行项目

```bash
kratosx run
# 等效于 go run main.go
```

---

## protoc-gen-go-errorsx

### 安装

```bash
go install github.com/limes-cloud/kratosx/cmd/protoc-gen-go-errorsx@latest
```

### 功能

根据 proto 中的枚举定义生成错误函数：

```protobuf
// errors.proto
enum ErrorReason {
  option (errors.default_code) = 500;

  ParamsError   = 0 [(errors.message)="参数错误"];
  SystemError   = 1 [(errors.message)="系统错误"];
  GetError      = 4 [(errors.message)="获取数据失败"];
}
```

生成代码：

```go
// errors_errors.pb.go（自动生成）

func ParamsError(args ...string) *errors.Error {
    msg := "参数错误"
    if len(args) > 0 {
        msg += ": " + strings.Join(args, ", ")
    }
    return errors.New(500, "ParamsError", msg)
}

func IsParamsError(err error) bool {
    return errors.Reason(err) == "ParamsError"
}
```

### Proto 注解

| 注解 | 位置 | 说明 |
|------|------|------|
| `(errors.default_code)` | enum option | 默认 HTTP 状态码 |
| `(errors.code)` | enum value option | 单个错误的 HTTP 状态码 |
| `(errors.message)` | enum value option | 错误消息 |

### errors.proto 依赖

```protobuf
import "errorsx/errors.proto";
```

需要在项目的 `third_party/errorsx/` 目录放置 `errors.proto`：

```protobuf
syntax = "proto3";
package errors;

import "google/protobuf/descriptor.proto";

option go_package = "github.com/limes-cloud/kratosx/third_party/errorsx;errors";

extend google.protobuf.EnumOptions {
    int32 default_code = 1108;
}

extend google.protobuf.EnumValueOptions {
    int32 code = 1109;
    string message = 1110;
}
```

---

## protoc-gen-go-httpx

### 安装

```bash
go install github.com/limes-cloud/kratosx/cmd/protoc-gen-go-httpx@latest
```

### 功能

生成增强版 HTTP 网关代码，相比 kratos 官方的 `protoc-gen-go-http`：

- 支持 `BindQuery`（自动绑定 URL Query 参数）
- 支持 `BindVars`（路径参数绑定）
- 生成 HTTP 客户端实现

### 使用

```protobuf
service ExampleService {
  rpc CreateExample(CreateExampleRequest) returns (CreateExampleResponse) {
    option (google.api.http) = {
      post: "/api/v1/example"
      body: "*"
    };
  }

  rpc GetExample(GetExampleRequest) returns (GetExampleResponse) {
    option (google.api.http) = {
      get: "/api/v1/example/{id}"
    };
  }
}
```

生成的 Handler 会自动：
1. `ctx.Bind(&in)` — 绑定 Body
2. `ctx.BindQuery(&in)` — 绑定 Query
3. `ctx.BindVars(&in)` — 绑定路径变量

---

## Proto 生成完整命令

```bash
protoc --proto_path=. \
    --proto_path=./third_party \
    --go_out=paths=source_relative:. \
    --go-grpc_out=paths=source_relative:. \
    --go-httpx_out=paths=source_relative:. \
    --go-errorsx_out=paths=source_relative:. \
    api/example/proto/example_service.proto
```

或使用 kratosx CLI 简化：

```bash
kratosx proto client api/example/proto/example_service.proto
```
