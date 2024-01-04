package migrate

import (
	"github.com/limes-cloud/kratosx"
	gte "github.com/limes-cloud/kratosx/library/db/gormtranserror"

	"github.com/limes-cloud/resource/config"
	"github.com/limes-cloud/resource/internal/model"
	"github.com/limes-cloud/resource/pkg/store/local"
)

func IsInit(ctx kratosx.Context) bool {
	db := ctx.DB().Migrator()
	return db.HasTable(model.Directory{}) &&
		db.HasTable(model.File{}) &&
		db.HasTable(local.Chunk{})
}

func Init(ctx kratosx.Context, config *config.Config) {
	db := ctx.DB()
	_ = db.Set("gorm:table_options", "COMMENT='目录信息' ENGINE=InnoDB CHARSET=utf8mb4").AutoMigrate(model.Directory{})
	_ = db.Set("gorm:table_options", "COMMENT='文件信息' ENGINE=InnoDB CHARSET=utf8mb4").AutoMigrate(model.File{})
	_ = db.Set("gorm:table_options", "COMMENT='切片信息' ENGINE=InnoDB CHARSET=utf8mb4").AutoMigrate(local.Chunk{})
	// 重新载入gorm错误插件
	gte.NewGlobalGormErrorPlugin().Initialize(ctx.DB())
}
