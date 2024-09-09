package repository

import (
	"github.com/limes-cloud/kratosx"

	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/types"
)

type File interface {
	// GetFile 获取指定的文件信息
	GetFile(ctx kratosx.Context, id uint32) (*entity.File, error)

	// GetFileBySha 获取指定的文件信息
	GetFileBySha(ctx kratosx.Context, sha string) (*entity.File, error)

	// GetFileByUploadId 获取指定的文件信息
	GetFileByUploadId(ctx kratosx.Context, uid string) (*entity.File, error)

	// GetFileBySrc 获取指定的文件信息
	GetFileBySrc(ctx kratosx.Context, src string) (*entity.File, error)

	// ListFile 获取文件信息列表
	ListFile(ctx kratosx.Context, req *types.ListFileRequest) ([]*entity.File, uint32, error)

	// CreateFile 创建文件信息
	CreateFile(ctx kratosx.Context, req *entity.File) (uint32, error)

	// CopyFile 复制文件信息
	CopyFile(ctx kratosx.Context, src *entity.File, directoryId uint32, fileName string) error

	// UpdateFile 更新文件信息
	UpdateFile(ctx kratosx.Context, req *entity.File) error

	// DeleteFile 删除文件信息
	DeleteFile(ctx kratosx.Context, ids []uint32, call func(file *entity.File)) (uint32, error)
}
