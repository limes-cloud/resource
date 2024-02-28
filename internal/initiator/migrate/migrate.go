package migrate

import (
	"github.com/limes-cloud/kratosx"
	gte "github.com/limes-cloud/kratosx/library/db/gormtranserror"

	biz "github.com/limes-cloud/resource/internal/biz/file"
	"github.com/limes-cloud/resource/internal/pkg/store/local"
)

func Run(ctx kratosx.Context) {
	db := ctx.DB()
	_ = db.Set("gorm:table_options", "COMMENT='目录信息' ENGINE=InnoDB CHARSET=utf8mb4").AutoMigrate(biz.Directory{})
	_ = db.Set("gorm:table_options", "COMMENT='文件信息' ENGINE=InnoDB CHARSET=utf8mb4").AutoMigrate(biz.File{})
	_ = db.Set("gorm:table_options", "COMMENT='切片信息' ENGINE=InnoDB CHARSET=utf8mb4").AutoMigrate(local.Chunk{})
	// 重新载入gorm错误插件
	_ = gte.NewGlobalGormErrorPlugin().Initialize(ctx.DB())
}
