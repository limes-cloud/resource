package repository

import (
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/types"
)

type File interface {
	// GetFile 获取指定的文件信息
	GetFile(ctx core.Context, id uint32) (*entity.File, error)

	// GetFileBySha 获取指定的文件信息
	GetFileBySha(ctx core.Context, sha string) (*entity.File, error)

	// GetFileByUploadId 获取指定的文件信息
	GetFileByUploadId(ctx core.Context, uid string) (*entity.File, error)

	// GetFileByKey 获取指定的文件信息
	GetFileByKey(ctx core.Context, key string) (*entity.File, error)

	// CreateFile 创建文件信息
	CreateFile(ctx core.Context, req *entity.File) (uint32, error)

	// UpdateFile 更新文件信息
	UpdateFile(ctx core.Context, req *entity.File) error

	// DeleteFile 删除文件信息
	DeleteFile(ctx core.Context, ids []uint32, call func(file *entity.File)) (uint32, error)

	// CreateUserFile 创建文件信息
	CreateUserFile(ctx core.Context, req *entity.UserFile) (uint32, error)

	// UpdateUserFile 更新文件信息
	UpdateUserFile(ctx core.Context, req *entity.UserFile) error

	// DeleteUserFile 删除文件信息
	DeleteUserFile(ctx core.Context, ids []uint32, call func(UserFile *entity.File)) (uint32, error)

	// ListUserFile 获取文件信息列表
	ListUserFile(ctx core.Context, req *types.ListFileRequest) ([]*entity.UserFile, uint32, error)
}
