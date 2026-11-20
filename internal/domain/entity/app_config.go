package entity

import "github.com/limes-cloud/kratosx/model"

// AppConfig 应用存储配置
type AppConfig struct {
	App                string  `json:"app"                gorm:"column:app"`                  // 应用标识
	StoreKeyword       string  `json:"storeKeyword"       gorm:"column:store_keyword"`        // 上传落地的存储后端
	ExportStoreKeyword string  `json:"exportStoreKeyword" gorm:"column:export_store_keyword"` // 导出产物的存储后端；必填，不为空
	DefaultAccept      *string `json:"defaultAccept"      gorm:"column:default_accept"`       // 默认允许后缀：'*'全部|''不允许任何类型|后缀白名单；'' 是合法零值，用指针避免更新被跳过
	DefaultMaxSize     uint32  `json:"defaultMaxSize"     gorm:"column:default_max_size"`     // 默认单文件上限（MB）；4294967295 = 不限
	TempUrlTtl         uint32  `json:"tempUrlTtl"         gorm:"column:temp_url_ttl"`         // 临时URL有效期（秒）
	ExportTtl          uint32  `json:"exportTtl"          gorm:"column:export_ttl"`           // 导出产物保留期（秒）
	Quota              *uint64 `json:"quota"              gorm:"column:quota"`                // 存储配额（字节）；0=不限；0 是合法值，用指针避免更新被跳过
	UsedBytes          uint64  `json:"usedBytes"          gorm:"column:used_bytes"`           // 已用量（字节）；仅经原子表达式维护，不走 Updates
	Status             *uint8  `json:"status"             gorm:"column:status"`               // 状态：1启用 0停用；0 是合法零值，用指针避免更新被跳过
	UpdatedBy          uint32  `json:"updatedBy"          gorm:"column:updated_by"`           // 最后修改人
	model.BaseModel
}
