package app

import (
	"context"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/limes-cloud/resource/api/export"
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/infra/dbs"
	"github.com/limes-cloud/resource/internal/infra/store"

	"github.com/limes-cloud/kratosx/pkg/value"

	"github.com/limes-cloud/resource/api/errors"
	"github.com/limes-cloud/resource/internal/domain/service"
	"github.com/limes-cloud/resource/internal/types"
)

type Export struct {
	export.UnimplementedExportServer
	srv *service.Export
}

func NewExport() *Export {
	return &Export{
		srv: service.NewExport(dbs.NewExport(), dbs.NewFile(), store.NewStore()),
	}
}

func init() {
	register(func(hs *http.Server, gs *grpc.Server) {
		app := NewExport()
		export.RegisterExportHTTPServer(hs, app)
		export.RegisterExportServer(gs, app)

		//cr := hs.Route("/")
		//cr.GET("/resource/api/v1/download/{expire}/{sign}/{src}", app.srv.Download())
		//cr.GET("/resource/api/v1/download/{src}", app.srv.Download())
		//
		//cr.GET("/resource/api/v1/target", app.srv.DownloadTarget())
	})
}

// ListExport 获取导出信息列表
func (s *Export) ListExport(c context.Context, req *export.ListExportRequest) (*export.ListExportReply, error) {
	list, total, err := s.srv.ListExport(core.MustContext(c), &types.ListExportRequest{
		Page:     req.Page,
		PageSize: req.PageSize,
		Order:    req.Order,
		OrderBy:  req.OrderBy,
	})
	if err != nil {
		return nil, err
	}

	reply := export.ListExportReply{Total: total}
	for _, item := range list {
		reply.List = append(reply.List, &export.ListExportReply_Export{
			Id:        item.Id,
			Name:      item.Name,
			Size:      item.Size,
			Sha:       item.Sha,
			Key:       item.Key,
			Status:    item.Status,
			Reason:    item.Reason,
			ExpiredAt: uint32(item.ExpiredAt),
			CreatedAt: uint32(item.CreatedAt),
			UpdatedAt: uint32(item.UpdatedAt),
			UserId:    item.UserId,
			DeptId:    item.DeptId,
			TenantId:  item.TenantId,
		})
	}
	return &reply, nil
}

// ExportFile 创建导出文件
func (s *Export) ExportFile(c context.Context, req *export.ExportFileRequest) (*export.ExportFileReply, error) {
	var (
		in  = types.ExportFileRequest{}
		ctx = core.MustContext(c)
	)

	if err := value.Transform(req, &in); err != nil {
		ctx.Logger().Warnw("msg", "req transform err", "err", err.Error())
		return nil, errors.TransformError()
	}

	res, err := s.srv.ExportFile(ctx, &in)
	if err != nil {
		return nil, err
	}

	return &export.ExportFileReply{Id: res.Id}, nil
}

// ExportExcel 创建导出excel文件
func (s *Export) ExportExcel(c context.Context, req *export.ExportExcelRequest) (*export.ExportExcelReply, error) {
	var in = types.ExportExcelRequest{
		Name:    req.Name,
		Headers: req.Headers,
	}

	for _, row := range req.Rows {
		var temp []*types.ExportExcelCol
		for _, col := range row.Cols {
			temp = append(temp, &types.ExportExcelCol{
				Type:  col.Type,
				Value: col.Value,
			})
		}
		in.Rows = append(in.Rows, temp)
	}

	var files []*types.ExportFileItem
	for _, item := range req.Files {
		files = append(files, &types.ExportFileItem{
			Value:  item.Value,
			Rename: item.Rename,
		})
	}
	in.Files = files

	res, err := s.srv.ExportExcel(core.MustContext(c), &in)
	if err != nil {
		return nil, err
	}

	return &export.ExportExcelReply{Id: res.Id}, nil
}

// DeleteExport 删除导出信息
func (s *Export) DeleteExport(c context.Context, req *export.DeleteExportRequest) (*export.DeleteExportReply, error) {
	total, err := s.srv.DeleteExport(core.MustContext(c), req.Ids)
	if err != nil {
		return nil, err
	}
	return &export.DeleteExportReply{Total: total}, nil
}

// GetExport 获取指定的导出信息
func (s *Export) GetExport(c context.Context, req *export.GetExportRequest) (*export.GetExportReply, error) {
	result, err := s.srv.GetExport(core.MustContext(c), &types.GetExportRequest{
		Id:  req.Id,
		Sha: req.Sha,
	})
	if err != nil {
		return nil, err
	}

	return &export.GetExportReply{
		Id:        result.Id,
		Name:      result.Name,
		Size:      result.Size,
		Sha:       result.Sha,
		Key:       result.Key,
		Status:    result.Status,
		Reason:    result.Reason,
		ExpiredAt: uint32(result.ExpiredAt),
		CreatedAt: uint32(result.CreatedAt),
		UpdatedAt: uint32(result.UpdatedAt),
	}, nil
}
