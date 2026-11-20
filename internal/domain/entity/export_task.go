package entity

import "github.com/limes-cloud/kratosx/model"

// 导出任务状态
const (
	ExportStatusPending uint8 = 0 // 待执行
	ExportStatusRunning uint8 = 1 // 执行中
	ExportStatusSuccess uint8 = 2 // 成功
	ExportStatusFailed  uint8 = 3 // 失败
	ExportStatusExpired uint8 = 4 // 已过期
)

// ExportTask 导出产出物与生产状态
type ExportTask struct {
	App          string  `json:"app"          gorm:"column:app"`           // 应用标识
	Md5          string  `json:"md5"          gorm:"column:md5"`           // 产出规格摘要
	StoreKeyword string  `json:"storeKeyword" gorm:"column:store_keyword"` // 产出物实际落地的存储后端
	Size         uint64  `json:"size"         gorm:"column:size"`          // 产出物大小（字节）
	Status       uint8   `json:"status"       gorm:"column:status"`        // 状态：0待执行 1执行中 2成功 3失败 4已过期
	Retry        uint8   `json:"retry"        gorm:"column:retry"`         // 已尝试次数（含首次），领取时 +1
	Reason       *string `json:"reason"       gorm:"column:reason"`        // 最后一次失败原因；成功时 NULL
	ExpiredAt    int64   `json:"expiredAt"    gorm:"column:expired_at"`    // 产出物过期时间（Unix 秒）；0 = 未成功
	LeasedAt     int64   `json:"leasedAt"     gorm:"column:leased_at"`     // worker 最后一次领取或心跳时间（Unix 秒）；0 = 从未领取
	FailedAt     int64   `json:"failedAt"     gorm:"column:failed_at"`     // 本轮退避计时起点（Unix 秒）；0 = 本轮未失败过
	model.BaseModel
}
