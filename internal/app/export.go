package app

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/pkg/valx"

	"github.com/limes-cloud/resource/api/resource/errors"
	pb "github.com/limes-cloud/resource/api/resource/export/v1"
	"github.com/limes-cloud/resource/internal/conf"
	"github.com/limes-cloud/resource/internal/domain/service"
	"github.com/limes-cloud/resource/internal/infra/dbs"
	"github.com/limes-cloud/resource/internal/infra/store"
	"github.com/limes-cloud/resource/internal/types"
)

type Export struct {
	pb.UnimplementedExportServer
	srv *service.Export
}

func NewExport(conf *conf.Config) *Export {
	return &Export{
		srv: service.NewExport(conf, dbs.NewExport(conf), dbs.NewFile(), store.NewExportStore(conf)),
	}
}

func init() {
	register(func(c *conf.Config, hs *http.Server, gs *grpc.Server) {
		app := NewExport(c)
		pb.RegisterExportHTTPServer(hs, app)
		pb.RegisterExportServer(gs, app)

		cr := hs.Route("/")
		cr.GET("/resource/api/v1/download/{expire}/{sign}/{src}", app.srv.Download())

		cr.GET("/resource/api/v1/target/{src}", app.srv.DownloadTarget())
	})
}

// ListExport 获取导出信息列表
func (s *Export) ListExport(c context.Context, req *pb.ListExportRequest) (*pb.ListExportReply, error) {
	list, total, err := s.srv.ListExport(kratosx.MustContext(c), &types.ListExportRequest{
		Page:          req.Page,
		PageSize:      req.PageSize,
		Order:         req.Order,
		OrderBy:       req.OrderBy,
		All:           req.All,
		UserIds:       req.UserIds,
		DepartmentIds: req.DepartmentIds,
	})
	if err != nil {
		return nil, err
	}

	reply := pb.ListExportReply{Total: total}
	for _, item := range list {
		reply.List = append(reply.List, &pb.ListExportReply_Export{
			Id:           item.Id,
			UserId:       item.UserId,
			DepartmentId: item.DepartmentId,
			Scene:        item.Scene,
			Name:         item.Name,
			Size:         item.Size,
			Sha:          item.Sha,
			Src:          item.Src,
			Status:       item.Status,
			Reason:       item.Reason,
			ExpiredAt:    uint32(item.ExpiredAt),
			CreatedAt:    uint32(item.CreatedAt),
			UpdatedAt:    uint32(item.UpdatedAt),
			Url:          item.Url,
		})
	}
	return &reply, nil
}

// ExportFile 创建导出文件
func (s *Export) ExportFile(c context.Context, req *pb.ExportFileRequest) (*pb.ExportFileReply, error) {
	var (
		in  = types.ExportFileRequest{}
		ctx = kratosx.MustContext(c)
	)

	if err := valx.Transform(req, &in); err != nil {
		ctx.Logger().Warnw("msg", "req transform err", "err", err.Error())
		return nil, errors.TransformError()
	}

	res, err := s.srv.ExportFile(ctx, &in)
	if err != nil {
		return nil, err
	}

	return &pb.ExportFileReply{Id: res.Id}, nil
}

// ExportExcel 创建导出excel文件
func (s *Export) ExportExcel(c context.Context, req *pb.ExportExcelRequest) (*pb.ExportExcelReply, error) {
	var in = types.ExportExcelRequest{
		Name:         req.Name,
		UserId:       req.UserId,
		DepartmentId: req.DepartmentId,
		Scene:        req.Scene,
		Headers:      req.Headers,
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

	res, err := s.srv.ExportExcel(kratosx.MustContext(c), &in)
	if err != nil {
		return nil, err
	}

	return &pb.ExportExcelReply{Id: res.Id}, nil
}

// DeleteExport 删除导出信息
func (s *Export) DeleteExport(c context.Context, req *pb.DeleteExportRequest) (*pb.DeleteExportReply, error) {
	total, err := s.srv.DeleteExport(kratosx.MustContext(c), req.Ids)
	if err != nil {
		return nil, err
	}
	return &pb.DeleteExportReply{Total: total}, nil
}

// GetExport 获取指定的导出信息
func (s *Export) GetExport(c context.Context, req *pb.GetExportRequest) (*pb.GetExportReply, error) {
	result, err := s.srv.GetExport(kratosx.MustContext(c), &types.GetExportRequest{
		Id:  req.Id,
		Sha: req.Sha,
	})
	if err != nil {
		return nil, err
	}

	return &pb.GetExportReply{
		Id:           result.Id,
		UserId:       result.UserId,
		DepartmentId: result.DepartmentId,
		Scene:        result.Scene,
		Name:         result.Name,
		Size:         result.Size,
		Sha:          result.Sha,
		Src:          result.Src,
		Status:       result.Status,
		Reason:       result.Reason,
		ExpiredAt:    uint32(result.ExpiredAt),
		CreatedAt:    uint32(result.CreatedAt),
		UpdatedAt:    uint32(result.UpdatedAt),
		Url:          result.Url,
	}, nil
}
