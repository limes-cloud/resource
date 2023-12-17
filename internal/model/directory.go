package model

import "github.com/limes-cloud/kratosx"

type Directory struct {
	BaseModel
	ParentID uint32 `json:"parent_id"`
	Name     string `json:"name"`
	App      string `json:"app"`
}

// Create 创建目录信息
func (u *Directory) Create(ctx kratosx.Context) error {
	return ctx.DB().Model(u).Create(u).Error
}

// OneByID 获取目录信息
func (u *Directory) OneByID(ctx kratosx.Context, id uint32) error {
	return ctx.DB().Model(u).First(u, "id=?", id).Error
}

// OneByName 获取目录信息
func (u *Directory) OneByName(ctx kratosx.Context, id uint32, name string) error {
	return ctx.DB().Model(u).First(u, "parent_id=? and name=?", id, name).Error
}

// OneByPaths 获取目录信息
func (u *Directory) OneByPaths(ctx kratosx.Context, app string, paths []string) error {
	parent := uint32(0)
	for _, path := range paths {
		nd := &Directory{}
		if err := ctx.DB().Where(Directory{
			App:      app,
			ParentID: parent,
			Name:     path,
		}).FirstOrCreate(nd).Error; err != nil {
			return err
		}
		parent = nd.ID
		*u = *nd
	}
	return nil
}

// Update 更新目录信息
func (u *Directory) Update(ctx kratosx.Context) error {
	return ctx.DB().Model(u).Updates(&u).Error
}

// DeleteByID 通过id删除目录信息
func (u *Directory) DeleteByID(ctx kratosx.Context, id uint32) error {
	return ctx.DB().Where("id=?", id).Delete(Directory{}).Error
}

// AllByParentID 通过id查询目录信息
func (u *Directory) AllByParentID(ctx kratosx.Context, app string, id uint32) ([]*Directory, error) {
	var list []*Directory
	return list, ctx.DB().Model(u).Find(&list, "app=? and parent_id=?", app, id).Error
}
