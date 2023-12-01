package handler

import (
	"context"
	"resource/internal/logic"
	"resource/internal/types"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/limes-cloud/kratos"

	v1 "resource/api/v1"
	"resource/config"
)

// Handler is a file service handler.
type Handler struct {
	v1.UnimplementedServiceServer
	file      *logic.File
	directory *logic.Directory
}

func New(conf *config.Config) *Handler {
	return &Handler{
		file:      logic.NewFile(conf),
		directory: logic.NewDirectory(conf),
	}
}

// GetDirectory 获取目录
func (h *Handler) GetDirectory(ctx context.Context, in *v1.GetDirectoryRequest) (*v1.GetDirectoryReply, error) {
	return h.directory.Get(kratos.MustContext(ctx), in)
}

// AddDirectory 添加目录
func (h *Handler) AddDirectory(ctx context.Context, in *v1.AddDirectoryRequest) (*v1.Directory, error) {
	return h.directory.Add(kratos.MustContext(ctx), in)
}

// UpdateDirectory 更新目录
func (h *Handler) UpdateDirectory(ctx context.Context, in *v1.UpdateDirectoryRequest) (*empty.Empty, error) {
	return h.directory.Update(kratos.MustContext(ctx), in)
}

// DeleteDirectory 删除目录
func (h *Handler) DeleteDirectory(ctx context.Context, in *v1.DeleteDirectoryRequest) (*empty.Empty, error) {
	return h.directory.Delete(kratos.MustContext(ctx), in)
}

// PrepareUploadFile 文件预上传
func (h *Handler) PrepareUploadFile(ctx context.Context, in *v1.PrepareUploadFileRequest) (*v1.PrepareUploadFileReply, error) {
	return h.file.PrepareUploadFile(kratos.MustContext(ctx), in)
}

// UploadFile 文件上传
func (h *Handler) UploadFile(ctx context.Context, in *v1.UploadFileRequest) (*v1.UploadFileReply, error) {
	return h.file.UploadFile(kratos.MustContext(ctx), in)
}

// PageFile 文件分野查询
func (h *Handler) PageFile(ctx context.Context, in *v1.PageFileRequest) (*v1.PageFileReply, error) {
	return h.file.PageFile(kratos.MustContext(ctx), in)
}

// UpdateFile 修改文件
func (h *Handler) UpdateFile(ctx context.Context, in *v1.UpdateFileRequest) (*empty.Empty, error) {
	return h.file.UpdateFile(kratos.MustContext(ctx), in)
}

// DeleteFile 删除文件
func (h *Handler) DeleteFile(ctx context.Context, in *v1.DeleteFileRequest) (*empty.Empty, error) {
	return h.file.DeleteFile(kratos.MustContext(ctx), in)
}

// GetFile 获取文件
func (h *Handler) GetFile(ctx context.Context, in *types.GetFileRequest) (*types.GetFileResponse, error) {
	return h.file.GetFile(kratos.MustContext(ctx), in)
}
