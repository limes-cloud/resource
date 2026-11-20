package types

// FileItem 文件列表项（含派生 URL）
type FileItem struct {
	Id          uint32 `json:"id"`          // 文件行 ID
	App         string `json:"app"`         // 应用标识
	UserId      uint32 `json:"userId"`      // 上传用户 ID
	DirectoryId uint32 `json:"directoryId"` // 目录 ID
	Name        string `json:"name"`        // 业务文件名
	ObjectId    uint32 `json:"objectId"`    // 物理对象 ID
	Size        uint64 `json:"size"`        // 对象大小（字节）
	Ext         string `json:"ext"`         // 后缀（来自 object，显示用）
	Url         string `json:"url"`         // 派生可读 URL
	Key         string `json:"key"`         // 对象 key（md5.ext），业务持久化只用它
	CreatedAt   int64  `json:"createdAt"`   // 创建时间（Unix 秒）
}

// GetFileRequest 取单个文件
type GetFileRequest struct {
	Id uint32 `json:"id"`
}

// ListFileByDirectoryRequest 目录内文件分页
type ListFileByDirectoryRequest struct {
	App         string `json:"app"`
	DirectoryId uint32 `json:"directoryId"`
	Page        uint32 `json:"page"`
	PageSize    uint32 `json:"pageSize"`
}

// ListFileByUserRequest 我的文件分页
type ListFileByUserRequest struct {
	App      string `json:"app"`
	UserId   uint32 `json:"userId"`
	Page     uint32 `json:"page"`
	PageSize uint32 `json:"pageSize"`
}

// UpdateFileRequest 改名 / 移动（nil = 该列不改；id/app 为归属与 WHERE 条件）
type UpdateFileRequest struct {
	App         string  `json:"app"`
	Id          uint32  `json:"id"`
	Name        *string `json:"name"`
	DirectoryId *uint32 `json:"directoryId"`
}

// GetFileURLByKeyReply 按对象 key 取可访问地址。
type GetFileURLByKeyReply struct {
	Url string `json:"url"` // 可访问地址（按 store 的 CDN 鉴权类型生成，TTL 内有效）
}
