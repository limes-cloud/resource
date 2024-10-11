package repository

import (
	"github.com/limes-cloud/kratosx"

	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/types"
)

type Export interface {
	// CreateExport 新增导出信息
	CreateExport(ctx kratosx.Context, export *entity.Export) (uint32, error)

	// ListExport 获取导出信息列表
	ListExport(ctx kratosx.Context, req *types.ListExportRequest) ([]*entity.Export, uint32, error)

	// DeleteExport 删除导出信息
	DeleteExport(ctx kratosx.Context, ids []uint32) (uint32, error)

	// GetExport 获取指定的导出信息
	GetExport(ctx kratosx.Context, id uint32) (*entity.Export, error)

	// CopyExport 获取指定的导出信息
	CopyExport(ctx kratosx.Context, export *entity.Export, req *types.CopyExportRequest) (uint32, error)

	// UpdateExport 更新导出信息
	UpdateExport(ctx kratosx.Context, req *entity.Export) error

	// GetExportBySha 获取指定的导出信息
	GetExportBySha(ctx kratosx.Context, sha string) (*entity.Export, error)

	// GetExportFileCount 获取导出文件数量
	GetExportFileCount(ctx kratosx.Context, req *types.GetExportFileCountRequest) (int64, error)
}
