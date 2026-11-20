# 生命周期管理 (library/tasker)

> 包路径：`github.com/limes-cloud/kratosx/library/tasker`

## 概述

应用生命周期钩子管理，支持在服务启动/停止的前后注册回调函数。

## Tasker 接口

```go
type Tasker interface {
    // BeforeStart 注册启动前的处理函数
    BeforeStart(name string, f func())

    // AfterStart 注册启动后的处理函数
    AfterStart(name string, f func())

    // BeforeStop 注册停止前的处理函数
    BeforeStop(name string, f func())

    // AfterStop 注册停止后的处理函数
    AfterStop(name string, f func())

    // Remove 移除指定 name 的回调
    Remove(name string)
}
```

## 使用方式

### 通过 Context 注册

```go
// 服务启动后执行（如预热缓存）
ctx.AfterStart("warmup-cache", func() {
    warmupCache()
})

// 服务停止前执行（如刷新缓冲区）
ctx.BeforeStop("flush-buffer", func() {
    flushAllBuffers()
})

// 服务停止后执行（如关闭连接）
ctx.AfterStop("close-connections", func() {
    closeExternalConnections()
})
```

### 执行顺序

```
BeforeStart → kratos.App.Start() → AfterStart
                    ...运行中...
BeforeStop → kratos.App.Stop() → AfterStop
```

### 注意事项

- 回调按注册顺序执行
- 每个回调有日志记录（开始/完成时间）
- 回调执行完成后才继续下一步
- 可通过 `Remove(name)` 移除不再需要的回调
