package export

import (
	"github.com/limes-cloud/kratosx"

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

// ListExport 获取导出信息列表
func (u *UseCase) ListExport(ctx kratosx.Context, req *ListExportRequest) ([]*Export, uint32, error) {
	list, total, err := u.repo.ListExport(ctx, req)
	if err != nil {
		return nil, 0, errors.ListError(err.Error())
	}
	return list, total, nil
}

// CreateExport 创建导出信息
func (u *UseCase) CreateExport(ctx kratosx.Context, req *Export) (uint32, error) {
	id, err := u.repo.CreateExport(ctx, req)
	if err != nil {
		return 0, errors.CreateError(err.Error())
	}
	return id, nil
}

// DeleteExport 删除导出信息
func (u *UseCase) DeleteExport(ctx kratosx.Context, ids []uint32) (uint32, error) {
	total, err := u.repo.DeleteExport(ctx, ids)
	if err != nil {
		return 0, errors.DeleteError(err.Error())
	}
	return total, nil
}
