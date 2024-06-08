package file

import (
	"github.com/limes-cloud/kratosx"

	"github.com/limes-cloud/resource/internal/pkg/store"
)

type Repo interface {
	// GetFile 获取指定的文件信息
	GetFile(ctx kratosx.Context, id uint32) (*File, error)

	// ListFile 获取文件信息列表
	ListFile(ctx kratosx.Context, req *ListFileRequest) ([]*File, uint32, error)

	// CreateFile 创建文件信息
	CreateFile(ctx kratosx.Context, req *File) (uint32, error)

	// CopyFile 复制文件信息
	CopyFile(ctx kratosx.Context, src *File, directoryId uint32, fileName string) error

	// UpdateFile 更新文件信息
	UpdateFile(ctx kratosx.Context, req *File) error

	// UpdateFileStatus 更新文	件状态
	UpdateFileStatus(ctx kratosx.Context, id uint32, status string) error

	// DeleteFile 删除文件信息
	DeleteFile(ctx kratosx.Context, ids []uint32) (uint32, error)

	// GetFileBySha 获取指定的文件信息
	GetFileBySha(ctx kratosx.Context, sha string) (*File, error)

	// GetFileByUploadId 获取指定的文件信息
	GetFileByUploadId(ctx kratosx.Context, uid string) (*File, error)

	// GetDirectoryLimitByPath 获取指定的path上传限制信息
	GetDirectoryLimitByPath(ctx kratosx.Context, paths []string) (*DirectoryLimit, error)

	// GetDirectoryLimitById 获取指定的id上传限制信息
	GetDirectoryLimitById(ctx kratosx.Context, id uint32) (*DirectoryLimit, error)

	// GetStore 获取上传器
	GetStore() store.Store
}
