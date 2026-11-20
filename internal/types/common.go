package types

// 分页默认值与上限
const (
	DefaultPage     uint32 = 1   // 默认页码
	DefaultPageSize uint32 = 20  // 默认每页条数
	MaxPageSize     uint32 = 100 // 每页条数上限
)

// PageRequest 分页请求（内部转换用，不暴露给 proto 层）
type PageRequest struct {
	Page     uint32 `json:"page"`     // 页码，从 1 起；0 视为 1
	PageSize uint32 `json:"pageSize"` // 每页条数；0 视为 20，上限 100
}

// Normalize 归一化分页参数，返回 SQL 用的 (offset, limit)
func (p PageRequest) Normalize() (offset, limit int) {
	page := p.Page
	if page == 0 {
		page = DefaultPage
	}
	size := p.PageSize
	if size == 0 {
		size = DefaultPageSize
	}
	if size > MaxPageSize {
		size = MaxPageSize
	}
	return int((page - 1) * size), int(size)
}
