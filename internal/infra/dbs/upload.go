package dbs

import (
	"sync"
	"time"

	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/repository"
	"github.com/limes-cloud/resource/internal/types"
)

// Upload 分片上传会话仓储实现
type Upload struct{}

var (
	uploadIns  *Upload
	uploadOnce sync.Once
)

// NewUpload 取分片上传会话仓储单例
func NewUpload() *Upload {
	uploadOnce.Do(func() { uploadIns = &Upload{} })
	return uploadIns
}

// GetUpload 命中会话；必须带 user_id 做鉴权，查不到即"会话不存在"。
func (r Upload) GetUpload(ctx core.Context, uploadID string, userID uint32) (*entity.Upload, error) {
	var ent entity.Upload
	if err := ctx.DB().Where("upload_id = ? AND user_id = ?", uploadID, userID).First(&ent).Error; err != nil {
		return nil, err
	}
	return &ent, nil
}

// GetUploadByKey 按会话身份 (store_keyword, md5, user_id, directory_id) 召回在飞会话。
func (r Upload) GetUploadByKey(ctx core.Context, key *types.UploadKey) (*entity.Upload, error) {
	var ent entity.Upload
	if err := ctx.DB().Where("store_keyword = ? AND md5 = ? AND user_id = ? AND directory_id = ?", key.StoreKeyword, key.Md5, key.UserID, key.DirectoryID).First(&ent).Error; err != nil {
		return nil, err
	}
	return &ent, nil
}

// CreateUpload 建会话
func (r Upload) CreateUpload(ctx core.Context, ent *entity.Upload) (uint32, error) {
	if err := ctx.DB().Create(ent).Error; err != nil {
		return 0, err
	}
	return ent.Id, nil
}

// TouchUpload 顶 updated_at（滑动过期时钟）。显式写列，不依赖空 map 触发自动时间戳。
func (r Upload) TouchUpload(ctx core.Context, id uint32) error {
	return ctx.DB().Model(&entity.Upload{}).Where("id = ?", id).
		UpdateColumn("updated_at", time.Now().Unix()).Error
}

// DeleteUpload 删会话
func (r Upload) DeleteUpload(ctx core.Context, id uint32) error {
	return ctx.DB().Where("id = ?", id).Delete(&entity.Upload{}).Error
}

// ListExpiredUpload 扫过期会话（GC 用），按 updated_at 升序。
func (r Upload) ListExpiredUpload(ctx core.Context, before int64, limit int) ([]*entity.Upload, error) {
	var list []*entity.Upload
	err := ctx.DB().Where("updated_at < ?", before).Order("updated_at asc").Limit(limit).Find(&list).Error
	return list, err
}

// 编译期断言：*Upload 实现 repository.Upload
var _ repository.Upload = (*Upload)(nil)
