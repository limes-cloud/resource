# 环境管理 (library/env)

> 包路径：`github.com/limes-cloud/kratosx/library/env`

## 概述

应用环境变量管理，支持 `.env` 文件加载和环境标识。

## Env 接口

```go
type Env interface {
    // AppName 获取应用名称
    AppName() string

    // IsDev 是否为开发环境
    IsDev() bool

    // IsTest 是否为测试环境
    IsTest() bool

    // IsProd 是否为生产环境
    IsProd() bool

    // IsUnitTest 是否为单元测试模式
    IsUnitTest() bool

    // SetRunUnitTest 标记为单元测试
    SetRunUnitTest()
}
```

## 使用方式

```go
// 通过 context 获取
env := ctx.Env()

if env.IsDev() {
    // 开发环境特有逻辑
}

if env.IsProd() {
    // 生产环境特有逻辑
}
```

## .env 文件

在项目根目录创建 `.env` 文件：

```bash
APP_NAME=resource
APP_ENV=dev        # dev / test / prod
APP_VERSION=v1.0.0
```

kratosx 启动时通过 `env.Load()` 自动加载 `.env` 文件（使用 [joho/godotenv](https://github.com/joho/godotenv)）。
