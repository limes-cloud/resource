package repository

import (
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/types"
)

// File 业务文件条目仓储接口。
type File interface {
	// CreateFile 新建文件；撞 uidx_file_upload_id 即"完成已成交"，调用方重查复用赢家。
	CreateFile(ctx core.Context, ent *entity.File) (uint32, error)
	// GetFile 按主键查正常文件（status=1；待删除视为不存在）。
	GetFile(ctx core.Context, id uint32) (*entity.File, error)
	// GetFileByUploadId 按 upload_id 查文件（幂等收口 / 召回复用）。
	GetFileByUploadId(ctx core.Context, uploadID string) (*entity.File, error)
	// ListFileByDirectory 目录内分页（仅 status=1，走 idx_file_app_directory）。
	ListFileByDirectory(ctx core.Context, app string, directoryID uint32, req *types.PageRequest) ([]*entity.File, uint32, error)
	// ListFileByUser "我的文件"分页（仅 status=1，走 idx_file_app_user）。
	ListFileByUser(ctx core.Context, app string, userID uint32, req *types.PageRequest) ([]*entity.File, uint32, error)
	// UpdateFile 改名 / 移动。
	UpdateFile(ctx core.Context, ent *entity.File) error
	// DeleteFile 删除文件行（单文件删除走此硬删）。
	DeleteFile(ctx core.Context, id uint32) error
	// MarkDeleteInSubtree 软删：把子树（含自身目录）下 status=1 的 file 行标为待删除(2)，返回标记行数。
	MarkDeleteInSubtree(ctx core.Context, app string, id uint32, path string) (int64, error)
	// DeleteFilesByStatus 后台回收：按状态分批硬删 file 行（ORDER BY id LIMIT），返回删除行数。
	DeleteFilesByStatus(ctx core.Context, status uint8, limit int) (int64, error)
}
