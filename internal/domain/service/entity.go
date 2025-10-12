package service

import (
	"github.com/limes-cloud/kratosx/library/db"
	"github.com/limes-cloud/kratosx/pkg/value"
	"github.com/limes-cloud/resource/internal/core"
)

type Entity struct {
}

func NewEntity() *Entity {
	return &Entity{}
}

// LoadEntity 获取租户列表
func (u *Entity) LoadEntity(ctx core.Context) ([]*db.Entity, error) {
	selectTable := []string{
		"export",
		"user_file",
		"directory",
	}
	filterColumn := []string{
		"tenant_id",
		"deleted_at",
	}

	var list []*db.Entity
	// 加载数据库的全部信息
	res := ctx.Database().Entities()
	for _, item := range res {
		if value.InList(selectTable, item.Name) {
			ent := &db.Entity{
				Database: item.Database,
				Name:     item.Name,
				Comment:  item.Comment,
			}

			var fields []db.Field
			// 过滤字段
			for _, field := range item.Fields {
				if value.InList(filterColumn, field.Name) {
					continue
				}
				fields = append(fields, db.Field{
					Name:    field.Name,
					Comment: field.Comment,
				})
			}

			ent.Fields = fields
			list = append(list, ent)
		}
	}

	return list, nil
}
