package dbs

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/limes-cloud/kratosx/library/lock"

	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/repository"
	"github.com/limes-cloud/resource/internal/types"
)

const (
	// appConfigKeyPrefix app_config 缓存键前缀（正/负缓存同 key）
	appConfigKeyPrefix = "resource:app_config:"
	// appConfigLockPrefix app_config 重建锁键前缀
	appConfigLockPrefix = "resource:lock:app_config:"

	// appConfigRebuildLockTtl 重建锁 TTL
	appConfigRebuildLockTtl = 3 * time.Second
	// appConfigRebuildSpinTimes / appConfigRebuildSpinInterval 未抢到锁时的自旋重试
	appConfigRebuildSpinTimes    = 5
	appConfigRebuildSpinInterval = 20 * time.Millisecond
	// appConfigDoubleDeleteDelay 延迟双删第二次的延迟
	appConfigDoubleDeleteDelay = 500 * time.Millisecond
	// 正/负缓存兜底 TTL（配置缺失时默认）
	defaultAppConfigTtl         = time.Hour
	defaultAppConfigNegativeTtl = 10 * time.Second
)

// AppConfig 应用配置仓储实现（Redis 缓存 + 延迟双删）
type AppConfig struct{}

var (
	appConfigIns  *AppConfig
	appConfigOnce sync.Once
)

// NewAppConfig 取应用配置仓储单例
func NewAppConfig() *AppConfig {
	appConfigOnce.Do(func() { appConfigIns = &AppConfig{} })
	return appConfigIns
}

// AppConfigCacheKey 返回某 app 的配置缓存键（正/负缓存共用）
func AppConfigCacheKey(app string) string { return appConfigKeyPrefix + app }

// GetAppConfig 读配置：Redis 优先；miss 时抢重建锁回源并回填。
func (r AppConfig) GetAppConfig(ctx core.Context, app string) (*entity.AppConfig, error) {
	key := AppConfigCacheKey(app)

	// 1) 命中即用
	if ent, hit := r.cacheLookup(ctx, app, key); hit {
		if ent == nil {
			return nil, gorm.ErrRecordNotFound
		}
		return ent, nil
	}

	// 2) miss → 抢重建锁（kratosx 分布式锁，自动续期/退出释放）
	lk := ctx.Lock(appConfigLockPrefix+app, lock.WithLockTimeout(appConfigRebuildLockTtl))
	locked, err := lk.TryAcquire()
	if err != nil {
		ctx.Logger().Warnw("msg", "acquire app config rebuild lock error", "key", app, "err", err.Error())
	}
	if locked {
		defer func() {
			if relErr := lk.Release(); relErr != nil {
				ctx.Logger().Warnw("msg", "release app config rebuild lock error", "key", app, "err", relErr.Error())
			}
		}()

		// 双检：从 miss 到抢到锁之间，上一个持锁者可能已回填
		if ent, hit := r.cacheLookup(ctx, app, key); hit {
			if ent == nil {
				return nil, gorm.ErrRecordNotFound
			}
			return ent, nil
		}
		return r.rebuild(ctx, app, key)
	}

	// 3) 没抢到 → 自旋重试 GET（持锁者可能马上就回填好了）
	for i := 0; i < appConfigRebuildSpinTimes; i++ {
		time.Sleep(appConfigRebuildSpinInterval)
		if ent, hit := r.cacheLookup(ctx, app, key); hit {
			if ent == nil {
				return nil, gorm.ErrRecordNotFound
			}
			return ent, nil
		}
	}

	// 4) 仍未命中 → 兜底直查返回，不回填
	return r.queryByApp(ctx, app)
}

// ListAppConfig 分页查询应用配置
func (r AppConfig) ListAppConfig(ctx core.Context, req *types.ListAppConfigRequest) ([]*entity.AppConfig, uint32, error) {
	var (
		list  []*entity.AppConfig
		total int64
	)

	db := ctx.DB().Model(&entity.AppConfig{})
	if req.App != nil {
		db = db.Where("app = ?", *req.App)
	}
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset, limit := req.Normalize()
	if err := db.Order("id asc").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, uint32(total), nil
}

// CreateAppConfig 创建应用配置。
func (r AppConfig) CreateAppConfig(ctx core.Context, ent *entity.AppConfig) (uint32, error) {
	if err := ctx.DB().Create(ent).Error; err != nil {
		return 0, err
	}
	// 清负缓存哨兵：配置前若有人访问过该 app，会写入"未配置"哨兵，不清会让配好后仍返回未配置直到过期
	if err := ctx.Redis().Del(ctx.Ctx(), AppConfigCacheKey(ent.App)).Err(); err != nil {
		ctx.Logger().Warnw("msg", "create app config cache delete error", "key", ent.App, "err", err.Error())
	}
	return ent.Id, nil
}

// UpdateAppConfig 局部更新行并执行延迟双删（零值可写列在 entity 里是指针，非 nil 即写入；不得被 ctx.Transaction 包裹）。
func (r AppConfig) UpdateAppConfig(ctx core.Context, ent *entity.AppConfig) error {
	if err := ctx.DB().Model(&entity.AppConfig{}).Where("app = ?", ent.App).Updates(ent).Error; err != nil {
		return err
	}

	// 第一次删
	key := AppConfigCacheKey(ent.App)
	if err := ctx.Redis().Del(ctx.Ctx(), key).Err(); err != nil {
		ctx.Logger().Warnw("msg", "first app config cache delete error", "key", ent.App, "err", err.Error())
	}

	// 第二次删：脱离请求上下文
	go func(c core.Context) {
		time.Sleep(appConfigDoubleDeleteDelay)
		if err := c.Redis().Del(c.Ctx(), key).Err(); err != nil {
			c.Logger().Warnw("msg", "second app config cache delete error", "key", ent.App, "err", err.Error())
		}
	}(ctx.Clone())

	return nil
}

