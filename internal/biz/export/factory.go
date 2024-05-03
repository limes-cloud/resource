package export

import (
	"github.com/limes-cloud/kratosx"
)

type Factory interface {
	ExportFileSrc(src string) string
	ExportFile(ctx kratosx.Context, in *AddExportRequest) (int64, error)
	ExportExcel(ctx kratosx.Context, in *AddExportExcelRequest) (int64, error)
}
