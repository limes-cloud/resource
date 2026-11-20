package dbs

import (
	"sync"

	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/repository"
	"github.com/limes-cloud/resource/internal/pkg/pattern"
	"github.com/limes-cloud/resource/internal/types"
)

// File 业务文件条目仓储实现
type File struct{}

var (
	fileIns  *File
	fileOnce sync.Once
)

// NewFile 取业务文件仓储单例
func NewFile() *File {
	fileOnce.Do(func() { fileIns = &File{} })
	return fileIns
}

// CreateFile 新建文件
func (r File) CreateFile(ctx core.Context, ent *entity.File) (uint32, error) {
	if err := ctx.DB().Create(ent).Error; err != nil {
		return 0, err
	}
	return ent.Id, nil
}

// GetFile 按主键查正常文件（status=1）
func (r File) GetFile(ctx core.Context, id uint32) (*entity.File, error) {
	var ent entity.File
	if err := ctx.DB().Where("id = ? AND status = 1", id).First(&ent).Error; err != nil {
		return nil, err
	}
	return &ent, nil
}

// GetFileByUploadId 按 upload_id 查文件（幂等收口 / 召回复用）。
func (r File) GetFileByUploadId(ctx core.Context, uploadID string) (*entity.File, error) {
	var ent entity.File
	if err := ctx.DB().Where("upload_id = ?", uploadID).First(&ent).Error; err != nil {
		return nil, err
	}
	return &ent, nil
}

// DeleteFile 删除文件行
func (r File) DeleteFile(ctx core.Context, id uint32) error {
	return ctx.DB().Where("id = ?", id).Delete(&entity.File{}).Error
}

// ListFileByDirectory 目录内分页（仅 status=1，走 idx_file_app_directory）。
func (r File) ListFileByDirectory(ctx core.Context, app string, directoryID uint32, req *types.PageRequest) ([]*entity.File, uint32, error) {
	db := ctx.DB().Model(&entity.File{}).Where("app = ? AND directory_id = ? AND status = 1", app, directoryID)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := req.Normalize()
	var list []*entity.File
	if err := db.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, uint32(total), nil
}

// ListFileByUser "我的文件"分页（仅 status=1，走 idx_file_app_user）。
func (r File) ListFileByUser(ctx core.Context, app string, userID uint32, req *types.PageRequest) ([]*entity.File, uint32, error) {
	db := ctx.DB().Model(&entity.File{}).Where("app = ? AND user_id = ? AND status = 1", app, userID)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := req.Normalize()
	var list []*entity.File
	if err := db.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, uint32(total), nil
}

// UpdateFile 改名 / 移动（name 非空 / directory_id 非 0 才落列，零值自动跳过；updated_at 由 gorm 自动填）。
func (r File) UpdateFile(ctx core.Context, ent *entity.File) error {
	return ctx.DB().Model(&entity.File{}).Where("id = ?", ent.Id).Updates(ent).Error
}

// MarkDeleteInSubtree 软删：子树（含自身目录）下 status=1 的 file 行标为待删除(2)。前缀谓词必须转义，口径须与 directory 树一致。
func (r File) MarkDeleteInSubtree(ctx core.Context, app string, id uint32, path string) (int64, error) {
	res := ctx.DB().Model(&entity.File{}).
		Where("status = 1 AND directory_id IN (?)",
			ctx.DB().Model(&entity.Directory{}).Select("id").
				Where(`app = ? AND (id = ? OR path LIKE ? ESCAPE '\\')`, app, id, pattern.PrefixPattern(path))).
		UpdateColumn("status", 2)
	return res.RowsAffected, res.Error
}

// DeleteFilesByStatus 后台回收：按状态分批硬删 file 行（ORDER BY id LIMIT limit），返回删除行数。
func (r File) DeleteFilesByStatus(ctx core.Context, status uint8, limit int) (int64, error) {
	res := ctx.DB().Where("status = ?", status).Order("id").Limit(limit).Delete(&entity.File{})
	return res.RowsAffected, res.Error
}

// 编译期断言：*File 实现 repository.File
var _ repository.File = (*File)(nil)
