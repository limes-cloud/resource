package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/pkg/valx"

	"github.com/limes-cloud/resource/api/resource/errors"
	pb "github.com/limes-cloud/resource/api/resource/file/v1"
	"github.com/limes-cloud/resource/internal/biz/file"
	"github.com/limes-cloud/resource/internal/conf"
	"github.com/limes-cloud/resource/internal/data"
)

type FileService struct {
	pb.UnimplementedFileServer
	uc   *file.UseCase
	conf *conf.Config
}

func NewFileService(conf *conf.Config) *FileService {
	return &FileService{
		uc:   file.NewUseCase(conf, data.NewFileRepo(conf, globalStore)),
		conf: conf,
	}
}

func init() {
	register(func(c *conf.Config, hs *http.Server, gs *grpc.Server) {
		srv := NewFileService(c)
		pb.RegisterFileHTTPServer(hs, srv)
		pb.RegisterFileServer(gs, srv)

		cr := hs.Route("/")
		cr.GET("/resource/api/v1/static/{expire}/{sign}/{src}", srv.SrcBlob())
		cr.POST("/resource/api/v1/upload", srv.Upload())
		cr.POST("/resource/client/v1/upload", srv.Upload())
	})
}

// GetFile 获取指定的文件信息
func (s *FileService) GetFile(c context.Context, req *pb.GetFileRequest) (*pb.GetFileReply, error) {
	var (
		in  = file.GetFileRequest{}
		ctx = kratosx.MustContext(c)
	)

	if err := valx.Transform(req, &in); err != nil {
		ctx.Logger().Warnw("msg", "req transform err", "err", err.Error())
		return nil, errors.TransformError()
	}

	result, err := s.uc.GetFile(ctx, &in)
	if err != nil {
		return nil, err
	}

	reply := pb.GetFileReply{}
	if err := valx.Transform(result, &reply); err != nil {
		ctx.Logger().Warnw("msg", "reply transform err", "err", err.Error())
		return nil, errors.TransformError()
	}
	return &reply, nil
}

// ListFile 获取文件信息列表
func (s *FileService) ListFile(c context.Context, req *pb.ListFileRequest) (*pb.ListFileReply, error) {
	var (
		in  = file.ListFileRequest{}
		ctx = kratosx.MustContext(c)
	)

	if err := valx.Transform(req, &in); err != nil {
		ctx.Logger().Warnw("msg", "req transform err", "err", err.Error())
		return nil, errors.TransformError()
	}

	result, total, err := s.uc.ListFile(ctx, &in)
	if err != nil {
		return nil, err
	}

	reply := pb.ListFileReply{Total: total}
	if err := valx.Transform(result, &reply.List); err != nil {
		ctx.Logger().Warnw("msg", "reply transform err", "err", err.Error())
		return nil, errors.TransformError()
	}

	return &reply, nil
}

// PrepareUploadFile 预上传文件信息
func (s *FileService) PrepareUploadFile(c context.Context, req *pb.PrepareUploadFileRequest) (*pb.PrepareUploadFileReply, error) {
	var (
		in  = file.PrepareUploadFileRequest{}
		ctx = kratosx.MustContext(c)
	)

	if err := valx.Transform(req, &in); err != nil {
		ctx.Logger().Warnw("msg", "req transform err", "err", err.Error())
		return nil, errors.TransformError()
	}

	if req.DirectoryPath == nil && req.DirectoryId == nil {
		return nil, errors.ParamsError()
	}

	res, err := s.uc.PrepareUploadFile(ctx, &in)
	if err != nil {
		return nil, err
	}

	return &pb.PrepareUploadFileReply{
		Uploaded:     res.Uploaded,
		Src:          res.Src,
		ChunkSize:    res.ChunkSize,
		ChunkCount:   res.ChunkCount,
		UploadId:     res.UploadId,
		UploadChunks: res.UploadChunks,
		Sha:          res.Sha,
		Url:          res.URL,
	}, nil
}

// UploadFile 上传文件信息
func (s *FileService) UploadFile(c context.Context, req *pb.UploadFileRequest) (*pb.UploadFileReply, error) {
	reply, err := s.uc.UploadFile(kratosx.MustContext(c), &file.UploadFileRequest{
		UploadId: req.UploadId,
		Index:    req.Index,
		Data:     req.Data,
	})
	if err != nil {
		return nil, err
	}
	return &pb.UploadFileReply{
		Src: reply.Src,
		Sha: reply.Sha,
		Url: reply.URL,
	}, nil
}

// UpdateFile 更新文件信息
func (s *FileService) UpdateFile(c context.Context, req *pb.UpdateFileRequest) (*pb.UpdateFileReply, error) {
	var (
		in  = file.File{}
		ctx = kratosx.MustContext(c)
	)

	if err := valx.Transform(req, &in); err != nil {
		ctx.Logger().Warnw("msg", "req transform err", "err", err.Error())
		return nil, errors.TransformError()
	}

	if err := s.uc.UpdateFile(ctx, &in); err != nil {
		return nil, err
	}

	return &pb.UpdateFileReply{}, nil
}

// DeleteFile 删除文件信息
func (s *FileService) DeleteFile(c context.Context, req *pb.DeleteFileRequest) (*pb.DeleteFileReply, error) {
	total, err := s.uc.DeleteFile(kratosx.MustContext(c), req.Ids)
	if err != nil {
		return nil, err
	}
	return &pb.DeleteFileReply{Total: total}, nil
}
