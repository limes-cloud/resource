package model

import (
	"github.com/limes-cloud/kratos"
)

type Directory struct {
	BaseModel
	ParentID uint32 `json:"parent_id"`
	Name     string `json:"name"`
	App      string `json:"app"`
}

// Create 创建目录信息
func (u *Directory) Create(ctx kratos.Context) error {
	return ctx.DB().Model(u).Create(u).Error
}

// OneByID 获取目录信息
func (u *Directory) OneByID(ctx kratos.Context, id uint32) error {
	return ctx.DB().Model(u).First(u, "id=?", id).Error
}

// OneByName 获取目录信息
func (u *Directory) OneByName(ctx kratos.Context, id uint32, name string) error {
	return ctx.DB().Model(u).First(u, "parent_id=? and name=?", id, name).Error
}

// Update 更新目录信息
func (u *Directory) Update(ctx kratos.Context) error {
	return ctx.DB().Model(u).Updates(&u).Error
}

// DeleteByID 通过id删除目录信息
func (u *Directory) DeleteByID(ctx kratos.Context, id uint32) error {
	return ctx.DB().Where("id=?", id).Delete(Directory{}).Error
}

// AllByParentID 通过id查询目录信息
func (u *Directory) AllByParentID(ctx kratos.Context, app string, id uint32) ([]*Directory, error) {
	var list []*Directory
	return list, ctx.DB().Model(u).Find(&list, "app=? and parent_id=?", app, id).Error
}
