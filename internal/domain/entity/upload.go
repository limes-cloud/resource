package entity

import "github.com/limes-cloud/kratosx/model"

// Upload 分片上传会话
type Upload struct {
	StoreKeyword string `json:"storeKeyword" gorm:"column:store_keyword"` // 存储后端标识
	Md5          string `json:"md5"          gorm:"column:md5"`           // 内容摘要（分片摘要的摘要）
	Ext          string `json:"ext"          gorm:"column:ext"`           // 后缀
	UserId       uint32 `json:"userId"       gorm:"column:user_id"`       // 发起用户ID，必须 > 0
	DirectoryId  uint32 `json:"directoryId"  gorm:"column:directory_id"`  // 目标目录ID
	Name         string `json:"name"         gorm:"column:name"`          // 业务文件名
	UploadId     string `json:"uploadId"     gorm:"column:upload_id"`     // 厂商分片上传ID
	TotalSize    uint64 `json:"totalSize"    gorm:"column:total_size"`    // 声明总大小（字节）
	model.BaseModel
}
