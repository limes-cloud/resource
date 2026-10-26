package repository

import (
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/types"
)

// AppConfig 应用配置仓储接口。
type AppConfig interface {
	// GetAppConfig 按 app 读取应用配置（Redis 优先）。
	GetAppConfig(ctx core.Context, app string) (*entity.AppConfig, error)
	// ListAppConfig 分页查询
	ListAppConfig(ctx core.Context, req *types.ListAppConfigRequest) ([]*entity.AppConfig, uint32, error)
	// CreateAppConfig 创建应用配置
	CreateAppConfig(ctx core.Context, ent *entity.AppConfig) (uint32, error)
	// UpdateAppConfig 保存整行并执行延迟双删。
	UpdateAppConfig(ctx core.Context, ent *entity.AppConfig) error

	// ReserveQuota 原子占用配额：used_bytes += size，仅当 quota=0（不限）或 used_bytes+size<=quota 时成功。
	// 返回 true=占用成功；false=超限或 app 不存在。必须在上传完成事务内、建 file 行后调用。
	ReserveQuota(ctx core.Context, app string, size uint64) (bool, error)
	// ReleaseQuota 归还配额：used_bytes -= size（下溢夹 0）。删除文件事务内调用。
	ReleaseQuota(ctx core.Context, app string, size uint64) error
	// RecalcUsage 按明细重算 used_bytes（对账纠漂移），失效缓存并返回重算后的用量。
	RecalcUsage(ctx core.Context, app string) (uint64, error)
}
