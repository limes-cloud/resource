package dbs

import (
	"sync"
	"time"

	"gorm.io/gorm/clause"

	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/repository"
	"github.com/limes-cloud/resource/internal/pkg/pattern"
	"github.com/limes-cloud/resource/internal/types"
)

// moveTreeSQL 移动目录：一条 UPDATE 重写整棵子树的 path 并刷新 updated_at/updated_by。必须用 CHAR_LENGTH，LENGTH 返回字节数，中文路径上会把整棵子树截成裸前缀。
const moveTreeSQL = `UPDATE directory
   SET path = CONCAT(?, SUBSTRING(path, CHAR_LENGTH(?) + 1)),
       updated_at = ?,
       updated_by = ?
 WHERE app = ? AND (id = ? OR path LIKE ? ESCAPE '\\')`

// deleteTreeSQL 删目录 = 删子树。前缀谓词必须转义：目录 图_片 会命中 /图A片/...，误删整棵树。
const deleteTreeSQL = `DELETE FROM directory WHERE app = ? AND (id = ? OR path LIKE ? ESCAPE '\\')`

// Directory 目录仓储实现（物化路径）
type Directory struct{}

var (
	directoryIns  *Directory
	directoryOnce sync.Once
)

// NewDirectory 取目录仓储单例
func NewDirectory() *Directory {
	directoryOnce.Do(func() { directoryIns = &Directory{} })
	return directoryIns
}

// GetDirectory 按主键查询目录（必须带 app 限定）
func (r Directory) GetDirectory(ctx core.Context, app string, id uint32) (*entity.Directory, error) {
	var ent entity.Directory
	if err := ctx.DB().Where("app = ? AND id = ?", app, id).First(&ent).Error; err != nil {
		return nil, err
	}
	return &ent, nil
}

// GetDirectoryById 按主键查询目录（不限定 app，用于从目录派生 app 等读场景）
func (r Directory) GetDirectoryById(ctx core.Context, id uint32) (*entity.Directory, error) {
	var ent entity.Directory
	if err := ctx.DB().Where("id = ?", id).First(&ent).Error; err != nil {
		return nil, err
	}
	return &ent, nil
}

// GetDirectoryByPath 按路径查询目录
func (r Directory) GetDirectoryByPath(ctx core.Context, app, path string) (*entity.Directory, error) {
	var ent entity.Directory
	if err := ctx.DB().Where("app = ? AND path = ?", app, path).First(&ent).Error; err != nil {
		return nil, err
	}
	return &ent, nil
}

// ListDirectory 全量取回某 app 的目录
func (r Directory) ListDirectory(ctx core.Context, app string) ([]*entity.Directory, error) {
	var list []*entity.Directory
	err := ctx.DB().Where("app = ?", app).Order("path asc").Find(&list).Error
	return list, err
}

// ListDirectoryInPaths 按路径集合批量查询
func (r Directory) ListDirectoryInPaths(ctx core.Context, app string, paths []string) ([]*entity.Directory, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	var list []*entity.Directory
	err := ctx.DB().Where("app = ? AND path IN ?", app, paths).Find(&list).Error
	return list, err
}

// CreateDirectoryIgnore 幂等新建，返回最终行。
func (r Directory) CreateDirectoryIgnore(ctx core.Context, ent *entity.Directory) (*entity.Directory, error) {
	if err := ctx.DB().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "app"}, {Name: "path"}},
		DoNothing: true,
	}).Create(ent).Error; err != nil {
		return nil, err
	}
	if ent.Id == 0 {
		return r.GetDirectoryByPath(ctx, ent.App, ent.Path)
	}
	return ent, nil
}

// UpdateDirectory 按主键+app 更新目录实体。零值可写列（accept）在 entity 里是指针，非 nil 即写入，直接 Updates(ent)。
func (r Directory) UpdateDirectory(ctx core.Context, ent *entity.Directory) error {
	return ctx.DB().Model(&entity.Directory{}).
		Where("id = ? AND app = ?", ent.Id, ent.App).
		Updates(ent).Error
}

// MoveDirectory 移动子树：一条 UPDATE 重写整棵子树 path，并把 updated_at/updated_by 刷到本次操作。
func (r Directory) MoveDirectory(ctx core.Context, p *types.MoveDirectoryParams) error {
	var cur entity.Directory
	if err := ctx.DB().Where("app = ? AND id = ?", p.App, p.Id).First(&cur).Error; err != nil {
		return err
	}
	if cur.Path == p.NewPath {
		return nil
	}

	// 唯一键冲突表示目标位置已存在同名目录，交由上层分类。
	err := ctx.DB().Exec(moveTreeSQL, p.NewPath, cur.Path, time.Now().Unix(), p.UpdatedBy, p.App, p.Id, pattern.PrefixPattern(cur.Path)).Error
	if err != nil {
		return err
	}
	return nil
}

// DeleteDirectoryTree 删子树：一条语句删掉自己与全部后代
func (r Directory) DeleteDirectoryTree(ctx core.Context, app string, id uint32, path string) error {
	return ctx.DB().Exec(deleteTreeSQL, app, id, pattern.PrefixPattern(path)).Error
}

// 编译期断言：*Directory 实现 repository.Directory
var _ repository.Directory = (*Directory)(nil)
