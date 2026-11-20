package entity

import "github.com/limes-cloud/kratosx/model"

// Object 物理对象（内容寻址）
type Object struct {
	StoreKeyword string `json:"storeKeyword" gorm:"column:store_keyword"` // 存储后端标识
	Md5          string `json:"md5"          gorm:"column:md5"`           // 内容摘要（分片摘要的摘要），32位小写hex
	Ext          string `json:"ext"          gorm:"column:ext"`           // 后缀，小写字母数字；空串 = 无后缀
	Size         uint64 `json:"size"         gorm:"column:size"`          // 对象大小（字节）
	model.BaseModel
}
