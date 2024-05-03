package export

import (
	"github.com/limes-cloud/kratosx"
)

type Repo interface {
	PageExport(ctx kratosx.Context, req *PageExportRequest) ([]*Export, uint32, error)
	AddExport(ctx kratosx.Context, c *Export) (uint32, error)
	GetExportByVersion(ctx kratosx.Context, uid uint32, version string) (*Export, error)
	GetExport(ctx kratosx.Context, id uint32) (*Export, error)
	UpdateExport(ctx kratosx.Context, c *Export) error
	DeleteExport(ctx kratosx.Context, uid, id uint32) error
	UpdateExportExpire(ctx kratosx.Context, t int64) error
}
