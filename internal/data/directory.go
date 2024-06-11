package data

import (
	"errors"
	"fmt"

	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/pkg/valx"
	"google.golang.org/protobuf/proto"

	biz "github.com/limes-cloud/resource/internal/biz/directory"
	"github.com/limes-cloud/resource/internal/data/model"
)

type directoryRepo struct {
}

func NewDirectoryRepo() biz.Repo {
	return &directoryRepo{}
}

// ToDirectoryEntity model转entity
func (r directoryRepo) ToDirectoryEntity(m *model.Directory) *biz.Directory {
	e := &biz.Directory{}
	_ = valx.Transform(m, e)
	return e
}

// ToDirectoryModel entity转model
func (r directoryRepo) ToDirectoryModel(e *biz.Directory) *model.Directory {
	m := &model.Directory{}
	_ = valx.Transform(e, m)
	return m
}

// GetDirectory 获取指定的数据
func (r directoryRepo) GetDirectory(ctx kratosx.Context, id uint32) (*biz.Directory, error) {
	var (
		m  = model.Directory{}
		fs = []string{"*"}
	)
	db := ctx.DB().Select(fs)
	if err := db.First(&m, id).Error; err != nil {
		return nil, err
	}

	return r.ToDirectoryEntity(&m), nil
}

// ListDirectory 获取列表
func (r directoryRepo) ListDirectory(ctx kratosx.Context, req *biz.ListDirectoryRequest) ([]*biz.Directory, uint32, error) {
	var (
		bs    []*biz.Directory
		ms    []*model.Directory
		total int64
		fs    = []string{"*"}
	)

	db := ctx.DB().Model(model.Directory{}).Select(fs)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

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
		bs = append(bs, r.ToDirectoryEntity(m))
	}
	return bs, uint32(total), nil
}

// CreateDirectory 创建数据
func (r directoryRepo) CreateDirectory(ctx kratosx.Context, req *biz.Directory) (uint32, error) {
	m := r.ToDirectoryModel(req)
	return m.Id, ctx.Transaction(func(ctx kratosx.Context) error {
		if err := ctx.DB().Create(m).Error; err != nil {
			return err
		}
		if m.ParentId != 0 {
			return r.appendDirectoryChildren(ctx, req.ParentId, m.Id)
		}
		return nil
	})
}

// UpdateDirectory 更新数据
func (r directoryRepo) UpdateDirectory(ctx kratosx.Context, req *biz.Directory) error {
	if req.Id == req.ParentId {
		return errors.New("父级不能为自己")
	}
	old, err := r.GetDirectory(ctx, req.Id)
	if err != nil {
		return err
	}

	return ctx.Transaction(func(ctx kratosx.Context) error {
		if old.ParentId != req.ParentId {
			if err := r.removeDirectoryParent(ctx, req.Id); err != nil {
				return err
			}
			if err := r.appendDirectoryChildren(ctx, req.ParentId, req.Id); err != nil {
				return err
			}
		}
		return ctx.DB().Updates(r.ToDirectoryModel(req)).Error
	})
}

// DeleteDirectory 删除数据
func (r directoryRepo) DeleteDirectory(ctx kratosx.Context, ids []uint32) (uint32, error) {
	var del []uint32
	for _, id := range ids {
		del = append(del, id)
		childrenIds, err := r.GetDirectoryChildrenIds(ctx, id)
		if err != nil {
			return 0, err
		}
		del = append(del, childrenIds...)
	}
	db := ctx.DB().Where("id in ?", del).Delete(&model.Directory{})
	return uint32(db.RowsAffected), db.Error
}

// GetDirectoryChildrenIds 获取指定id的所有子id
func (r directoryRepo) GetDirectoryChildrenIds(ctx kratosx.Context, id uint32) ([]uint32, error) {
	var ids []uint32
	return ids, ctx.DB().Model(model.DirectoryClosure{}).
		Select("children").
		Where("parent=?", id).
		Scan(&ids).Error
}

// GetDirectoryParentIds 获取指定id的所有父id
func (r directoryRepo) GetDirectoryParentIds(ctx kratosx.Context, id uint32) ([]uint32, error) {
	var ids []uint32
	return ids, ctx.DB().Model(model.DirectoryClosure{}).
		Select("parent").
		Where("children=?", id).
		Scan(&ids).Error
}

// appendDirectoryChildren 添加id到指定的父id下
func (r directoryRepo) appendDirectoryChildren(ctx kratosx.Context, pid uint32, id uint32) error {
	list := []*model.DirectoryClosure{
		{
			Parent:   pid,
			Children: id,
		},
	}
	ids, _ := r.GetDirectoryParentIds(ctx, pid)
	for _, item := range ids {
		list = append(list, &model.DirectoryClosure{
			Parent:   item,
			Children: id,
		})
	}
	return ctx.DB().Create(&list).Error
}

// removeDirectoryParent 删除指定id的所有父层级
func (r directoryRepo) removeDirectoryParent(ctx kratosx.Context, id uint32) error {
	return ctx.DB().Delete(&model.DirectoryClosure{}, "children=?", id).Error
}
