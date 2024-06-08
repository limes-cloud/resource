package data

import (
	"fmt"

	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/pkg/valx"
	"google.golang.org/protobuf/proto"

	biz "github.com/limes-cloud/resource/internal/biz/export"
	"github.com/limes-cloud/resource/internal/data/model"
)

type exportRepo struct {
}

func NewExportRepo() biz.Repo {
	return &exportRepo{}
}

// ToExportEntity model转entity
func (r exportRepo) ToExportEntity(m *model.Export) *biz.Export {
	e := &biz.Export{}
	_ = valx.Transform(m, e)
	return e
}

// ToExportModel entity转model
func (r exportRepo) ToExportModel(e *biz.Export) *model.Export {
	m := &model.Export{}
	_ = valx.Transform(e, m)
	return m
}

// ListExport 获取列表
func (r exportRepo) ListExport(ctx kratosx.Context, req *biz.ListExportRequest) ([]*biz.Export, uint32, error) {
	var (
		bs    []*biz.Export
		ms    []*model.Export
		total int64
		fs    = []string{"*"}
	)

	db := ctx.DB().Model(model.Export{}).Select(fs)

	if req.UserId != nil {
		db = db.Where("user_id = ?", *req.UserId)
	}
	if req.DepartmentId != nil {
		db = db.Where("department_id = ?", *req.DepartmentId)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	db = db.Offset(int((req.Page - 1) * req.PageSize)).Limit(int(req.PageSize))

	if req.OrderBy == nil || *req.OrderBy == "" {
		req.OrderBy = proto.String("id")
	}
	if req.Order == nil || *req.Order == "" {
		req.Order = proto.String("asc")
	}
	db = db.Order(fmt.Sprintf("%s %s", *req.OrderBy, *req.Order))
	if *req.OrderBy != "id" {
		db = db.Order("id asc")
	}

	if err := db.Find(&ms).Error; err != nil {
		return nil, 0, err
	}

	for _, m := range ms {
		bs = append(bs, r.ToExportEntity(m))
	}
	return bs, uint32(total), nil
}

// CreateExport 创建数据
func (r exportRepo) CreateExport(ctx kratosx.Context, req *biz.Export) (uint32, error) {
	m := r.ToExportModel(req)
	return m.Id, ctx.DB().Create(m).Error
}

// DeleteExport 删除数据
func (r exportRepo) DeleteExport(ctx kratosx.Context, ids []uint32) (uint32, error) {
	db := ctx.DB().Where("id in ?", ids).Delete(&model.Export{})
	return uint32(db.RowsAffected), db.Error
}
