# VSCode 开发环境配置

## 必装扩展

| 扩展 | 用途 |
|---|---|
| `bufbuild.vscode-buf` | Proto 文件格式化（buf format）+ import 路径检查 |
| `golang.go` | Go 语言支持 |
| `emeraldwalk.RunOnSave` | 保存时自动执行命令 |

## 保存时自动化

### Proto 文件（`.proto`）

保存时自动执行两步：
1. **格式化**：由 `bufbuild.vscode-buf` 扩展完成（buf format）
2. **代码生成**：由 `emeraldwalk.RunOnSave` 触发 `kratosx proto client` 生成 Go 代码

语言 ID 是 `proto`（不是 `proto3`）。

### Go 文件（`.go`）

- 格式化：`goimports`
- 保存时自动 organize imports
- gopls local 前缀：`github.com/limes-cloud/resource`

## Proto 代码生成

```bash
# 单文件生成（RunOnSave 自动执行）
kratosx proto client api/example/proto/example_service.proto

# 全量生成
make proto
```

`kratosx proto client` 接收**相对路径**参数，如 `api/example/proto/xxx.proto`。

## go_package 格式

Proto 文件的 `go_package` 使用项目内相对路径，不含完整 module 前缀：

```protobuf
option go_package = "resource/api/example;example";
```

## Proto Import 路径解析（buf.yaml + third_party 软链）

### 问题

Proto 文件中 `import "google/api/annotations.proto"` 等第三方 proto 依赖在 VSCode 中会报 `imported file does not exist`。这些 proto 文件已存在于 kratosx 模块的 `third_party/` 目录中，无需重复下载。

### 解决方案

通过在项目 `third_party/` 目录创建软链接指向 kratosx 模块缓存中的 proto 文件，配合 `buf.yaml` 配置多搜索根。

### 目录结构

```
third_party/
├── errorsx -> <kratosx>/third_party/errorsx   # 软链接（kratosx 错误码扩展）
├── google -> <kratosx>/third_party/google     # 软链接
└── validate -> <kratosx>/third_party/validate # 软链接
```

### 步骤 1：定位 kratosx 模块路径

从 `go.sum` 或 `go.mod` 中获取当前使用的 kratosx 版本，然后定位模块缓存路径：

```bash
# 获取 kratosx 版本（从 go.mod）
KRATOSX_VERSION=$(grep "github.com/limes-cloud/kratosx" go.mod | awk '{print $2}')

# 模块缓存路径
KRATOSX_PATH="$(go env GOPATH)/pkg/mod/github.com/limes-cloud/kratosx@${KRATOSX_VERSION}/third_party"

# 验证路径存在
ls ${KRATOSX_PATH}/google/api/annotations.proto
```

如果模块缓存中不存在（如首次 clone 项目），先下载依赖：

```bash
go mod download github.com/limes-cloud/kratosx
```

### 步骤 2：创建软链接

```bash
# 删除可能存在的旧目录或坏链接
rm -rf third_party/google third_party/validate third_party/errorsx

# 创建软链接
ln -s ${KRATOSX_PATH}/google third_party/google
ln -s ${KRATOSX_PATH}/validate third_party/validate
ln -s ${KRATOSX_PATH}/errorsx third_party/errorsx
```

### 步骤 3：buf.yaml 配置

项目根目录的 `buf.yaml` 定义两个模块搜索根：

```yaml
version: v2
modules:
  - path: .
    excludes:
      - third_party
  - path: third_party
```

- `path: .`（排除 third_party）→ 解析项目自有 proto，如 `api/example/proto/example.proto`
- `path: third_party` → 解析第三方 proto，如 `google/api/annotations.proto`、`validate/validate.proto`、`errorsx/errors.proto`

### 注意事项

1. **kratosx 版本升级后**需重建软链接（指向新版本路径）
2. **CI/Docker 环境**不使用软链接，kratosx proto client 内部已自带 proto 搜索路径
3. **软链接不提交 git**（`.gitignore` 中应忽略 `third_party/google`、`third_party/validate`、`third_party/errorsx`）
4. **buf.yaml 需要提交 git**，它定义了 proto 模块结构

### 一键初始化脚本

```bash
#!/bin/bash
# 初始化 third_party proto 软链接
set -e

KRATOSX_VERSION=$(grep "github.com/limes-cloud/kratosx" go.mod | awk '{print $2}')
KRATOSX_PATH="$(go env GOPATH)/pkg/mod/github.com/limes-cloud/kratosx@${KRATOSX_VERSION}/third_party"

if [ ! -d "${KRATOSX_PATH}" ]; then
    echo "kratosx 模块缓存不存在，正在下载..."
    go mod download github.com/limes-cloud/kratosx
fi

rm -rf third_party/google third_party/validate third_party/errorsx
ln -s "${KRATOSX_PATH}/google" third_party/google
ln -s "${KRATOSX_PATH}/validate" third_party/validate
ln -s "${KRATOSX_PATH}/errorsx" third_party/errorsx

echo "软链接创建完成："
ls -la third_party/google third_party/validate third_party/errorsx
```

## 工作区要求

VSCode 必须直接打开 `resource/` 目录作为工作区根目录（不要打开上级目录），否则 `${workspaceRoot}` 变量会指向错误路径，导致 RunOnSave 命令失败。
