// Package reclaim 后台维护 worker：定时只清理过期上传会话。
// 待删除文件行与无引用对象不做周期扫描；仅在删除目录后由 Kick 立即回收一次。
// 多实例靠分布式锁串行化，一次只有一个实例在扫。
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
	maxDrainLoops     = 50 // 单轮最多清 maxDrainLoops*batch 条，防一轮卡太久；剩余等下次删除目录触发
)

// kickCh 立即回收信号（缓冲 1，合并多次触发）：删目录后跑一轮文件/对象回收，不等 ticker。
var kickCh = make(chan struct{}, 1)

// Kick 请求立即回收一轮（非阻塞）。只由删除目录触发，周期 ticker 不执行文件回收。
func Kick() {
	select {
	case kickCh <- struct{}{}:
	default:
	}
}

// Register 注册后台维护 worker（服务启动后拉起循环）。
func Register(ctx core.Context) {
	if !enabled(ctx) {
		ctx.Logger().Infow("msg", "resource periodic cleanup disabled; immediate reclaim on delete remains enabled")
	}
	ctx.AfterStart("resource reclaim worker", func() {
		go run(ctx.Clone())
	})
}

// enabled 判断周期扫描是否启用。关闭时仍保留删除目录后的即时回收。
func enabled(ctx core.Context) bool {
	c := ctx.Config()
	return c != nil && c.ObjectAudit != nil && c.ObjectAudit.Enable
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
			sweep(ctx, recycle, upload, false)
		case <-kickCh:
			sweep(ctx, recycle, upload, true)
		}
	}
}

// sweep 一轮维护（抢到分布式锁才执行，确保多实例之间串行）。
// cleanFiles=false 时只 GC 上传会话；true 时才回收待删除文件行与无引用对象。
func sweep(ctx core.Context, recycle *service.Recycle, upload *service.Upload, cleanFiles bool) {
	if !cleanFiles && !enabled(ctx) {
		return
	}
	lk := ctx.Lock(lockKey, lock.WithLockTimeout(lockTtl))
	if cleanFiles {
		// 删除目录触发的回收必须执行，等锁而不是直接丢弃。
		if err := lk.Acquire(); err != nil {
			ctx.Logger().Warnw("msg", "acquire reclaim lock for delete error", "err", err.Error())
			return
		}
	} else {
		locked, err := lk.TryAcquire()
		if err != nil || !locked {
			// 周期任务不排队：已有一轮在跑就跳过本次上传会话 GC。
			return
		}
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

	if cleanFiles {
		// 删除目录后的即时回收：A 删待删文件行后 B 才能判出零引用对象，故每轮 A 先于 B。
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
	}

	// 过期上传会话 GC（超时未完成的分片会话 + Abort 厂商分片）
	before := time.Now().Add(-sessionTtl).Unix()
	if n, e := upload.GarbageCollect(ctx, before, batch); e != nil {
		ctx.Logger().Warnw("msg", "gc upload sessions error", "err", e.Error())
	} else if n > 0 {
		ctx.Logger().Infow("msg", "gc upload sessions", "count", n)
	}
}
