package dbs

import (
	"errors"
	"github.com/limes-cloud/resource/internal/core"
	"strings"
	"sync"

	"github.com/limes-cloud/resource/internal/domain/entity"
)

type Directory struct {
}

var (
	directoryIns  *Directory
	directoryOnce sync.Once
)

func NewDirectory() *Directory {
	directoryOnce.Do(func() {
		directoryIns = &Directory{}
	})
	return directoryIns
}

// GetDirectory 获取指定的数据
func (r Directory) GetDirectory(ctx core.Context, id uint32) (*entity.Directory, error) {
	var ent = entity.Directory{}
	return &ent, ctx.DB().First(&ent, id).Error
}

// ListDirectory 获取列表
func (r Directory) ListDirectory(ctx core.Context) ([]*entity.Directory, error) {
	var list []*entity.Directory
	return list, ctx.DB().Model(entity.Directory{}).Order("id asc").Find(&list).Error
}

// CreateDirectory 创建数据
func (r Directory) CreateDirectory(ctx core.Context, directory *entity.Directory) (uint32, error) {
	return directory.Id, ctx.Transaction(func(ctx core.Context) error {
		if err := ctx.DB().Create(directory).Error; err != nil {
			return err
		}
		if directory.ParentId != 0 {
			return r.appendDirectoryChildren(ctx, directory.ParentId, directory.Id)
		}
		return nil
	})
}

// UpdateDirectory 更新数据
func (r Directory) UpdateDirectory(ctx core.Context, directory *entity.Directory) error {
	if directory.Id == directory.ParentId {
		return errors.New("父级不能为自己")
	}
	old, err := r.GetDirectory(ctx, directory.Id)
	if err != nil {
		return err
	}

	return ctx.Transaction(func(ctx core.Context) error {
		if old.ParentId != directory.ParentId {
			if err := r.removeDirectoryParent(ctx, directory.Id); err != nil {
				return err
			}
			if err := r.appendDirectoryChildren(ctx, directory.ParentId, directory.Id); err != nil {
				return err
			}
		}
		return ctx.DB().Updates(directory).Error
	})
}

// DeleteDirectory 删除数据
func (r Directory) DeleteDirectory(ctx core.Context, ids []uint32) (uint32, error) {
	var del []uint32
	for _, id := range ids {
		del = append(del, id)
		childrenIds, err := r.GetDirectoryChildrenIds(ctx, id)
		if err != nil {
			return 0, err
		}
		del = append(del, childrenIds...)
	}
	db := ctx.DB().Where("id in ?", del).Delete(&entity.Directory{})
	return uint32(db.RowsAffected), db.Error
}

// GetDirectoryChildrenIds 获取指定id的所有子id
func (r Directory) GetDirectoryChildrenIds(ctx core.Context, id uint32) ([]uint32, error) {
	var ids []uint32
	return ids, ctx.DB().Model(entity.DirectoryClosure{}).
		Select("children").
		Where("parent=?", id).
		Scan(&ids).Error
}

// GetDirectoryParentIds 获取指定id的所有父id
func (r Directory) GetDirectoryParentIds(ctx core.Context, id uint32) ([]uint32, error) {
	var ids []uint32
	return ids, ctx.DB().Model(entity.DirectoryClosure{}).
		Select("parent").
		Where("children=?", id).
		Scan(&ids).Error
}

// appendDirectoryChildren 添加id到指定的父id下
func (r Directory) appendDirectoryChildren(ctx core.Context, pid uint32, id uint32) error {
	list := []*entity.DirectoryClosure{
		{
			Parent:   pid,
			Children: id,
		},
	}
	ids, _ := r.GetDirectoryParentIds(ctx, pid)
	for _, item := range ids {
		list = append(list, &entity.DirectoryClosure{
			Parent:   item,
			Children: id,
		})
	}
	return ctx.DB().Create(&list).Error
}

// removeDirectoryParent 删除指定id的所有父层级
func (r Directory) removeDirectoryParent(ctx core.Context, id uint32) error {
	return ctx.DB().Delete(&entity.DirectoryClosure{}, "children=?", id).Error
}

func (r Directory) GetDirectoryLimitByPath(ctx core.Context, paths []string) (*entity.DirectoryLimit, error) {
	var (
		dir    = entity.Directory{}
		parent = uint32(0)
		conf   = ctx.Config()
	)

	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		nr := entity.Directory{}
		if err := ctx.DB().Where(entity.Directory{
			ParentId: parent,
			Name:     path,
		}).Attrs(entity.Directory{
			Accept:  strings.Join(conf.DefaultAcceptTypes, ","),
			MaxSize: conf.DefaultMaxSize,
		}).FirstOrCreate(&nr).Error; err != nil {
			return nil, err
		}
		parent = nr.Id
		dir = nr
	}

	return &entity.DirectoryLimit{
		DirectoryId: dir.Id,
		Accepts:     strings.Split(dir.Accept, ","),
		MaxSize:     dir.MaxSize,
	}, nil
}

func (r Directory) GetDirectoryLimitById(ctx core.Context, id uint32) (*entity.DirectoryLimit, error) {
	var dir = entity.Directory{}
	if err := ctx.DB().Where("id=?", id).First(&dir).Error; err != nil {
		return nil, err
	}
	return &entity.DirectoryLimit{
		DirectoryId: dir.Id,
		Accepts:     strings.Split(dir.Accept, ","),
		MaxSize:     dir.MaxSize,
	}, nil
}
