package repository

import (
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/types"
)

type Export interface {
	// CreateExport 新增导出信息
	CreateExport(ctx core.Context, export *entity.Export) (uint32, error)

	// ListExport 获取导出信息列表
	ListExport(ctx core.Context, req *types.ListExportRequest) ([]*entity.Export, uint32, error)

	// DeleteExport 删除导出信息
	DeleteExport(ctx core.Context, ids []uint32) (uint32, error)

	// GetExport 获取指定的导出信息
	GetExport(ctx core.Context, id uint32) (*entity.Export, error)

	// CopyExport 获取指定的导出信息
	CopyExport(ctx core.Context, export *entity.Export, req *types.CopyExportRequest) (uint32, error)

	// UpdateExport 更新导出信息
	UpdateExport(ctx core.Context, req *entity.Export) error

	// GetExportBySha 获取指定的导出信息
	GetExportBySha(ctx core.Context, sha string) (*entity.Export, error)

	// GetExportFileCount 获取导出文件数量
	GetExportFileCount(ctx core.Context, req *types.GetExportFileCountRequest) (int64, error)
}
