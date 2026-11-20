# Pprof 性能分析 (library/pprof)

> 包路径：`github.com/limes-cloud/kratosx/library/pprof`

## 概述

在 HTTP Server 上暴露 pprof 性能分析端点，通过 Secret 保护访问。

## 配置

```yaml
server:
  http:
    pprof:
      query: "token"           # URL Query 参数名
      secret: "your-secret"    # 访问密钥
```

## 使用方式

启用后，以下端点可用（需附带密钥）：

```
GET /debug/pprof?token=your-secret
GET /debug/cmdline?token=your-secret
GET /debug/profile?token=your-secret
GET /debug/symbol?token=your-secret
GET /debug/trace?token=your-secret
GET /debug/allocs?token=your-secret
GET /debug/block?token=your-secret
GET /debug/goroutine?token=your-secret
GET /debug/heap?token=your-secret
GET /debug/mutex?token=your-secret
GET /debug/threadcreate?token=your-secret
```

不提供正确密钥返回 `401 Unauthorized`。

## 使用 go tool pprof

```bash
# CPU Profile（30s）
go tool pprof http://localhost:7010/debug/profile?token=your-secret

# Heap
go tool pprof http://localhost:7010/debug/heap?token=your-secret

# Goroutine
go tool pprof http://localhost:7010/debug/goroutine?token=your-secret
```
