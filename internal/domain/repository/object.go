package repository

import (
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
)

// Object 物理对象（内容寻址）仓储接口。
type Object interface {
	// GetObjectByMd5 按 (store_keyword, md5) 查对象；查不到即 gorm.ErrRecordNotFound。
	GetObjectByMd5(ctx core.Context, storeKeyword, md5 string) (*entity.Object, error)
	// GetObjectByMd5Any 按 md5 全局查对象（不限定存储，取最早入库一条；内容寻址，同 md5 内容一致）。
	GetObjectByMd5Any(ctx core.Context, md5 string) (*entity.Object, error)
	// GetObject 按主键查对象。
	GetObject(ctx core.Context, id uint32) (*entity.Object, error)
	// ListObjectByIDs 主键批量查询（列表出 URL 用，禁止退化成逐行点查）。
	ListObjectByIDs(ctx core.Context, ids []uint32) ([]*entity.Object, error)
	// CreateObject 幂等新建；撞 uidx_object_store_md5 时重查复用赢家，返回最终行。
	CreateObject(ctx core.Context, ent *entity.Object) (*entity.Object, error)
	// ExistsObject 存在性判断。
	ExistsObject(ctx core.Context, storeKeyword, md5 string) (bool, error)
	// DeleteObject 删除对象行（桶对象由调用方另行处理）。
	DeleteObject(ctx core.Context, id uint32) error
	// IsReferenced 引用判定：SELECT 1 FROM file WHERE object_id=? LIMIT 1。
	IsReferenced(ctx core.Context, objectID uint32) (bool, error)
	// DeleteObjectIfUnreferenced 回收：带 NOT EXISTS 的 CAS 删除。
	DeleteObjectIfUnreferenced(ctx core.Context, objectID uint32) (bool, error)
	// ScanUnreferenced 离线核对：按主键区间分批扫零引用行。
	ScanUnreferenced(ctx core.Context, afterID uint32, limit int) ([]*entity.Object, error)
}
