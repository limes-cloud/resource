# Proto 与 API 规则

## api/ 独立 Go 模块

`api/` 目录使用独立的 `go.mod`，可被其他服务引用：

```
api/
├── example/
│   ├── proto/
│   │   ├── example.proto          # 消息定义
│   │   └── example_service.proto  # 服务定义
│   ├── example.pb.go
│   ├── example.pb.validate.go
│   ├── example_service.pb.go
│   ├── example_service_grpc.pb.go
│   ├── example_service_http.pb.go
│   └── openapi.yaml
├── store/
│   ├── proto/
│   │   ├── store.proto
│   │   └── store_service.proto
│   └── ...
├── errors/
│   ├── proto/
│   │   └── errors.proto               # 错误码定义
│   ├── errors.pb.go
│   └── errors_errors.pb.go            # 生成的错误函数
├── go.mod                              # 独立模块
└── go.sum
```

## Proto 文件规范

### package 命名规则

package 使用 `resource.api.<service>` 格式，以项目名为前缀：

| 文件路径 | package |
|---|---|
| `api/example/proto/xxx.proto` | `resource.api.example` |
| `api/store/proto/xxx.proto` | `resource.api.store` |
| `api/errors/proto/xxx.proto` | `resource.api.errors` |

规则：`resource.api.<服务名>`，不包含 `proto` 子目录层级（已禁用 `PACKAGE_DIRECTORY_MATCH` lint）。

### go_package 格式

使用项目内相对路径，**不含完整 module 前缀**（如 `github.com/limes-cloud/`）：

```protobuf
option go_package = "resource/api/example;example";
```

格式：`resource/api/<service>;<package_alias>`

### import 路径

import 使用相对于项目根目录的完整路径：

```protobuf
// 导入同服务的消息定义
import "api/example/proto/example.proto";

// 导入第三方 proto（通过 third_party/ 软链接解析）
import "google/api/annotations.proto";

// 导入项目内错误码扩展
import "errorsx/errors.proto";
```

### 消息定义（xxx.proto）

```protobuf
syntax = "proto3";

package resource.api.example;

option go_package = "resource/api/example;example";

import "validate/validate.proto";

message CreateExampleRequest {
  string app     = 1 [(validate.rules).string = {min_len: 1, max_len: 64}];
  string name    = 2 [(validate.rules).string = {min_len: 1, max_len: 64}];
  string status  = 3 [(validate.rules).string = {in: ["enabled", "disabled"]}];
  string remark  = 4 [(validate.rules).string = {max_len: 255}];
}

message CreateExampleResponse {
  string id = 1;
  int64 created_at = 2;
}
```

### 服务定义（xxx_service.proto）

```protobuf
syntax = "proto3";

package resource.api.example;

option go_package = "resource/api/example;example";

import "api/example/proto/example.proto";
import "google/api/annotations.proto";

service ExampleService {
  // CreateExample 创建示例资源
  rpc CreateExample(CreateExampleRequest) returns (CreateExampleResponse) {
    option (google.api.http) = {
      post: "/api/v1/example"
      body: "*"
    };
  }
}
```

### 编写规则

