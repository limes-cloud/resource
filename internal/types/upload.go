package types

// GetUploadChunkSizeRequest 取分片大小请求
type GetUploadChunkSizeRequest struct {
	TotalSize uint64 `json:"totalSize"` // 文件总大小（字节）
}

// UploadKey 在飞上传会话的身份键（召回唯一会话）
type UploadKey struct {
	StoreKeyword string // 存储后端标识
	Md5          string // 内容摘要
	UserID       uint32 // 上传用户
	DirectoryID  uint32 // 目标目录
}

// GetUploadChunkSizeResponse 取分片大小响应
type GetUploadChunkSizeResponse struct {
	ChunkSize  uint32 `json:"chunkSize"`  // 单分片大小（字节）
	ChunkCount uint32 `json:"chunkCount"` // 分片数
}

// PrepareUploadFileRequest 预上传（秒传判定 + 落桶解析）请求
type PrepareUploadFileRequest struct {
	App  string `json:"app"`  // 应用标识
	Path string `json:"path"` // 目标目录路径，如 /图片/头像
	Name string `json:"name"` // 业务文件名
	Ext  string `json:"ext"`  // 后缀（小写字母数字；空串 = 无后缀）
	Md5  string `json:"md5"`  // 内容摘要（分片摘要的摘要）
	Size uint64 `json:"size"` // 声明总大小（字节）
}

// PrepareUploadFileResponse 预上传响应
type PrepareUploadFileResponse struct {
	UploadId      string   `json:"uploadId"`      // 分片上传会话ID（秒传时为空）
	Md5           string   `json:"md5"`           // 内容摘要（与请求一致）
	ChunkSize     uint32   `json:"chunkSize"`     // 单分片大小（字节）
	ChunkCount    uint32   `json:"chunkCount"`    // 分片数
	UploadedParts []uint32 `json:"uploadedParts"` // 已传片号（召回命中时返回；新建 / 秒传为空）
	Skipped       bool     `json:"skipped"`       // 是否秒传（true = 已直接建 file，零字节）
	Url           string   `json:"url"`           // 秒传时的文件 URL
	Key           string   `json:"key"`           // 对象 key（md5.ext），业务持久化只用它
}

// GetUploadPartURLsRequest 取分片上传地址请求
type GetUploadPartURLsRequest struct {
	UploadId    string   `json:"uploadId"`    // 分片上传会话ID
	PartNumbers []uint32 `json:"partNumbers"` // 本次要取地址的片号（1..chunkCount）
}

// PartURL 单个分片的上传地址
type PartURL struct {
	PartNumber uint32 `json:"partNumber"`
	Url        string `json:"url"`
}

// GetUploadPartURLsResponse 取分片上传地址响应
type GetUploadPartURLsResponse struct {
	Urls []PartURL `json:"urls"`
}

// CompleteUploadRequest 完成分片上传请求
type CompleteUploadRequest struct {
	UploadId string `json:"uploadId"`
}

// CompleteUploadResponse 完成分片上传响应
type CompleteUploadResponse struct {
	Url string `json:"url"` // 文件 URL
	Key string `json:"key"` // 对象 key（md5.ext），业务持久化只用它
}

// UploadFileByURLRequest 服务端从 URL 下载并上传请求（app 层注入 App，其余字段来自 proto）
type UploadFileByURLRequest struct {
	App           string  `json:"app"`           // 应用标识（app 层注入）
	Url           string  `json:"url"`           // 远端下载地址（仅 http/https）
	Name          string  `json:"name"`          // 文件名（含后缀，如 a.png；ext 由其推导）
	DirectoryPath *string `json:"directoryPath"` // 目标目录路径（空 = 根目录）
}

// UploadFileByURLResponse 服务端 URL 上传响应
type UploadFileByURLResponse struct {
	Key string `json:"key"` // 对象 key（md5.ext），内容指纹唯一、持久可引用（业务存 key，不存临时 URL）
}
