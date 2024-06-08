package directory

import (
	"github.com/limes-cloud/kratosx"
)

type Repo interface {
	// GetDirectory 获取指定的文件目录信息
	GetDirectory(ctx kratosx.Context, id uint32) (*Directory, error)

	// ListDirectory 获取文件目录信息列表
	ListDirectory(ctx kratosx.Context, req *ListDirectoryRequest) ([]*Directory, uint32, error)

	// CreateDirectory 创建文件目录信息
	CreateDirectory(ctx kratosx.Context, req *Directory) (uint32, error)

	// UpdateDirectory 更新文件目录信息
	UpdateDirectory(ctx kratosx.Context, req *Directory) error

	// DeleteDirectory 删除文件目录信息
	DeleteDirectory(ctx kratosx.Context, ids []uint32) (uint32, error)

	// GetDirectoryParentIds 获取父文件目录信息ID列表
	GetDirectoryParentIds(ctx kratosx.Context, id uint32) ([]uint32, error)

	// GetDirectoryChildrenIds 获取子文件目录信息ID列表
	GetDirectoryChildrenIds(ctx kratosx.Context, id uint32) ([]uint32, error)
}
