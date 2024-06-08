package export

import (
	"github.com/limes-cloud/kratosx"
)

type Repo interface {
	// ListExport 获取导出信息列表
	ListExport(ctx kratosx.Context, req *ListExportRequest) ([]*Export, uint32, error)

	// CreateExport 创建导出信息
	CreateExport(ctx kratosx.Context, req *Export) (uint32, error)

	// DeleteExport 删除导出信息
	DeleteExport(ctx kratosx.Context, ids []uint32) (uint32, error)
}
