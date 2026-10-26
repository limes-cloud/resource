package types

// ListStoreRequest 存储后端列表查询请求
type ListStoreRequest struct {
	PageRequest

	Keyword  *string `json:"keyword"`  // 可选：按后端标识精确过滤
	Provider *string `json:"provider"` // 可选：按厂商标识过滤
	Status   *uint32 `json:"status"`   // 可选：按状态过滤（1启用 0停用）
}

// ListProviderRequest 厂商清单查询请求（无入参，保留类型以便后续扩展）
type ListProviderRequest struct{}

// UpdateStoreRequest 存储后端局部更新请求（nil = 该列不改；id 为 WHERE 条件）
type UpdateStoreRequest struct {
	Id            uint32
	Provider      *string
	Name          *string
	Endpoint      *string
	Region        *string
	Bucket        *string
	AK            *string
	SK            *string // nil 或 "" = 不修改
	PublicBaseUrl *string
	CdnAuthType   *string
	CdnAuthKey    *string
	Status        *uint32
	Description   *string
	UpdatedBy     uint32
}
