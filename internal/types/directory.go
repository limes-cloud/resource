package types

// CreateDirectoryRequest 创建目录请求（按完整路径创建）
type CreateDirectoryRequest struct {
	App  string `json:"app"`  // 应用标识
	Path string `json:"path"` // 完整路径，如 /图片/头像
}

// UpdateDirectoryRequest 更新目录请求（改名 / 移动 / 改策略）。
type UpdateDirectoryRequest struct {
	App       string  `json:"app"`       // 应用标识
	Id        uint32  `json:"id"`        // 目录 ID
	Path      *string `json:"path"`      // 可选：新完整路径（改名或移动）
	Accept    *string `json:"accept"`    // 可选：新允许后缀
	MaxSize   *uint32 `json:"maxSize"`   // 可选：新单文件上限（MB）
	UpdatedBy uint32  `json:"updatedBy"` // 最后修改人
}

// ListDirectoryRequest 目录列表查询请求（全量取回某 app 的目录，内存建树）
type ListDirectoryRequest struct {
	App string `json:"app"` // 应用标识
}

// MoveDirectoryParams 移动子树参数
type MoveDirectoryParams struct {
	App       string // 应用标识
	Id        uint32 // 被移动的子树根目录 ID
	NewPath   string // 新完整路径
	UpdatedBy uint32 // 最后修改人
}
