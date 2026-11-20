# Redis (library/redis)

> 包路径：`github.com/limes-cloud/kratosx/library/redis`

## 概述

封装 go-redis/v9，支持多 Redis 实例管理。

## 配置

```yaml
redis:
  - enable: true
    name: cache           # 实例名称
    host: 127.0.0.1:6379
    username: ""
    password: ""
    db: 0
    poolSize: 10
    minIdleConns: 5
    dialTimeout: 5s
    readTimeout: 3s
    writeTimeout: 3s
  - enable: true
    name: session
    host: 127.0.0.1:6380
```

## 使用方式

```go
// 获取默认（第一个）Redis 实例
rdb := ctx.Redis()

// 获取指定名称的 Redis 实例
rdb := ctx.Redis("session")

// 标准 go-redis 操作
err := rdb.Set(ctx.Ctx(), "key", "value", time.Hour).Err()
val, err := rdb.Get(ctx.Ctx(), "key").Result()

// Pipeline
pipe := rdb.Pipeline()
pipe.Set(ctx.Ctx(), "key1", "val1", 0)
pipe.Set(ctx.Ctx(), "key2", "val2", 0)
_, err := pipe.Exec(ctx.Ctx())
```

## Redis 接口

```go
type Redis interface {
    // Get 获取指定名称的 redis 实例
    // 当只存在一个配置时，name 可不传
    Get(name ...string) *goredis.Client
}
```

## 注意事项

- 第一个配置的 Redis 为默认实例，`ctx.Redis()` 不传参时返回默认实例
- 配置 `enable: false` 的实例不会被初始化
- 初始化时会执行 `Ping` 验证连接，失败则 panic
