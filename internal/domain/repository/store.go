package repository

import (
	"io"
	"time"

	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/types"
)

// Store 存储后端仓储接口。
type Store interface {
	// ListStore 分页查询存储后端（返回完整行含明文 sk；响应侧掩码由 app 层 toStore 负责）
	ListStore(ctx core.Context, req *types.ListStoreRequest) ([]*entity.Store, uint32, error)
	// GetStore 按主键查询存储后端（返回完整行含明文 sk；UpdateStore 读改存依赖此处的真实 sk，勿在此掩码）
	GetStore(ctx core.Context, id uint32) (*entity.Store, error)
	// GetStoreByKeyword 按 keyword 取存储后端（含 status，用于启用态判定）
	GetStoreByKeyword(ctx core.Context, keyword string) (*entity.Store, error)
	// CreateStore 新建存储后端
	CreateStore(ctx core.Context, ent *entity.Store) (uint32, error)
	// UpdateStore 更新存储后端（字段白名单，不含 keyword；无删除方法）
	UpdateStore(ctx core.Context, ent *entity.Store) error

	// GetClient 取厂商客户端（缓存读行→构造，读路径零网络跳）
	GetClient(keyword string) (Client, bool)
	// ListClients 取全部客户端（GC / 离线核对遍历用）
	ListClients() []Client
}

// Client 厂商对象存储客户端。
type Client interface {
	// Keyword 返回该客户端对应的 store.keyword
	Keyword() string

	// Put 覆盖写入一份字节
	Put(ctx core.Context, in *PutInput) error
	// Get 读取一份字节
	Get(ctx core.Context, key string) (io.ReadCloser, error)
	// Head 取对象元信息；对象不存在时返回 ErrObjectNotFound
	Head(ctx core.Context, key string) (*ObjectInfo, error)
	// Check 连通性自检：HeadBucket 探测（200 = 认证/桶/网络全通；仅配置解析，不写任何数据）
	Check(ctx core.Context) error
	// Delete 删除对象；对象不存在也返回 nil（S3 系 DELETE 幂等）
	Delete(ctx core.Context, key string) error
	// DeleteBatch 批量删除对象（内部按厂商上限分批，如 S3 一次 1000）；对象不存在不报错
	DeleteBatch(ctx core.Context, keys []string) error

	// NewMultipart 发起分片上传并返回厂商 uploadId（对象元数据只能在此刻写死）。
	NewMultipart(ctx core.Context, key, contentType string) (uploadID string, err error)
	// ListParts 列出已传分片（实现必须全量分页、按片号升序）；会话已在厂商侧结束时返回 ErrNoSuchUpload
	ListParts(ctx core.Context, key, uploadID string) ([]UploadedPart, error)
	// Complete 合并分片；会话已在厂商侧结束时返回 ErrNoSuchUpload
	Complete(ctx core.Context, key, uploadID string, parts []CompletedPart) error
	// Abort 释放分片；会话已在厂商侧结束时返回 ErrNoSuchUpload（幂等）
	Abort(ctx core.Context, key, uploadID string) error
	// PresignPart 生成分片上传的预签名地址。
	PresignPart(ctx core.Context, in *PresignPartInput) (string, error)

	// GenTemporaryURL 生成读侧临时地址。
	GenTemporaryURL(ctx core.Context, key string, ttl time.Duration) (string, error)
}

// PutInput 覆盖写入对象的参数
type PutInput struct {
	Key         string
	Body        io.Reader
	Size        int64
	ContentType string
}

// PresignPartInput 生成分片预签名地址的参数
type PresignPartInput struct {
	Key        string
	UploadID   string
	PartNumber uint32
	Ttl        time.Duration
}

// ObjectInfo 对象元信息
type ObjectInfo struct {
	Size        uint64 // 字节数
	ContentType string // 内容类型
	ETag        string // 厂商 ETag
}

// UploadedPart 已传分片（来自厂商 ListParts）。
type UploadedPart struct {
	Number uint32 // 片号，从 1 起
	Size   uint64 // 该片字节数
	ETag   string // 该片 ETag（厂商原值）
}

// CompletedPart 合并时提交的分片（ETag 原样回传厂商）
type CompletedPart struct {
	Number uint32 // 片号，从 1 起
	ETag   string // 该片 ETag（厂商原值）
}
