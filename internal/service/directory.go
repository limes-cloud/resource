package service

import (
	"context"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/pkg/valx"
	pb "github.com/limes-cloud/resource/api/resource/directory/v1"
	"github.com/limes-cloud/resource/api/resource/errors"
	"github.com/limes-cloud/resource/internal/biz/directory"
	"github.com/limes-cloud/resource/internal/conf"
	"github.com/limes-cloud/resource/internal/data"
)

type DirectoryService struct {
	pb.UnimplementedDirectoryServer
	uc *directory.UseCase
}

func NewDirectoryService(conf *conf.Config) *DirectoryService {
	return &DirectoryService{
		uc: directory.NewUseCase(conf, data.NewDirectoryRepo()),
	}
}

func init() {
	register(func(c *conf.Config, hs *http.Server, gs *grpc.Server) {
		srv := NewDirectoryService(c)
		pb.RegisterDirectoryHTTPServer(hs, srv)
		pb.RegisterDirectoryServer(gs, srv)
	})
}

// GetDirectory 获取指定的文件目录信息
func (s *DirectoryService) GetDirectory(c context.Context, req *pb.GetDirectoryRequest) (*pb.GetDirectoryReply, error) {
	var (
		in  = directory.GetDirectoryRequest{}
		ctx = kratosx.MustContext(c)
	)

	if err := valx.Transform(req, &in); err != nil {
		ctx.Logger().Warnw("msg", "req transform err", "err", err.Error())
		return nil, errors.TransformError()
	}

	result, err := s.uc.GetDirectory(ctx, &in)
	if err != nil {
		return nil, err
	}

	reply := pb.GetDirectoryReply{}
	if err := valx.Transform(result, &reply); err != nil {
		ctx.Logger().Warnw("msg", "reply transform err", "err", err.Error())
		return nil, errors.TransformError()
	}
	return &reply, nil
}

// ListDirectory 获取文件目录信息列表
func (s *DirectoryService) ListDirectory(c context.Context, req *pb.ListDirectoryRequest) (*pb.ListDirectoryReply, error) {
	var (
		in  = directory.ListDirectoryRequest{}
		ctx = kratosx.MustContext(c)
	)

	if err := valx.Transform(req, &in); err != nil {
		ctx.Logger().Warnw("msg", "req transform err", "err", err.Error())
		return nil, errors.TransformError()
	}

	result, total, err := s.uc.ListDirectory(ctx, &in)
	if err != nil {
		return nil, err
	}

	reply := pb.ListDirectoryReply{Total: total}
	if err := valx.Transform(result, &reply.List); err != nil {
		ctx.Logger().Warnw("msg", "reply transform err", "err", err.Error())
		return nil, errors.TransformError()
	}

	return &reply, nil
}

// CreateDirectory 创建文件目录信息
func (s *DirectoryService) CreateDirectory(c context.Context, req *pb.CreateDirectoryRequest) (*pb.CreateDirectoryReply, error) {
	var (
		in  = directory.Directory{}
		ctx = kratosx.MustContext(c)
	)

	if err := valx.Transform(req, &in); err != nil {
		ctx.Logger().Warnw("msg", "req transform err", "err", err.Error())
		return nil, errors.TransformError()
	}

	id, err := s.uc.CreateDirectory(ctx, &in)
	if err != nil {
		return nil, err
	}

	return &pb.CreateDirectoryReply{Id: id}, nil
}

// UpdateDirectory 更新文件目录信息
func (s *DirectoryService) UpdateDirectory(c context.Context, req *pb.UpdateDirectoryRequest) (*pb.UpdateDirectoryReply, error) {
	var (
		in  = directory.Directory{}
		ctx = kratosx.MustContext(c)
	)

	if err := valx.Transform(req, &in); err != nil {
		ctx.Logger().Warnw("msg", "req transform err", "err", err.Error())
		return nil, errors.TransformError()
	}

	if err := s.uc.UpdateDirectory(ctx, &in); err != nil {
		return nil, err
	}

	return &pb.UpdateDirectoryReply{}, nil
}

// DeleteDirectory 删除文件目录信息
func (s *DirectoryService) DeleteDirectory(c context.Context, req *pb.DeleteDirectoryRequest) (*pb.DeleteDirectoryReply, error) {
	total, err := s.uc.DeleteDirectory(kratosx.MustContext(c), req.Ids)
	if err != nil {
		return nil, err
	}
	return &pb.DeleteDirectoryReply{Total: total}, nil
}
