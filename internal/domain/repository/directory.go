package repository

import (
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/types"
)

// Directory 目录仓储接口。
type Directory interface {
	// GetDirectory 按主键查询（必须带 app 限定）
	GetDirectory(ctx core.Context, app string, id uint32) (*entity.Directory, error)
	// GetDirectoryById 按主键查询（不限定 app，用于从目录派生 app 等读场景）
	GetDirectoryById(ctx core.Context, id uint32) (*entity.Directory, error)
	// GetDirectoryByPath 按路径查询
	GetDirectoryByPath(ctx core.Context, app, path string) (*entity.Directory, error)
	// ListDirectory 全量取回某 app 的目录（内存建树用，按 path 升序）
	ListDirectory(ctx core.Context, app string) ([]*entity.Directory, error)
	// ListDirectoryInPaths 按路径集合批量查询（自动建目录"先定位最深已存在层"用，一次查询代替逐层 SELECT）
	ListDirectoryInPaths(ctx core.Context, app string, paths []string) ([]*entity.Directory, error)
	// CreateDirectoryIgnore 幂等新建，返回最终行（INSERT IGNORE 后重查）。
	CreateDirectoryIgnore(ctx core.Context, ent *entity.Directory) (*entity.Directory, error)
	// UpdateDirectory 按主键+app 更新目录实体（零值可写列用指针，GORM 直接 Updates(ent)）
	UpdateDirectory(ctx core.Context, ent *entity.Directory) error
	// MoveDirectory 移动子树：一条 UPDATE 重写整棵子树 path 并刷新 updated_at/updated_by（必须用 CHAR_LENGTH 配 SUBSTRING 的字符语义）
	MoveDirectory(ctx core.Context, p *types.MoveDirectoryParams) error
	// DeleteDirectoryTree 删子树：一条语句删掉自己与全部后代（前缀谓词必须转义）
	DeleteDirectoryTree(ctx core.Context, app string, id uint32, path string) error
}
