package entity

import "github.com/limes-cloud/kratosx/model"

// File 业务文件条目
type File struct {
	App         string  `json:"app"         gorm:"column:app"`          // 应用标识
	UserId      uint32  `json:"userId"      gorm:"column:user_id"`      // 上传用户ID，必须 > 0
	DirectoryId uint32  `json:"directoryId" gorm:"column:directory_id"` // 目录ID
	Name        string  `json:"name"        gorm:"column:name"`         // 业务文件名
	ObjectId    uint32  `json:"objectId"    gorm:"column:object_id"`    // 物理对象ID
	UploadId    *string `json:"uploadId"    gorm:"column:upload_id"`    // 上传会话ID；NULL = 秒传
	Status      *uint8  `json:"status"      gorm:"column:status;default:1"` // 1正常 2待删除（软删，后台回收）；指针避免 Updates 跳过，建行 nil 时取 DB 默认 1
	model.BaseModel
}
