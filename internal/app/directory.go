package app

import (
	"context"
	"github.com/limes-cloud/kratosx/model"
	"github.com/limes-cloud/kratosx/pkg/value"
	"github.com/limes-cloud/resource/api/directory"
	"github.com/limes-cloud/resource/api/errors"
	"github.com/limes-cloud/resource/internal/core"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/service"
	"github.com/limes-cloud/resource/internal/infra/dbs"
)

type Directory struct {
	directory.UnimplementedDirectoryServer
	srv *service.Directory
}

func NewDirectory() *Directory {
	return &Directory{
		srv: service.NewDirectory(dbs.NewDirectory()),
	}
}

func init() {
	register(func(hs *http.Server, gs *grpc.Server) {
		srv := NewDirectory()
		directory.RegisterDirectoryHTTPServer(hs, srv)
		directory.RegisterDirectoryServer(gs, srv)
	})
}

// GetDirectory 获取指定的文件目录信息
func (s *Directory) GetDirectory(c context.Context, req *directory.GetDirectoryRequest) (*directory.GetDirectoryReply, error) {
	result, err := s.srv.GetDirectory(core.MustContext(c), req.Id)
	if err != nil {
		return nil, err
	}
	return &directory.GetDirectoryReply{
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
func (s *Directory) ListDirectory(c context.Context, _ *emptypb.Empty) (*directory.ListDirectoryReply, error) {
	result, err := s.srv.ListDirectory(core.MustContext(c))
	if err != nil {
		return nil, err
	}
	reply := directory.ListDirectoryReply{}
	if err := value.Transform(result, &reply.List); err != nil {
		return nil, errors.TransformError()
	}
	return &reply, nil
}

// CreateDirectory 创建文件目录信息
func (s *Directory) CreateDirectory(c context.Context, req *directory.CreateDirectoryRequest) (*directory.CreateDirectoryReply, error) {
	id, err := s.srv.CreateDirectory(core.MustContext(c), &entity.Directory{
		ParentId: req.ParentId,
		Name:     req.Name,
		Accept:   req.Accept,
		MaxSize:  req.MaxSize,
	})
	if err != nil {
		return nil, err
	}

	return &directory.CreateDirectoryReply{Id: id}, nil
}

// UpdateDirectory 更新文件目录信息
func (s *Directory) UpdateDirectory(c context.Context, req *directory.UpdateDirectoryRequest) (*directory.UpdateDirectoryReply, error) {
	if err := s.srv.UpdateDirectory(core.MustContext(c), &entity.Directory{
		BaseTenantModel: model.BaseTenantModel{Id: req.Id},
		ParentId:        req.ParentId,
		Name:            req.Name,
		Accept:          req.Accept,
		MaxSize:         req.MaxSize,
	}); err != nil {
		return nil, err
	}
	return &directory.UpdateDirectoryReply{}, nil
}

// DeleteDirectory 删除文件目录信息
func (s *Directory) DeleteDirectory(c context.Context, req *directory.DeleteDirectoryRequest) (*directory.DeleteDirectoryReply, error) {
	total, err := s.srv.DeleteDirectory(core.MustContext(c), req.Ids)
	if err != nil {
		return nil, err
	}
	return &directory.DeleteDirectoryReply{Total: total}, nil
}
