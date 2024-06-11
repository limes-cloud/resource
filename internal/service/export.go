package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/pkg/valx"

	"github.com/limes-cloud/resource/api/resource/errors"
	pb "github.com/limes-cloud/resource/api/resource/export/v1"
	"github.com/limes-cloud/resource/internal/biz/export"
	"github.com/limes-cloud/resource/internal/conf"
	"github.com/limes-cloud/resource/internal/data"
)

type ExportService struct {
	pb.UnimplementedExportServer
	uc   *export.UseCase
	conf *conf.Config
}

func NewExportService(conf *conf.Config) *ExportService {
	return &ExportService{
		conf: conf,
		uc:   export.NewUseCase(conf, data.NewExportRepo(conf, globalStore, globalExportStore)),
	}
}

func init() {
	register(func(c *conf.Config, hs *http.Server, gs *grpc.Server) {
		srv := NewExportService(c)
		pb.RegisterExportHTTPServer(hs, srv)
		pb.RegisterExportServer(gs, srv)

		cr := hs.Route("/")
		cr.GET("/resource/api/v1/download/{expire}/{sign}/{src}", srv.Download())
	})
}

// ListExport 获取导出信息列表
func (s *ExportService) ListExport(c context.Context, req *pb.ListExportRequest) (*pb.ListExportReply, error) {
	var (
		in  = export.ListExportRequest{}
		ctx = kratosx.MustContext(c)
	)

	if err := valx.Transform(req, &in); err != nil {
		ctx.Logger().Warnw("msg", "req transform err", "err", err.Error())
		return nil, errors.TransformError()
	}

	result, total, err := s.uc.ListExport(ctx, &in)
	if err != nil {
		return nil, err
	}

	reply := pb.ListExportReply{Total: total}
	if err := valx.Transform(result, &reply.List); err != nil {
		ctx.Logger().Warnw("msg", "reply transform err", "err", err.Error())
		return nil, errors.TransformError()
	}

	return &reply, nil
}

// ExportFile 创建导出文件
func (s *ExportService) ExportFile(c context.Context, req *pb.ExportFileRequest) (*pb.ExportFileReply, error) {
	var (
		in  = export.ExportFileRequest{}
		ctx = kratosx.MustContext(c)
	)

	if err := valx.Transform(req, &in); err != nil {
		ctx.Logger().Warnw("msg", "req transform err", "err", err.Error())
		return nil, errors.TransformError()
	}

	res, err := s.uc.ExportFile(ctx, &in)
	if err != nil {
		return nil, err
	}

	return &pb.ExportFileReply{Id: res.Id}, nil
}

// ExportExcel 创建导出excel文件
func (s *ExportService) ExportExcel(c context.Context, req *pb.ExportExcelRequest) (*pb.ExportExcelReply, error) {
	var (
		in = export.ExportExcelRequest{
			Name:         req.Name,
			UserId:       req.UserId,
			DepartmentId: req.DepartmentId,
			Scene:        req.Scene,
		}
		ctx = kratosx.MustContext(c)
	)

	for _, row := range req.Rows {
		var temp []*export.ExportExcelCol
		for _, col := range row.Cols {
			temp = append(temp, &export.ExportExcelCol{
				Type:  col.Type,
				Value: col.Value,
			})
		}
		in.Rows = append(in.Rows, temp)
	}

	res, err := s.uc.ExportExcel(ctx, &in)
	if err != nil {
		return nil, err
	}

	return &pb.ExportExcelReply{Id: res.Id}, nil
}

// DeleteExport 删除导出信息
func (s *ExportService) DeleteExport(c context.Context, req *pb.DeleteExportRequest) (*pb.DeleteExportReply, error) {
	total, err := s.uc.DeleteExport(kratosx.MustContext(c), req.Ids)
	if err != nil {
		return nil, err
	}
	return &pb.DeleteExportReply{Total: total}, nil
}

// GetExport 获取指定的导出信息
func (s *ExportService) GetExport(c context.Context, req *pb.GetExportRequest) (*pb.GetExportReply, error) {
	var (
		in  = export.GetExportRequest{}
		ctx = kratosx.MustContext(c)
	)

	if err := valx.Transform(req, &in); err != nil {
		ctx.Logger().Warnw("msg", "req transform err", "err", err.Error())
		return nil, errors.TransformError()
	}

	result, err := s.uc.GetExport(ctx, &in)
	if err != nil {
		return nil, err
	}

	reply := pb.GetExportReply{}
	if err := valx.Transform(result, &reply); err != nil {
		ctx.Logger().Warnw("msg", "reply transform err", "err", err.Error())
		return nil, errors.TransformError()
	}
	return &reply, nil
}