1. 每个服务拆分为 `xxx.proto`（消息）+ `xxx_service.proto`（服务）
2. `xxx_service.proto` 必须 import 对应的 `xxx.proto`（使用完整相对路径）
3. `xxx_service.proto` 必须 import `google/api/annotations.proto` 以定义 HTTP 路由
4. HTTP 路由通过 `google.api.http` 注解定义
5. 字段命名使用 snake_case（proto 规范），生成 Go 代码自动转为 CamelCase
6. 时间戳使用 `int64`（Unix 毫秒），不使用 `google.protobuf.Timestamp`
7. 同一 `proto/` 目录下的所有文件必须使用相同的 `package` 和 `go_package`
8. RPC 请求/响应消息命名必须使用 `<MethodName>Request` 和 `<MethodName>Response`；不要使用 `<MethodName>Reply`
9. 所有 RPC Request 消息必须使用 `validate/validate.proto` 限制请求参数；字符串至少设置非空规则，外部输入字段必须设置合理长度上限或枚举范围
10. Response 消息不需要添加 validate 规则；返回内容由服务端构造，边界控制应在业务代码中完成
11. **可被设置成零值的字段必须用 `optional`**：更新类（`Update*`）或部分更新语义的 Request 里，凡"合法值包含 `0` / `""` / `false`"的字段（状态、开关、描述 / 简介、允许设 `0` 的数值、可空列）**必须声明为 `optional`**。proto3 裸标量无字段存在性，`optional` 生成 Go 指针（`*T`），服务端据此区分"未传"与"显式设为零值"；漏了它，把某字段改成 `0` / `""` / `false` 的请求会与"不改"无法区分而被静默丢弃（与 infra-layer.md「更新用实体、零值列用指针」是同一问题的接口侧对应）。必填且永不为零的字段（如 `id` / `app`）仍用裸标量。
12. **字符串 `max_len` 必须 ≤ 对应 DB 列宽**：缺上限或超过列宽时，超长输入会在插入 / 更新时撞 MySQL 1406（Data too long），表现为通用 500 `DatabaseError` 而非干净的 400 `ParamsError`。后缀 / 路径等还要带格式 `pattern`（如 `ext` → `^[a-z0-9]*$`）。
13. **凭证类字段（`sk` 等）永不进响应**：`optional` 回填类场景下，需要让前端知道"已配置"就回固定占位符（如 `******`）或布尔，**不要回显真实值、也不要回显能反推的部分掩码**；且写入侧必须能识别占位符回填并忽略（例：sk 含 `*` 即视为"不修改"），否则表单整体回填会把占位符写成真凭证。脱敏在 app 层响应转换做，不在 repo 读里做（见 infra-layer.md「repo 读返回权威值」）。

```protobuf
message UpdateDirectoryRequest {
  uint32 id              = 1 [(validate.rules).uint32 = {gte: 1}]; // 必填，裸标量
  string app             = 2 [(validate.rules).string = {min_len: 1}];
  optional string path   = 3; // 可选：传了才移动
  optional string accept = 4; // 允许设 ""（禁止所有类型）⇒ 必须 optional
  optional uint32 max_size = 5;
}
```

## 错误码定义

### errors.proto

```protobuf
syntax = "proto3";

package resource.api.errors;

import "errorsx/errors.proto";
option go_package = "resource/api/errors;errors";

enum ErrorReason {
  option (errors.default_code) = 500;

  ParamsError     = 0 [(errors.message)="参数错误"];
  SystemError     = 1 [(errors.message)="系统错误"];
  DatabaseError   = 2 [(errors.message)="数据库错误"];
  TransformError  = 3 [(errors.message)="数据转换失败"];
  GetError        = 4 [(errors.message)="获取数据失败"];
  ListError       = 5 [(errors.message)="获取列表数据失败"];
  CreateError     = 6 [(errors.message)="创建数据失败"];
  UpdateError     = 7 [(errors.message)="更新数据失败"];
  DeleteError     = 8 [(errors.message)="删除数据失败"];

  // 业务错误
  StatusDisabledError = 20 [(errors.message)="资源已停用"];
}
```

### 使用生成的错误函数

```go
import "resource/api/errors"

// 无参数 — 返回默认消息
return nil, errors.GetError()

// 带参数 — 追加详细信息
return nil, errors.CreateError("名称已存在")

// 判断错误类型
if errors.IsParamsError(err) {
    // 处理参数错误
}
```

## 代码生成

使用 kratosx proto 插件生成代码：

```bash
# 生成消息 + 服务 + gRPC + HTTP + OpenAPI
kratosx proto client api/example/proto/example_service.proto

# 生成错误码
kratosx proto client api/errors/proto/errors.proto

# 全量生成
make proto
```

生成文件包括：
- `*.pb.go`：消息结构体
- `*_grpc.pb.go`：gRPC 服务端/客户端
- `*_http.pb.go`：HTTP 网关
- `openapi.yaml`：OpenAPI 文档
- `*_errors.pb.go`：错误函数

**注意：**
- 生成代码不手动修改，修改 proto 后重新生成
- `kratosx proto client` 接收**相对路径**参数（相对于项目根目录）
- VSCode 保存 `.proto` 文件时由 RunOnSave 自动触发生成（见 vscode.md）

## buf.yaml 配置

项目根目录的 `buf.yaml` 定义 proto 模块结构：

```yaml
version: v2
modules:
  - path: .
    excludes:
      - third_party
  - path: third_party
```

- 模块 `.`：项目自有 proto（排除 third_party 避免重复扫描）
- 模块 `third_party`：第三方 proto 依赖（google/api、validate、errorsx）

buf 扩展依据此文件解析 import 路径和执行 lint 检查。
