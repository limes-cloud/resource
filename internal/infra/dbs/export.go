package dbs

import (
	"fmt"
	"github.com/limes-cloud/resource/internal/core"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/types"
)

type Export struct {
}

var (
	exportIns  *Export
	exportOnce sync.Once
)

func NewExport() *Export {
	exportOnce.Do(func() {
		exportIns = &Export{}
	})
	return exportIns
}

func (r Export) CopyExport(ctx core.Context, export *entity.Export, req *types.CopyExportRequest) (uint32, error) {
	conf := ctx.Config()
	exp := entity.Export{
		Name:      req.Name,
		Size:      export.Size,
		Sha:       export.Sha,
		Key:       export.Key,
		Status:    export.Status,
		ExpiredAt: time.Now().Unix() + int64(conf.Export.Expire.Seconds()),
	}
	return r.CreateExport(ctx, &exp)
}

func (r Export) GetKeyByValue(ctx core.Context, value string) (string, error) {
	key := value
	if strings.Contains(value, "/") {
		key = value[strings.Index(value, "/")+1:]
	} else if !strings.Contains(value, ".") {
		if err := ctx.DB().Model(entity.File{}).Select("key").Where("sha=?", value).Scan(&key).Error; err != nil {
			return "", err
		}
	}
	return key, nil
}

// ListExport 获取列表
func (r Export) ListExport(ctx core.Context, req *types.ListExportRequest) ([]*entity.Export, uint32, error) {
	var (
		list  []*entity.Export
		total int64
		fs    = []string{"*"}
	)

	db := ctx.DB().Model(entity.Export{}).Select(fs)

	if req.Name != nil {
		db = db.Where("name like ?", *req.Name+"%")
	}
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}
	if req.ExpiredAt != nil {
		db = db.Where("expired_at <= ?", *req.ExpiredAt)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	db = db.Offset(int((req.Page - 1) * req.PageSize)).Limit(int(req.PageSize))

	if req.OrderBy == nil || *req.OrderBy == "" {
		req.OrderBy = proto.String("id")
	}
	if req.Order == nil || *req.Order == "" {
		req.Order = proto.String("asc")
	}
	db = db.Order(fmt.Sprintf("%s %s", *req.OrderBy, *req.Order))
	if *req.OrderBy != "id" {
		db = db.Order("id asc")
	}

	return list, uint32(total), db.Find(&list).Error
}

// CreateExport 创建数据
func (r Export) CreateExport(ctx core.Context, export *entity.Export) (uint32, error) {
	return export.Id, ctx.DB().Create(export).Error
}

// DeleteExport 删除数据
func (r Export) DeleteExport(ctx core.Context, ids []uint32) (uint32, error) {
	db := ctx.DB().Where("id in ?", ids).Delete(&entity.Export{})
	return uint32(db.RowsAffected), db.Error
}

// GetExportBySha 获取指定数据
func (r Export) GetExportBySha(ctx core.Context, sha string) (*entity.Export, error) {
	var (
		export = entity.Export{}
		fs     = []string{"*"}
	)
	return &export, ctx.DB().Select(fs).Where("sha = ?", sha).First(&export).Error
}

// GetExport 获取指定的数据
func (r Export) GetExport(ctx core.Context, id uint32) (*entity.Export, error) {
	var (
		export = entity.Export{}
		fs     = []string{"*"}
	)
	return &export, ctx.DB().Select(fs).First(&export, id).Error
}

// UpdateExport 更新数据
func (r Export) UpdateExport(ctx core.Context, export *entity.Export) error {
	return ctx.DB().Where("id=?", export.Id).Updates(export).Error
}

// GetExportFileKeyById 获取导出的key
func (r Export) GetExportFileKeyById(ctx core.Context, id uint32) (string, error) {
	var key string
	if err := ctx.DB().Model(entity.File{}).Select("key").Where("id=?", id).Scan(&key).Error; err != nil {
		return "", err
	}
	return key, nil
}

func (r Export) GetExportFileCount(ctx core.Context, req *types.GetExportFileCountRequest) (int64, error) {
	var count int64
	return count, ctx.DB().Model(entity.Export{}).
		Where("sha=?", req.Sha).
		Where("status=?", req.Status).
		Count(&count).Error
}
