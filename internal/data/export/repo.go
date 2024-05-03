package export

import (
	"github.com/limes-cloud/kratosx"

	"github.com/limes-cloud/resource/internal/biz/export"
)

type repo struct {
}

func NewRepo() export.Repo {
	return &repo{}
}

func (r repo) GetExport(ctx kratosx.Context, id uint32) (*export.Export, error) {
	dir := export.Export{}
	return &dir, ctx.DB().First(&dir, "id=?", id).Error
}

func (r repo) GetExportByVersion(ctx kratosx.Context, uid uint32, version string) (*export.Export, error) {
	dir := export.Export{}
	return &dir, ctx.DB().First(&dir, "version=? and user_id=?", version, uid).Error
}

func (r repo) PageExport(ctx kratosx.Context, in *export.PageExportRequest) ([]*export.Export, uint32, error) {
	var list []*export.Export
	total := int64(0)

	db := ctx.DB().Model(export.Export{})
	if err := db.Count(&total).Error; err != nil {
		return nil, uint32(total), err
	}

	db = db.Offset(int((in.Page - 1) * in.PageSize)).Limit(int(in.PageSize))

	return list, uint32(total), db.Find(&list).Error
}

func (r repo) AddExport(ctx kratosx.Context, export *export.Export) (uint32, error) {
	return export.ID, ctx.DB().Create(export).Error
}

func (r repo) UpdateExport(ctx kratosx.Context, export *export.Export) error {
	return ctx.DB().Where("id=?", export.ID).Updates(export).Error
}

func (r repo) UpdateExportExpire(ctx kratosx.Context, t int64) error {
	return ctx.DB().Where("created_at <= ?", t).Update("status", export.StatusExpire).Error
}

func (r repo) DeleteExport(ctx kratosx.Context, uid uint32, id uint32) error {
	return ctx.DB().Where("id=? and user_id=?", id, uid).Delete(export.Export{}).Error
}
