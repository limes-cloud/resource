package types

// ListAppConfigRequest 应用配置列表查询请求
type ListAppConfigRequest struct {
	PageRequest

	App    *string `json:"app"`    // 可选：按应用标识精确过滤
	Status *uint32 `json:"status"` // 可选：按状态过滤（1启用 0停用）
}

// UpdateAppConfigRequest 应用配置局部更新请求（nil = 该列不改；app 为 WHERE 条件）
type UpdateAppConfigRequest struct {
	App                string
	StoreKeyword       *string
	ExportStoreKeyword *string
	DefaultAccept      *string
	DefaultMaxSize     *uint32
	TempUrlTtl         *uint32
	ExportTtl          *uint32
	Quota              *uint64 // nil=不改；0=不限
	Status             *uint32
	UpdatedBy          uint32
}