// cacheLookup 读缓存并解析；hit=false 表示需要回源，ent 为 nil 表示命中负缓存。
func (r AppConfig) cacheLookup(ctx core.Context, app, key string) (*entity.AppConfig, bool) {
	value, err := ctx.Redis().Get(ctx.Ctx(), key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false
		}
		ctx.Logger().Warnw("msg", "read app config cache error", "key", app, "err", err.Error())
		return nil, false
	}
	if value == "" {
		// 空串即哨兵：已知不存在，直接按"app 未配置"返回，不抢锁
		return nil, true
	}

	ent := &entity.AppConfig{}
	if err := json.Unmarshal([]byte(value), ent); err != nil {
		// 脏值一律按 miss 处理
		ctx.Logger().Warnw("msg", "decode app config cache error", "key", app, "err", err.Error())
		return nil, false
	}
	return ent, true
}

// rebuild 回源查库并回填缓存（必须在锁内调用）
func (r AppConfig) rebuild(ctx core.Context, app, key string) (*entity.AppConfig, error) {
	ent, err := r.queryByApp(ctx, app)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 回填空结果哨兵（与正缓存同 key），挡穿透
		if setErr := ctx.Redis().Set(ctx.Ctx(), key, "", appConfigNegativeTtl(ctx)).Err(); setErr != nil {
			ctx.Logger().Warnw("msg", "write app config negative cache error", "key", app, "err", setErr.Error())
		}
		return nil, gorm.ErrRecordNotFound
	}
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(ent)
	if err != nil {
		return nil, err
	}
	if setErr := ctx.Redis().Set(ctx.Ctx(), key, string(data), appConfigTtl(ctx)).Err(); setErr != nil {
		ctx.Logger().Warnw("msg", "write app config cache error", "key", app, "err", setErr.Error())
	}
	return ent, nil
}

// queryByApp 按 app 查库（走 uidx_app_config_app）
func (r AppConfig) queryByApp(ctx core.Context, app string) (*entity.AppConfig, error) {
	var ent entity.AppConfig
	if err := ctx.DB().Where("app = ?", app).First(&ent).Error; err != nil {
		return nil, err
	}
	return &ent, nil
}

// appConfigTtl 正缓存兜底 TTL（读配置，缺失时给默认值）
func appConfigTtl(ctx core.Context) time.Duration {
	if c := ctx.Config(); c != nil && c.AppConfigCache != nil && c.AppConfigCache.Ttl > 0 {
		return c.AppConfigCache.Ttl
	}
	return defaultAppConfigTtl
}

// appConfigNegativeTtl 负缓存 TTL（读配置，缺失时给默认值）
func appConfigNegativeTtl(ctx core.Context) time.Duration {
	if c := ctx.Config(); c != nil && c.AppConfigCache != nil && c.AppConfigCache.NegativeTtl > 0 {
		return c.AppConfigCache.NegativeTtl
	}
	return defaultAppConfigNegativeTtl
}

// ReserveQuota 原子占用配额：单条 CAS 既校验又自增（防并发绕过）。rowsAffected==1 即占用成功。
func (r AppConfig) ReserveQuota(ctx core.Context, app string, size uint64) (bool, error) {
	if size == 0 {
		return true, nil
	}
	res := ctx.DB().Model(&entity.AppConfig{}).
		Where("app = ? AND (quota = 0 OR used_bytes + ? <= quota)", app, size).
		UpdateColumn("used_bytes", gorm.Expr("used_bytes + ?", size))
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// ReleaseQuota 归还配额：used_bytes -= size，IF 夹 0 防无符号下溢。
func (r AppConfig) ReleaseQuota(ctx core.Context, app string, size uint64) error {
	if size == 0 {
		return nil
	}
	return ctx.DB().Model(&entity.AppConfig{}).
		Where("app = ?", app).
		UpdateColumn("used_bytes", gorm.Expr("IF(used_bytes >= ?, used_bytes - ?, 0)", size, size)).Error
}

// RecalcUsage 按逻辑口径重算 used_bytes（Σ 该 app 每个 file 行引用 object 的 size），失效缓存后回查。
func (r AppConfig) RecalcUsage(ctx core.Context, app string) (uint64, error) {
	const sql = "UPDATE app_config a SET a.used_bytes = (SELECT COALESCE(SUM(o.size),0) FROM file f JOIN object o ON o.id = f.object_id WHERE f.app = a.app AND f.status = 1) WHERE a.app = ?"
	if err := ctx.DB().Exec(sql, app).Error; err != nil {
		return 0, err
	}
	if err := ctx.Redis().Del(ctx.Ctx(), AppConfigCacheKey(app)).Err(); err != nil {
		ctx.Logger().Warnw("msg", "recalc usage cache delete error", "key", app, "err", err.Error())
	}
	var ent entity.AppConfig
	if err := ctx.DB().Select("used_bytes").Where("app = ?", app).First(&ent).Error; err != nil {
		return 0, err
	}
	return ent.UsedBytes, nil
}

// 编译期断言：*AppConfig 实现 repository.AppConfig
var _ repository.AppConfig = (*AppConfig)(nil)
