package dbs

import (
	"sync"

	"gorm.io/gorm/clause"

	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/repository"
)

// Object 物理对象（内容寻址）仓储实现
type Object struct{}

var (
	objectIns  *Object
	objectOnce sync.Once
)

// NewObject 取物理对象仓储单例
func NewObject() *Object {
	objectOnce.Do(func() { objectIns = &Object{} })
	return objectIns
}

// GetObjectByMd5 按 (store_keyword, md5) 查对象；查不到即 gorm.ErrRecordNotFound。
func (r Object) GetObjectByMd5(ctx core.Context, storeKeyword, md5 string) (*entity.Object, error) {
	var ent entity.Object
	if err := ctx.DB().Where("store_keyword = ? AND md5 = ?", storeKeyword, md5).First(&ent).Error; err != nil {
		return nil, err
	}
	return &ent, nil
}

// GetObjectByMd5Any 按 md5 全局查对象（不限定存储，取最早入库一条；内容寻址，同 md5 内容一致）。
func (r Object) GetObjectByMd5Any(ctx core.Context, md5 string) (*entity.Object, error) {
	var ent entity.Object
	if err := ctx.DB().Where("md5 = ?", md5).Order("id ASC").First(&ent).Error; err != nil {
		return nil, err
	}
	return &ent, nil
}

// GetObject 按主键查对象
func (r Object) GetObject(ctx core.Context, id uint32) (*entity.Object, error) {
	var ent entity.Object
	if err := ctx.DB().Where("id = ?", id).First(&ent).Error; err != nil {
		return nil, err
	}
	return &ent, nil
}

// CreateObject 幂等新建；撞 uidx_object_store_md5 时重查复用赢家，返回最终行。
func (r Object) CreateObject(ctx core.Context, ent *entity.Object) (*entity.Object, error) {
	if err := ctx.DB().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "store_keyword"}, {Name: "md5"}},
		DoNothing: true,
	}).Create(ent).Error; err != nil {
		return nil, err
	}
	if ent.Id == 0 {
		return r.GetObjectByMd5(ctx, ent.StoreKeyword, ent.Md5)
	}
	return ent, nil
}

// DeleteObject 删除对象行（桶对象由调用方另行处理）。
func (r Object) DeleteObject(ctx core.Context, id uint32) error {
	return ctx.DB().Where("id = ?", id).Delete(&entity.Object{}).Error
}

// ListObjectByIDs 主键批量查对象（列表出 URL 用，不入逐行点查）。
func (r Object) ListObjectByIDs(ctx core.Context, ids []uint32) ([]*entity.Object, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []*entity.Object
	if err := ctx.DB().Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// ExistsObject 存在性判断（按 store_keyword + md5）。
func (r Object) ExistsObject(ctx core.Context, storeKeyword, md5 string) (bool, error) {
	var n int64
	if err := ctx.DB().Model(&entity.Object{}).
		Where("store_keyword = ? AND md5 = ?", storeKeyword, md5).
		Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// IsReferenced 引用判定：file 表是否仍存在指向该对象的行。
func (r Object) IsReferenced(ctx core.Context, objectID uint32) (bool, error) {
	var n int64
	if err := ctx.DB().Model(&entity.File{}).
		Where("object_id = ?", objectID).
		Limit(1).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// DeleteObjectIfUnreferenced 零引用才删行；返回 true 表示已删除。
func (r Object) DeleteObjectIfUnreferenced(ctx core.Context, objectID uint32) (bool, error) {
	res := ctx.DB().Where("id = ? AND NOT EXISTS (SELECT 1 FROM file WHERE file.object_id = object.id)", objectID).
		Delete(&entity.Object{})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// ScanUnreferenced 按主键区间分批扫零引用行（离线核对）。
func (r Object) ScanUnreferenced(ctx core.Context, afterID uint32, limit int) ([]*entity.Object, error) {
	var list []*entity.Object
	if err := ctx.DB().
		Where("id > ? AND NOT EXISTS (SELECT 1 FROM file WHERE file.object_id = object.id)", afterID).
		Order("id ASC").
		Limit(limit).
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// 编译期断言：*Object 实现 repository.Object
var _ repository.Object = (*Object)(nil)
