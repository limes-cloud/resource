package entity

import (
	"github.com/limes-cloud/kratosx/model"
)

type Export struct {
	Name      string  `json:"name" gorm:"column:name"`
	Size      uint32  `json:"size" gorm:"column:size"`
	Sha       string  `json:"sha" gorm:"column:sha"`
	Key       string  `json:"key" gorm:"column:key"`
	Status    string  `json:"status" gorm:"column:status"`
	Reason    *string `json:"reason" gorm:"column:reason"`
	ExpiredAt int64   `json:"expiredAt" gorm:"column:expired_at"`
	Url       string  `json:"url" gorm:"-"`
	model.BaseTenantUserModel
}
