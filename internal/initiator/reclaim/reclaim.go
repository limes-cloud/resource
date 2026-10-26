// Package reclaim 后台回收 worker：按间隔分批清理「待删除文件行 / 无引用对象桶 / 过期上传会话」。
// 多实例靠分布式锁串行化，一次只有一个实例在扫；每轮每阶段处理一批（≤batch），下轮继续，天然可恢复。
package reclaim

import (
	"time"

	"github.com/limes-cloud/kratosx/library/lock"

	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/service"
	"github.com/limes-cloud/resource/internal/infra/dbs"
)

const (
	lockKey           = "resource:lock:reclaim"
	lockTtl           = 2 * time.Minute
	defaultInterval   = 30 * time.Second
	defaultBatch      = 1000
	defaultSessionTtl = time.Hour
	maxDrainLoops     = 50 // 单轮最多清 maxDrainLoops*batch 条，防一轮卡太久；剩余下轮续
)

// kickCh 立即触发信号（缓冲 1，合并多次触发）：删目录后 Kick 一下，sweeper 立刻跑一轮，不等 ticker。
var kickCh = make(chan struct{}, 1)

// Kick 请求立即回收一轮（非阻塞）。删目录成功后调用：第一层「立即后台删」。
func Kick() {
	select {
	case kickCh <- struct{}{}:
	default:
	}
}

// Register 注册后台回收 worker（服务启动后拉起循环）。ticker 为「第二层补偿」，Kick 为「第一层立即」。
func Register(ctx core.Context) {
	ctx.AfterStart("resource reclaim worker", func() {
		go run(ctx.Clone())
	})
}

func run(ctx core.Context) {
	interval := defaultInterval
	if c := ctx.Config(); c != nil && c.ObjectAudit != nil && c.ObjectAudit.Interval > 0 {
		interval = c.ObjectAudit.Interval
	}

	recycle := service.NewRecycle(dbs.NewFile(), dbs.NewObject(), dbs.NewStore(), dbs.NewDirectory(), dbs.NewAppConfig())
	upload := service.NewUpload(
		dbs.NewUpload(), dbs.NewObject(), dbs.NewFile(), dbs.NewStore(), dbs.NewAppConfig(),
		service.NewDirectory(dbs.NewDirectory(), dbs.NewAppConfig()),
	)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			sweep(ctx, recycle, upload)
		case <-kickCh:
			sweep(ctx, recycle, upload)
		}
	}
}

// sweep 一轮回收（抢到分布式锁才执行，确保多实例/立即触发与定时补偿之间串行）。
func sweep(ctx core.Context, recycle *service.Recycle, upload *service.Upload) {
	lk := ctx.Lock(lockKey, lock.WithLockTimeout(lockTtl))
	locked, err := lk.TryAcquire()
	if err != nil || !locked {
		// 已有一轮在跑：它会把当前所有待删/无引用一并清掉（含刚标记的），直接返回
		return
	}
	defer func() { _ = lk.Release() }()

	batch := defaultBatch
	sessionTtl := defaultSessionTtl
	if c := ctx.Config(); c != nil {
		if c.ObjectAudit != nil && c.ObjectAudit.BatchSize > 0 {
			batch = c.ObjectAudit.BatchSize
		}
		if c.Upload != nil && c.Upload.SessionTtl > 0 {
			sessionTtl = c.Upload.SessionTtl
		}
	}

	// 阶段 A/B 循环清到空（或到上限）：A 删待删文件行后 B 才能判出零引用对象，故每轮 A 先于 B
	for i := 0; i < maxDrainLoops; i++ {
		a, e := recycle.SweepDeletedFiles(ctx, batch)
		if e != nil {
			ctx.Logger().Warnw("msg", "sweep deleted files error", "err", e.Error())
		}
		b, e2 := recycle.SweepUnreferencedObjects(ctx, batch)
		if e2 != nil {
			ctx.Logger().Warnw("msg", "sweep unreferenced objects error", "err", e2.Error())
		}
		if a == 0 && b == 0 {
			break
		}
	}

	// 阶段 C：过期上传会话 GC（超时未完成的分片会话 + Abort 厂商分片）
	before := time.Now().Add(-sessionTtl).Unix()
	if n, e := upload.GarbageCollect(ctx, before, batch); e != nil {
		ctx.Logger().Warnw("msg", "gc upload sessions error", "err", e.Error())
	} else if n > 0 {
		ctx.Logger().Infow("msg", "gc upload sessions", "count", n)
	}
}
