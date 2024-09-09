package app

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/pkg/valx"
	ktypes "github.com/limes-cloud/kratosx/types"

	pb "github.com/limes-cloud/resource/api/resource/directory/v1"
	"github.com/limes-cloud/resource/api/resource/errors"
	"github.com/limes-cloud/resource/internal/conf"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/service"
	"github.com/limes-cloud/resource/internal/infra/dbs"
	"github.com/limes-cloud/resource/internal/types"
)

type Directory struct {
	pb.UnimplementedDirectoryServer
	srv *service.Directory
}

func NewDirectory(conf *conf.Config) *Directory {
	return &Directory{
		srv: service.NewDirectory(conf, dbs.NewDirectory(conf)),
	}
}

func init() {
	register(func(c *conf.Config, hs *http.Server, gs *grpc.Server) {
		srv := NewDirectory(c)
		pb.RegisterDirectoryHTTPServer(hs, srv)
		pb.RegisterDirectoryServer(gs, srv)
	})
}

// GetDirectory 获取指定的文件目录信息
func (s *Directory) GetDirectory(c context.Context, req *pb.GetDirectoryRequest) (*pb.GetDirectoryReply, error) {
	result, err := s.srv.GetDirectory(kratosx.MustContext(c), req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.GetDirectoryReply{
		Id:        result.Id,
		ParentId:  result.ParentId,
		Name:      result.Name,
		Accept:    result.Accept,
		MaxSize:   result.MaxSize,
		CreatedAt: uint32(result.CreatedAt),
		UpdatedAt: uint32(result.UpdatedAt),
	}, nil
}

// ListDirectory 获取文件目录信息列表
func (s *Directory) ListDirectory(c context.Context, req *pb.ListDirectoryRequest) (*pb.ListDirectoryReply, error) {
	result, total, err := s.srv.ListDirectory(kratosx.MustContext(c), &types.ListDirectoryRequest{
		Order:   req.Order,
		OrderBy: req.OrderBy,
	})
	if err != nil {
		return nil, err
	}
	reply := pb.ListDirectoryReply{Total: total}
	if err := valx.Transform(result, &reply.List); err != nil {
		return nil, errors.TransformError()
	}
	return &reply, nil
}

// CreateDirectory 创建文件目录信息
func (s *Directory) CreateDirectory(c context.Context, req *pb.CreateDirectoryRequest) (*pb.CreateDirectoryReply, error) {
	id, err := s.srv.CreateDirectory(kratosx.MustContext(c), &entity.Directory{
		ParentId: req.ParentId,
		Name:     req.Name,
		Accept:   req.Accept,
		MaxSize:  req.MaxSize,
	})
	if err != nil {
		return nil, err
	}

	return &pb.CreateDirectoryReply{Id: id}, nil
}

// UpdateDirectory 更新文件目录信息
func (s *Directory) UpdateDirectory(c context.Context, req *pb.UpdateDirectoryRequest) (*pb.UpdateDirectoryReply, error) {
	if err := s.srv.UpdateDirectory(kratosx.MustContext(c), &entity.Directory{
		BaseModel: ktypes.BaseModel{Id: req.Id},
		ParentId:  req.ParentId,
		Name:      req.Name,
		Accept:    req.Accept,
		MaxSize:   req.MaxSize,
	}); err != nil {
		return nil, err
	}
	return &pb.UpdateDirectoryReply{}, nil
}

// DeleteDirectory 删除文件目录信息
func (s *Directory) DeleteDirectory(c context.Context, req *pb.DeleteDirectoryRequest) (*pb.DeleteDirectoryReply, error) {
	total, err := s.srv.DeleteDirectory(kratosx.MustContext(c), req.Ids)
	if err != nil {
		return nil, err
	}
	return &pb.DeleteDirectoryReply{Total: total}, nil
}
