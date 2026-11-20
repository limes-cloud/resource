# 构建与部署规则

## 项目构建

### 构建命令

```bash
# 生成 proto 代码
kratosx proto client api/store/proto/store_service.proto
kratosx proto client api/file/proto/file_service.proto
kratosx proto client api/directory/proto/directory_service.proto
kratosx proto client api/app_config/proto/app_config_service.proto
kratosx proto client api/errors/proto/errors.proto

# 编译
go build -ldflags="-s -w" -o resource main.go
```

### Docker 多阶段构建

参考 limes-cloud/manager 的 Dockerfile：

```dockerfile
# 构建打包镜像
FROM golang:alpine AS build
RUN apk add git

ENV GOPROXY=https://goproxy.cn,direct
ENV GO111MODULE=on

WORKDIR /go/cache
ADD api/go.mod api/go.mod
ADD api/go.sum api/go.sum
ADD go.mod .
ADD go.sum .
RUN go mod download
WORKDIR /go/build
ADD . .
RUN GOOS=linux CGO_ENABLED=0 go build -ldflags="-s -w" -installsuffix cgo -o resource main.go

# 构建执行镜像
FROM alpine
WORKDIR /go/build
COPY ./conf/*.yaml /go/build/conf/
COPY ./migrations/ /go/build/migrations/
COPY --from=build /go/build/resource /go/build/resource
CMD ["./resource"]
```

**关键点：**
- 先 ADD api/go.mod 和 api/go.sum，利用 Docker 缓存层加速依赖下载
- CGO_ENABLED=0（无 CGO 依赖，静态编译）
- 最终镜像使用 alpine，只复制必要文件

## 目录结构补充

项目根目录应包含以下部署相关文件：

```
resource/
├── Dockerfile              # 多阶段构建
├── Makefile                # 常用命令快捷方式
├── conf/
│   └── config.yaml         # 本地开发配置
├── migrations/
│   ├── 001_init_tables.sql
│   └── 002_seed_data.sql
└── deploy/
    └── data.sql            # 初始数据（可选，用于 kratosx initializer）
```

## Makefile

```makefile
.PHONY: proto build run test

# 生成 proto 代码
proto:
	kratosx proto client api/store/proto/store_service.proto
	kratosx proto client api/file/proto/file_service.proto
	kratosx proto client api/directory/proto/directory_service.proto
	kratosx proto client api/app_config/proto/app_config_service.proto
	kratosx proto client api/errors/proto/errors.proto

# 编译
build:
	go build -ldflags="-s -w" -o resource main.go

# 本地运行
run:
	go run main.go -conf conf/

# 测试
test:
	go test ./... -v -count=1

# Docker 构建
docker:
	docker build -t resource:latest .
```

## 配置管理

### 本地开发

使用 `conf/config.yaml`，通过 `file.NewSource("conf/")` 加载。

### 生产环境

通过环境变量配置远程配置中心：

```bash
export CONF_HOST=https://config-center.example.com
export CONF_TOKEN=xxx
export APP_NAME=resource
```

环境变量存在时自动切换为远程配置源，不存在时回退到本地文件。

### 配置热更新

通过 `ScanWatch` 监听 `business` 键，运行时动态更新业务配置（如 gc_interval、max_size 等），无需重启服务。

## 数据库迁移

### 首次部署

```bash
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS resource < migrations/001_init_tables.sql
mysql -h $DB_HOST -u $DB_USER -p$DB_PASS resource < migrations/002_seed_data.sql
```

### 迁移脚本规则

1. 每个迁移文件只做一件事，编号递增
2. 必须幂等（`CREATE TABLE IF NOT EXISTS` / `INSERT ... ON DUPLICATE KEY UPDATE`）
3. 包含回滚语句（注释形式）
