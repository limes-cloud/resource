package entity

import "github.com/limes-cloud/kratosx/model"

// Export 用户导出请求记录
type Export struct {
	App    string `json:"app"    gorm:"column:app"`     // 应用标识
	TaskId uint32 `json:"taskId" gorm:"column:task_id"` // 导出产出物ID
	UserId uint32 `json:"userId" gorm:"column:user_id"` // 发起用户ID，0 = 系统
	Name   string `json:"name"   gorm:"column:name"`    // 用户可见名称
	model.BaseModel
}
