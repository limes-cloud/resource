# 分布式锁 (library/lock)

> 包路径：`github.com/limes-cloud/kratosx/library/lock`

## 概述

基于 Redis 的分布式锁实现，支持自动续期和超时控制。

## 使用方式

```go
import "github.com/limes-cloud/kratosx/library/lock"

// 获取锁（阻塞等待）
locker, err := lock.Lock(ctx.Redis(), "my-lock-key", 30*time.Second)
if err != nil {
    return err
}
defer locker.Unlock()

// 执行互斥操作
doExclusiveWork()
```

## Lock 函数签名

```go
// Lock 获取分布式锁
// client: Redis 客户端
// key: 锁的 key
// ttl: 锁的过期时间（防止死锁）
func Lock(client *redis.Client, key string, ttl time.Duration) (*Locker, error)

// TryLock 尝试获取锁（非阻塞）
func TryLock(client *redis.Client, key string, ttl time.Duration) (*Locker, bool)
```

## 适用场景

- 防止并发重复写入（如 entity 去重）
- 定时任务防重复执行
- 资源竞争保护
