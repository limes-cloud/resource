package export

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/pkg/util"
	"github.com/limes-cloud/kratosx/types"
	"github.com/limes-cloud/manager/api/auth"

	"github.com/limes-cloud/resource/api/errors"
	"github.com/limes-cloud/resource/internal/biz/file"
	"github.com/limes-cloud/resource/internal/config"
)

type UseCase struct {
	config   *config.Config
	repo     Repo
	fileRepo file.Repo
	factory  Factory
}

func NewUseCase(config *config.Config, repo Repo, factory Factory) *UseCase {
	return &UseCase{config: config, repo: repo, factory: factory}
}

func (u *UseCase) PageExport(ctx kratosx.Context, in *PageExportRequest) ([]*Export, uint32, error) {
	info, err := auth.Get(ctx)
	if err != nil {
		return nil, 0, err
	}
	in.UserId = info.UserId

	list, total, err := u.repo.PageExport(ctx, in)
	if err != nil {
		return nil, 0, errors.Database()
	}
	for ind, item := range list {
		list[ind].Src = u.factory.ExportFileSrc(item.Src)
	}
	return list, total, nil
}

func (u *UseCase) DeleteExport(ctx kratosx.Context, id uint32) error {
	info, err := auth.Get(ctx)
	if err != nil {
		return err
	}
	exp, err := u.repo.GetExport(ctx, id)
	if err != nil {
		return errors.NotFound()
	}
	if err := u.repo.DeleteExport(ctx, info.UserId, id); err != nil {
		return errors.Database()
	}
	_ = os.Remove(u.config.Export.LocalDir + exp.Src)
	return nil
}

func (u *UseCase) AddExport(ctx kratosx.Context, in *AddExportRequest) (uint32, error) {
	info, err := auth.Get(ctx)
	if err != nil {
		return 0, err
	}

	b, _ := json.Marshal(in)
	version := util.MD5(b)

	exp, err := u.repo.GetExportByVersion(ctx, info.UserId, version)
	src := fmt.Sprintf("%s.zip", version)
	var id uint32
	if err == nil {
		if exp.Status == StatusProcess {
			return 0, errors.ExportTaskProcess()
		}
		if exp.Status == StatusFinish {
			return exp.ID, nil
		}
		exp.Status = StatusProcess
		if err := u.repo.UpdateExport(ctx, exp); err != nil {
			ctx.Logger().Errorw("msg", "update export status error", "err", err.Error())
		}
		id = exp.ID
	} else {
		id, err = u.repo.AddExport(ctx, &Export{
			UserId:  info.UserId,
			Src:     src,
			Name:    in.Name,
			Version: version,
			Status:  StatusProcess,
		})
		if err != nil {
			return 0, errors.DatabaseFormat(err.Error())
		}
	}

	in.Name = u.config.Export.LocalDir + "/" + src
	go func() {
		kCtx := ctx.Clone()
		size, err := u.factory.ExportFile(kCtx, in)
		exp := &Export{
			BaseModel: types.BaseModel{ID: id},
			Status:    StatusFinish,
			Size:      uint32(size),
			Version:   version,
		}
		if err != nil {
			exp.Status = StatusFail
			exp.Reason = err.Error()
		}

		if err := u.repo.UpdateExport(kCtx, exp); err != nil {
			ctx.Logger().Errorw("msg", "update export status error", "err", err.Error())
		}
	}()

	return id, nil
}

func (u *UseCase) AddExportExcel(ctx kratosx.Context, in *AddExportExcelRequest) (uint32, error) {
	info, err := auth.Get(ctx)
	if err != nil {
		return 0, err
	}

	b, _ := json.Marshal(in)
	version := util.MD5(b)

	exp, err := u.repo.GetExportByVersion(ctx, info.UserId, version)
	src := fmt.Sprintf("%s.xlsx", version)
	var id uint32
	if err == nil {
		if exp.Status == StatusProcess {
			return 0, errors.ExportTaskProcess()
		}
		if exp.Status == StatusFinish {
			return exp.ID, nil
		}
		exp.Status = StatusProcess
		if err := u.repo.UpdateExport(ctx, exp); err != nil {
			ctx.Logger().Errorw("msg", "update export status error", "err", err.Error())
		}
		id = exp.ID
	} else {
		id, err = u.repo.AddExport(ctx, &Export{
			UserId:  info.UserId,
			Src:     src,
			Name:    in.Name,
			Version: version,
			Status:  StatusProcess,
		})
		if err != nil {
			return 0, errors.DatabaseFormat(err.Error())
		}
	}

	in.Name = u.config.Export.LocalDir + "/" + src
	go func() {
		kCtx := ctx.Clone()
		size, err := u.factory.ExportExcel(kCtx, in)
		exp := &Export{
			BaseModel: types.BaseModel{ID: id},
			Status:    StatusFinish,
			Size:      uint32(size),
		}
		if err != nil {
			exp.Status = StatusFail
			exp.Reason = err.Error()
		}

		if err := u.repo.UpdateExport(kCtx, exp); err != nil {
			ctx.Logger().Errorw("msg", "update export status error", "err", err.Error())
		}
	}()

	return id, nil
}
