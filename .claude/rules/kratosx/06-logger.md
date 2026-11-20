# 日志 (library/logger)

> 包路径：`github.com/limes-cloud/kratosx/library/logger`

## 概述

基于 zap 封装的结构化日志库，支持多输出（stdout/file）、日志切割、链路追踪集成。

## 配置

```yaml
logger:
  level: 0              # 0=Debug 1=Info 2=Warn 3=Error
  output: ["stdout"]    # 输出位置：stdout, file
  enCoder: json         # 编码器：json, console
  caller: false         # 是否显示调用者信息
  callerSkip: 2         # 调用者层级
  hookStd: false        # 是否劫持标准输出
  file:
    name: app.log       # 日志文件名
    splitTime: 24h      # 日志切割间隔
    maxAge: 168h        # 备份保留时间（7天）
    maxBackup: 7        # 最大备份数
    errorAlone: false   # 错误日志单独输出
```

## 使用方式

### 通过 Context 获取

```go
ctx.Logger().Info("操作成功", logger.F("userId", "123"))
ctx.Logger().Warn("警告信息", logger.F("err", err.Error()))
ctx.Logger().Error("系统错误", logger.F("err", err.Error()))
```

### Logger 接口

```go
type Logger interface {
    // 日志输出
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    Fatal(msg string, fields ...Field)

    // 带格式化输出
    Debugf(format string, args ...any)
    Infof(format string, args ...any)
    Warnf(format string, args ...any)
    Errorf(format string, args ...any)
    Fatalf(format string, args ...any)

    // 带 key-value 对输出（兼容 kratos 风格）
    Debugw(keysAndValues ...any)
    Infow(keysAndValues ...any)
    Warnw(keysAndValues ...any)
    Errorw(keysAndValues ...any)
    Fatalw(keysAndValues ...any)

    // 带上下文
    WithContext(ctx context.Context) Logger

    // 底层 Sync
    Sync() error
}
```

### Field 辅助函数

```go
// 创建日志字段
logger.F("key", value)    // 返回 Field 类型

// 使用示例
ctx.Logger().Info("create example",
    logger.F("userId", req.UserID),
    logger.F("sessionId", req.SessionID),
    logger.F("cost", time.Since(start).Milliseconds()),
)
```

### w 风格日志（key-value 对）

```go
// 与 limes-cloud/manager 保持一致的风格
ctx.Logger().Warnw("msg", "get role error", "err", err.Error())
ctx.Logger().Errorw("msg", "database connection failed", "err", err.Error())
```

## 日志输出示例

JSON 编码器：
```json
{"level":"info","ts":"2024-01-01T00:00:00.000Z","msg":"create example","userId":"u_123","sessionId":"s_456","cost":15}
```

Console 编码器：
```
2024-01-01T00:00:00.000Z  INFO  create example  {"userId":"u_123","sessionId":"s_456","cost":15}
```

## 全局实例

```go
// 不依赖 context 时使用全局实例
log := logger.Instance()
log.Info("application started")
```
