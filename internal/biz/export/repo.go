package export

import (
	"github.com/limes-cloud/kratosx"
)

type Repo interface {
	// ListExport 获取导出信息列表
	ListExport(ctx kratosx.Context, req *ListExportRequest) ([]*Export, uint32, error)

	// CreateExport 创建导出信息
	CreateExport(ctx kratosx.Context, req *Export) (uint32, error)

	// ExportExcel 导出excel信息返回文件大小
	ExportExcel(ctx kratosx.Context, src string, rows [][]*ExportExcelCol) (uint32, error)

	// ExportFile 导出文件信息，并返回文件大小
	ExportFile(ctx kratosx.Context, src string, rows []*ExportFileItem) (uint32, error)

	// DeleteExport 删除导出信息
	DeleteExport(ctx kratosx.Context, ids []uint32) (uint32, error)

	// GetExport 获取指定的导出信息
	GetExport(ctx kratosx.Context, id uint32) (*Export, error)

	// CopyExport 获取指定的导出信息
	CopyExport(ctx kratosx.Context, export *Export, req *CopyExportRequest) (uint32, error)

	// UpdateExport 更新导出信息
	UpdateExport(ctx kratosx.Context, req *Export) error

	// GetExportBySha 获取指定的导出信息
	GetExportBySha(ctx kratosx.Context, sha string) (*Export, error)

	// GetExportFileKeyById 获取导出文件的key
	GetExportFileKeyById(ctx kratosx.Context, id uint32) (string, error)

	// VerifyURL 验证url签名
	VerifyURL(key string, expire string, sign string) error
}
