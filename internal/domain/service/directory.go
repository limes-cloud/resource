package service

import (
	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/pkg/tree"

	"github.com/limes-cloud/resource/api/resource/errors"
	"github.com/limes-cloud/resource/internal/conf"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/repository"
	"github.com/limes-cloud/resource/internal/types"
)

type Directory struct {
	conf *conf.Config
	repo repository.Directory
}

func NewDirectory(
	conf *conf.Config,
	repo repository.Directory,
) *Directory {
	return &Directory{
		conf: conf,
		repo: repo,
	}
}

// GetDirectory 获取指定的文件目录信息
func (u *Directory) GetDirectory(ctx kratosx.Context, id uint32) (*entity.Directory, error) {
	res, err := u.repo.GetDirectory(ctx, id)
	if err != nil {
		return nil, errors.GetError(err.Error())
	}
	return res, nil
}

// ListDirectory 获取文件目录信息列表树
func (u *Directory) ListDirectory(ctx kratosx.Context, req *types.ListDirectoryRequest) ([]*entity.Directory, uint32, error) {
	list, total, err := u.repo.ListDirectory(ctx, req)
	if err != nil {
		return nil, 0, errors.ListError(err.Error())
	}
	return tree.BuildArrayTree(list), total, nil
}

// CreateDirectory 创建文件目录信息
func (u *Directory) CreateDirectory(ctx kratosx.Context, req *entity.Directory) (uint32, error) {
	id, err := u.repo.CreateDirectory(ctx, req)
	if err != nil {
		return 0, errors.CreateError(err.Error())
	}
	return id, nil
}

// UpdateDirectory 更新文件目录信息
func (u *Directory) UpdateDirectory(ctx kratosx.Context, req *entity.Directory) error {
	if err := u.repo.UpdateDirectory(ctx, req); err != nil {
		return errors.UpdateError(err.Error())
	}
	return nil
}

// DeleteDirectory 删除文件目录信息
func (u *Directory) DeleteDirectory(ctx kratosx.Context, ids []uint32) (uint32, error) {
	total, err := u.repo.DeleteDirectory(ctx, ids)
	if err != nil {
		return 0, errors.DeleteError(err.Error())
	}
	return total, nil
}
