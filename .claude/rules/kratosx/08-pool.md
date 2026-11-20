# 协程池 (library/pool)

> 包路径：`github.com/limes-cloud/kratosx/library/pool`

## 概述

基于 [panjf2000/ants](https://github.com/panjf2000/ants) 封装的全局协程池，支持上下文感知的任务执行和 WaitRunner 批量任务编排。

## 配置

```yaml
pool:
  size: 1000              # 最大协程数量
  expiryDuration: 10s     # 空闲协程过期时间
  preAlloc: true          # 是否预分配
  maxBlockingTasks: 100   # 最大阻塞等待任务数
  nonblocking: false      # 设为 true 则不限制阻塞任务数
```

## Pool 接口

```go
type Pool interface {
    // GoFunc 提交一个函数任务到协程池
    GoFunc(fn func()) error

    // Go 提交一个 Runner 接口任务
    Go(runner Runner) error

    // WithContext 携带上下文（任务会监听 context 取消）
    WithContext(ctx context.Context) Pool

    // NewWaitRunner 创建批量等待任务执行器
    NewWaitRunner(opts ...WaitRunnerOptionFunc) WaitRunner
}

type Runner interface {
    Run()
}
```

## 使用方式

### 基础用法

```go
// 提交简单任务
err := ctx.Pool().GoFunc(func() {
    // 异步执行的逻辑
    processData(data)
})

// 带上下文的任务（context 取消时任务不执行）
err := ctx.Pool().WithContext(ctx.Ctx()).GoFunc(func() {
    // 若 context 已取消，此函数不会执行
    sendNotification()
})
```

### WaitRunner — 批量任务等待

```go
// 创建等待执行器
runner := ctx.Pool().NewWaitRunner(
    pool.WithMaxWaitRunnerOption(10),       // 最大并发数
    pool.WithErrorBreakOption(true),        // 遇错停止
    pool.WithRetryCountOption(3),           // 失败重试3次
    pool.WithRetryWaitTimeOption(time.Second), // 重试间隔
)

// 添加任务
for _, item := range items {
    item := item
    runner.AddTask(func() error {
        return processItem(item)
    })
}

// 等待所有任务完成
if err := runner.Wait(); err != nil {
    ctx.Logger().Error("batch process failed", logger.F("err", err.Error()))
}

// 获取所有错误
for _, e := range runner.ErrorList() {
    ctx.Logger().Warn("task error", logger.F("err", e.Error()))
}
```

### WaitRunner 选项

| 选项 | 说明 |
|------|------|
| `WithMaxWaitRunnerOption(n)` | 限制最大并发数 |
| `WithErrorBreakOption(true)` | 遇到第一个错误后停止提交新任务 |
| `WithRetryCountOption(n)` | 单个任务失败时的重试次数 |
| `WithRetryWaitTimeOption(d)` | 重试间隔 |

## 适用场景

- **异步任务**：发送通知、日志上报等不需要等待结果的操作
- **并行处理**：多个独立子任务并行执行
- **批量操作**：批量计算、批量数据库写入
