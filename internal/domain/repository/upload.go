package repository

import (
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/types"
)

// Upload 分片上传会话仓储接口。
type Upload interface {
	// GetUpload 命中会话；必须带 user_id 做鉴权，查不到即"会话不存在"。
	GetUpload(ctx core.Context, uploadID string, userID uint32) (*entity.Upload, error)
	// GetUploadByKey 按会话身份 (store_keyword, md5, user_id, directory_id) 召回在飞会话。
	GetUploadByKey(ctx core.Context, key *types.UploadKey) (*entity.Upload, error)
	// CreateUpload 建会话；撞 uidx_upload_session 即触发召回（调用方重查、验活、复用既有行）。
	CreateUpload(ctx core.Context, ent *entity.Upload) (uint32, error)
	// TouchUpload 顶 updated_at（滑动过期时钟，传片取址 / 召回命中时调用）。
	TouchUpload(ctx core.Context, id uint32) error
	// DeleteUpload 删会话（完成事务内 / Abort 后）。
	DeleteUpload(ctx core.Context, id uint32) error
	// ListExpiredUpload 扫过期会话（GC 用），按 updated_at 升序。
	ListExpiredUpload(ctx core.Context, before int64, limit int) ([]*entity.Upload, error)
}
