package entity

import (
	"github.com/limes-cloud/kratosx/types"
)

type Directory struct {
	ParentId uint32       `json:"parentId" gorm:"column:parent_id"`
	Name     string       `json:"name" gorm:"column:name"`
	Accept   string       `json:"accept" gorm:"column:accept"`
	MaxSize  uint32       `json:"maxSize" gorm:"column:max_size"`
	Children []*Directory `json:"children" gorm:"-"`
	types.BaseModel
}

type DirectoryClosure struct {
	ID       uint32 `json:"id" gorm:"column:id"`
	Parent   uint32 `json:"parent" gorm:"column:parent"`
	Children uint32 `json:"children" gorm:"column:children"`
}

type DirectoryLimit struct {
	DirectoryId uint32   `json:"directoryId"`
	Accepts     []string `json:"accepts"`
	MaxSize     uint32   `json:"maxSize"`
}

// ID 获取菜单树ID
func (m *Directory) ID() uint32 {
	return m.Id
}

// Parent 获取父ID
func (m *Directory) Parent() uint32 {
	return m.ParentId
}

// AppendChildren 添加子节点
func (m *Directory) AppendChildren(child *Directory) {
	m.Children = append(m.Children, child)
}

// ChildrenNode 获取子节点
func (m *Directory) ChildrenNode() []*Directory {
	var list []*Directory
	list = append(list, m.Children...)
	return list
}
