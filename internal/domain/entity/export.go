package entity

import (
	"github.com/limes-cloud/kratosx/types"
)

type Export struct {
	UserId       uint32  `json:"userId" gorm:"column:user_id"`
	DepartmentId uint32  `json:"departmentId" gorm:"column:department_id"`
	Scene        string  `json:"scene" gorm:"column:scene"`
	Name         string  `json:"name" gorm:"column:name"`
	Size         uint32  `json:"size" gorm:"column:size"`
	Sha          string  `json:"sha" gorm:"column:sha"`
	Src          string  `json:"src" gorm:"column:src"`
	Status       string  `json:"status" gorm:"column:status"`
	Reason       *string `json:"reason" gorm:"column:reason"`
	ExpiredAt    int64   `json:"expiredAt" gorm:"column:expired_at"`
	Url          string  `json:"url" gorm:"-"`
	types.BaseModel
}
