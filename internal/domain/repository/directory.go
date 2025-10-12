package repository

import (
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
)

type Directory interface {
	// GetDirectory 获取指定的文件目录信息
	GetDirectory(ctx core.Context, id uint32) (*entity.Directory, error)

	// ListDirectory 获取文件目录信息列表
	ListDirectory(ctx core.Context) ([]*entity.Directory, error)

	// CreateDirectory 创建文件目录信息
	CreateDirectory(ctx core.Context, req *entity.Directory) (uint32, error)

	// UpdateDirectory 更新文件目录信息
	UpdateDirectory(ctx core.Context, req *entity.Directory) error

	// DeleteDirectory 删除文件目录信息
	DeleteDirectory(ctx core.Context, ids []uint32) (uint32, error)

	// GetDirectoryParentIds 获取父文件目录信息ID列表
	GetDirectoryParentIds(ctx core.Context, id uint32) ([]uint32, error)

	// GetDirectoryChildrenIds 获取子文件目录信息ID列表
	GetDirectoryChildrenIds(ctx core.Context, id uint32) ([]uint32, error)

	// GetDirectoryLimitByPath 获取指定的path上传限制信息
	GetDirectoryLimitByPath(ctx core.Context, paths []string) (*entity.DirectoryLimit, error)

	// GetDirectoryLimitById 获取指定的id上传限制信息
	GetDirectoryLimitById(ctx core.Context, id uint32) (*entity.DirectoryLimit, error)
}
