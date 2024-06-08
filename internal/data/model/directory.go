package model

import (
	"github.com/limes-cloud/kratosx/types"
)

type Directory struct {
	ParentId uint32 `json:"parentId" gorm:"column:parent_id"`
	Name     string `json:"name" gorm:"column:name"`
	Accept   string `json:"accept" gorm:"column:accept"`
	MaxSize  uint32 `json:"maxSize" gorm:"column:max_size"`
	types.BaseModel
}

type DirectoryClosure struct {
	ID       uint32 `json:"id" gorm:"column:id"`
	Parent   uint32 `json:"parent" gorm:"column:parent"`
	Children uint32 `json:"children" gorm:"column:children"`
}
