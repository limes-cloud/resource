package service

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/pkg/util"

	"github.com/limes-cloud/resource/api/errors"
	pb "github.com/limes-cloud/resource/api/file/v1"
	biz "github.com/limes-cloud/resource/internal/biz/file"
	"github.com/limes-cloud/resource/internal/config"
	"github.com/limes-cloud/resource/internal/data/export"
	data "github.com/limes-cloud/resource/internal/data/file"
	"github.com/limes-cloud/resource/internal/factory"
)

type FileService struct {
	pb.UnimplementedServiceServer
	uc   *biz.UseCase
	conf *config.Config
}

func NewFile(conf *config.Config) *FileService {
	return &FileService{
		conf: conf,
		uc:   biz.NewUseCase(conf, data.NewRepo(), factory.New(conf, data.NewRepo(), export.NewRepo())),
	}
}

func (fs *FileService) Config() *config.Config {
	return fs.conf
}

// AllDirectory 获取目录
func (fs *FileService) AllDirectory(ctx context.Context, in *pb.AllDirectoryRequest) (*pb.AllDirectoryReply, error) {
	list, err := fs.uc.AllDirectoryByParentID(kratosx.MustContext(ctx), in.ParentId, in.App)
	if err != nil {
		return nil, err
	}

	reply := pb.AllDirectoryReply{}
	if err := util.Transform(list, &reply.List); err != nil {
		return nil, errors.TransformFormat(err.Error())
	}
	return &reply, nil
}

// AddDirectory 添加目录
func (fs *FileService) AddDirectory(ctx context.Context, in *pb.AddDirectoryRequest) (*pb.Directory, error) {
	req := biz.Directory{}
	if err := util.Transform(in, &req); err != nil {
		return nil, errors.TransformFormat(err.Error())
	}
	id, err := fs.uc.AddDirectory(kratosx.MustContext(ctx), &req)
	if err != nil {
		return nil, err
	}
	req.ID = id
	reply := pb.Directory{}
	if err := util.Transform(req, &reply); err != nil {
		return nil, errors.TransformFormat(err.Error())
	}

	return &reply, nil
}

// UpdateDirectory 更新目录
func (fs *FileService) UpdateDirectory(ctx context.Context, in *pb.UpdateDirectoryRequest) (*empty.Empty, error) {
	req := biz.Directory{}
	if err := util.Transform(in, &req); err != nil {
		return nil, errors.TransformFormat(err.Error())
	}
	return nil, fs.uc.UpdateDirectory(kratosx.MustContext(ctx), &req)
}

// DeleteDirectory 删除目录
func (fs *FileService) DeleteDirectory(ctx context.Context, in *pb.DeleteDirectoryRequest) (*empty.Empty, error) {
	return nil, fs.uc.DeleteDirectory(kratosx.MustContext(ctx), in.Id, in.App)
}

// PrepareUploadFile 文件预上传
func (fs *FileService) PrepareUploadFile(ctx context.Context, in *pb.PrepareUploadFileRequest) (*pb.PrepareUploadFileReply, error) {
	req := biz.PrepareUploadFileRequest{}
	if err := util.Transform(in, &req); err != nil {
		return nil, errors.TransformFormat(err.Error())
	}

	res, err := fs.uc.PrepareUploadFile(kratosx.MustContext(ctx), &req)
	if err != nil {
		return nil, err
	}

	reply := pb.PrepareUploadFileReply{}
	if err := util.Transform(res, &reply); err != nil {
		return nil, errors.TransformFormat(err.Error())
	}
	return &reply, nil
}

// UploadFile 文件上传
func (fs *FileService) UploadFile(ctx context.Context, in *pb.UploadFileRequest) (*pb.UploadFileReply, error) {
	req := biz.UploadFileRequest{}
	if err := util.Transform(in, &req); err != nil {
		return nil, errors.TransformFormat(err.Error())
	}

	res, err := fs.uc.UploadFile(kratosx.MustContext(ctx), &req)
	if err != nil {
		return nil, err
	}

	reply := pb.UploadFileReply{}
	if err := util.Transform(res, &reply); err != nil {
		return nil, errors.TransformFormat(err.Error())
	}
	return &reply, nil
}

// GetFileBySha 文件查询
func (fs *FileService) GetFileBySha(ctx context.Context, in *pb.GetFileByShaRequest) (*pb.File, error) {
	res, err := fs.uc.GetFileBySha(kratosx.MustContext(ctx), in.Sha)
	if err != nil {
		return nil, err
	}

	reply := pb.File{}
	if err := util.Transform(res, &reply); err != nil {
		return nil, errors.TransformFormat(err.Error())
	}
	return &reply, nil
}

// PageFile 文件分野查询
func (fs *FileService) PageFile(ctx context.Context, in *pb.PageFileRequest) (*pb.PageFileReply, error) {
	req := biz.PageFileRequest{}
	if err := util.Transform(in, &req); err != nil {
		return nil, errors.TransformFormat(err.Error())
	}

	list, total, err := fs.uc.PageFile(kratosx.MustContext(ctx), &req)
	if err != nil {
		return nil, err
	}

	reply := pb.PageFileReply{Total: &total}
	if err := util.Transform(list, &reply.List); err != nil {
		return nil, errors.TransformFormat(err.Error())
	}
	return &reply, nil
}

// UpdateFile 修改文件
func (fs *FileService) UpdateFile(ctx context.Context, in *pb.UpdateFileRequest) (*empty.Empty, error) {
	req := biz.File{}
	if err := util.Transform(in, &req); err != nil {
		return nil, errors.TransformFormat(err.Error())
	}
	return nil, fs.uc.UpdateFile(kratosx.MustContext(ctx), &req)
}

// DeleteFile 删除文件
func (fs *FileService) DeleteFile(ctx context.Context, in *pb.DeleteFileRequest) (*empty.Empty, error) {
	return nil, fs.uc.DeleteFiles(kratosx.MustContext(ctx), in.DirectoryId, in.Ids)
}

// GetFile 获取文件
func (fs *FileService) GetFile(ctx context.Context, in *pb.GetFileRequest) (*pb.GetFileReply, error) {
	req := biz.GetFileRequest{}
	if err := util.Transform(in, &req); err != nil {
		return nil, errors.TransformFormat(err.Error())
	}

	res, err := fs.uc.GetFile(kratosx.MustContext(ctx), &req)
	if err != nil {
		return nil, err
	}
	return &pb.GetFileReply{
		Data: res.Data,
		Mime: res.Mime,
	}, nil
}
