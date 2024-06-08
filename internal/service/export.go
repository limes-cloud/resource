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
	uc *export.UseCase
}

func NewExportService(conf *conf.Config) *ExportService {
	return &ExportService{
		uc: export.NewUseCase(conf, data.NewExportRepo()),
	}
}

func init() {
	register(func(c *conf.Config, hs *http.Server, gs *grpc.Server) {
		srv := NewExportService(c)
		pb.RegisterExportHTTPServer(hs, srv)
		pb.RegisterExportServer(gs, srv)
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

// CreateExport 创建导出信息
func (s *ExportService) CreateExport(c context.Context, req *pb.CreateExportRequest) (*pb.CreateExportReply, error) {
	var (
		in  = export.Export{}
		ctx = kratosx.MustContext(c)
	)

	if err := valx.Transform(req, &in); err != nil {
		ctx.Logger().Warnw("msg", "req transform err", "err", err.Error())
		return nil, errors.TransformError()
	}

	id, err := s.uc.CreateExport(ctx, &in)
	if err != nil {
		return nil, err
	}

	return &pb.CreateExportReply{Id: id}, nil
}

// DeleteExport 删除导出信息
func (s *ExportService) DeleteExport(c context.Context, req *pb.DeleteExportRequest) (*pb.DeleteExportReply, error) {
	total, err := s.uc.DeleteExport(kratosx.MustContext(c), req.Ids)
	if err != nil {
		return nil, err
	}
	return &pb.DeleteExportReply{Total: total}, nil
}
