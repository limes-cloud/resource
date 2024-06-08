package directory

import (
	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/pkg/tree"
	"github.com/limes-cloud/resource/api/resource/errors"
	"github.com/limes-cloud/resource/internal/conf"
)

type UseCase struct {
	conf *conf.Config
	repo Repo
}

func NewUseCase(config *conf.Config, repo Repo) *UseCase {
	return &UseCase{conf: config, repo: repo}
}

// GetDirectory 获取指定的文件目录信息
func (u *UseCase) GetDirectory(ctx kratosx.Context, req *GetDirectoryRequest) (*Directory, error) {
	var (
		res *Directory
		err error
	)

	if req.Id != nil {
		res, err = u.repo.GetDirectory(ctx, *req.Id)
	} else {
		return nil, errors.ParamsError()
	}

	if err != nil {
		return nil, errors.GetError(err.Error())
	}
	return res, nil
}

// ListDirectory 获取文件目录信息列表树
func (u *UseCase) ListDirectory(ctx kratosx.Context, req *ListDirectoryRequest) ([]tree.Tree, uint32, error) {
	list, total, err := u.repo.ListDirectory(ctx, req)
	if err != nil {
		return nil, 0, errors.ListError(err.Error())
	}
	var ts []tree.Tree
	for _, item := range list {
		ts = append(ts, item)
	}
	return tree.BuildArrayTree(ts), total, nil
}

// CreateDirectory 创建文件目录信息
func (u *UseCase) CreateDirectory(ctx kratosx.Context, req *Directory) (uint32, error) {
	id, err := u.repo.CreateDirectory(ctx, req)
	if err != nil {
		return 0, errors.CreateError(err.Error())
	}
	return id, nil
}

// UpdateDirectory 更新文件目录信息
func (u *UseCase) UpdateDirectory(ctx kratosx.Context, req *Directory) error {
	if err := u.repo.UpdateDirectory(ctx, req); err != nil {
		return errors.UpdateError(err.Error())
	}
	return nil
}

// DeleteDirectory 删除文件目录信息
func (u *UseCase) DeleteDirectory(ctx kratosx.Context, ids []uint32) (uint32, error) {
	total, err := u.repo.DeleteDirectory(ctx, ids)
	if err != nil {
		return 0, errors.DeleteError(err.Error())
	}
	return total, nil
}
