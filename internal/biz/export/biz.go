package export

import (
	"encoding/json"
	ers "errors"
	"fmt"
	"time"

	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/pkg/crypto"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"

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

// ExportExcel 创建导出表格
func (u *UseCase) ExportExcel(ctx kratosx.Context, req *ExportExcelRequest) (*ExportExcelReply, error) {
	b, _ := json.Marshal(req.Rows)
	sha := crypto.MD5(b)
	export, err := u.repo.GetExportBySha(ctx, sha)
	if err != nil && !ers.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err == nil {
		if export.Status == STATUS_PROGRESS && export.UserId == req.UserId {
			return nil, errors.ExportTaskProcessError()
		}
		// 复制正在进行中的导入数据
		id, err := u.repo.CopyExport(ctx, export, &CopyExportRequest{
			UserId:       req.UserId,
			DepartmentId: req.DepartmentId,
			Scene:        req.Scene,
			Name:         req.Name,
		})
		if err != nil {
			return nil, err
		}
		return &ExportExcelReply{Id: id, Sha: sha}, nil
	}

	src := fmt.Sprintf("%s.xlsx", sha)
	id, err := u.repo.CreateExport(ctx, &Export{
		UserId:       req.UserId,
		DepartmentId: req.DepartmentId,
		Scene:        req.Scene,
		Name:         req.Name,
		Sha:          sha,
		Src:          src,
		Status:       STATUS_PROGRESS,
	})
	if err != nil {
		return nil, errors.DatabaseError(err.Error())
	}

	go func() {
		kCtx := ctx.Clone()
		size, err := u.repo.ExportExcel(kCtx, src, req.Rows)
		exp := &Export{
			Id:        id,
			Status:    STATUS_COMPLETED,
			Size:      size,
			ExpiredAt: time.Now().Unix() + int64(u.conf.Export.Expire.Seconds()),
		}
		if err != nil {
			exp.Status = STATUS_FAIL
			exp.Reason = proto.String(err.Error())
		}

		if err := u.repo.UpdateExport(kCtx, exp); err != nil {
			ctx.Logger().Errorw("msg", "update export status error", "err", err.Error())
		}
	}()

	return &ExportExcelReply{Id: id, Sha: sha, Src: src}, nil
}

// ExportFile 创建导出表格
func (u *UseCase) ExportFile(ctx kratosx.Context, req *ExportFileRequest) (*ExportFileReply, error) {
	b, _ := json.Marshal(req.Files)
	sha := crypto.MD5(b)
	export, err := u.repo.GetExportBySha(ctx, sha)
	if err != nil && !ers.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err == nil {
		if export.Status == STATUS_PROGRESS && export.UserId == req.UserId {
			return nil, errors.ExportTaskProcessError()
		}
		// 复制正在进行中的导入数据
		id, err := u.repo.CopyExport(ctx, export, &CopyExportRequest{
			UserId:       req.UserId,
			DepartmentId: req.DepartmentId,
			Scene:        req.Scene,
			Name:         req.Name,
		})
		if err != nil {
			return nil, err
		}
		return &ExportFileReply{Id: id}, nil
	}

	if len(req.Ids) != 0 {
		for _, id := range req.Ids {
			key, err := u.repo.GetExportFileKeyById(ctx, id)
			if err != nil {
				return nil, errors.DatabaseError(err.Error())
			}
			req.Files = append(req.Files, &ExportFileItem{Value: key})
		}
	}

	src := fmt.Sprintf("%s.zip", sha)
	id, err := u.repo.CreateExport(ctx, &Export{
		UserId:       req.UserId,
		DepartmentId: req.DepartmentId,
		Scene:        req.Scene,
		Name:         req.Name,
		Sha:          sha,
		Src:          src,
		Status:       STATUS_PROGRESS,
	})
	if err != nil {
		return nil, errors.DatabaseError(err.Error())
	}

	go func() {
		kCtx := ctx.Clone()
		size, err := u.repo.ExportFile(kCtx, src, req.Files)
		exp := &Export{
			Id:        id,
			Status:    STATUS_COMPLETED,
			Size:      size,
			ExpiredAt: time.Now().Unix() + int64(u.conf.Export.Expire.Seconds()),
		}
		if err != nil {
			exp.Status = STATUS_FAIL
			exp.Reason = proto.String(err.Error())
		}

		if err := u.repo.UpdateExport(kCtx, exp); err != nil {
			ctx.Logger().Errorw("msg", "update export status error", "err", err.Error())
		}
	}()

	return &ExportFileReply{Id: id}, nil
}

// DeleteExport 删除导出信息
func (u *UseCase) DeleteExport(ctx kratosx.Context, ids []uint32) (uint32, error) {
	total, err := u.repo.DeleteExport(ctx, ids)
	if err != nil {
		return 0, errors.DeleteError(err.Error())
	}
	return total, nil
}

// GetExport 获取指定的导出信息
func (u *UseCase) GetExport(ctx kratosx.Context, req *GetExportRequest) (*Export, error) {
	var (
		res *Export
		err error
	)

	if req.Id != nil {
		res, err = u.repo.GetExport(ctx, *req.Id)
	} else if req.Sha != nil {
		res, err = u.repo.GetExportBySha(ctx, *req.Sha)
	} else {
		return nil, errors.ParamsError()
	}

	if err != nil {
		return nil, errors.GetError(err.Error())
	}
	return res, nil
}

// VerifyURL 验证url
func (u *UseCase) VerifyURL(key, expire, sign string) error {
	return u.repo.VerifyURL(key, expire, sign)
}
