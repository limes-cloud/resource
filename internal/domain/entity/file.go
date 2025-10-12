package entity

import (
	"github.com/limes-cloud/kratosx/model"
)

type File struct {
	Type       string `json:"type" gorm:"column:type"`
	Size       uint32 `json:"size" gorm:"column:size"`
	Sha        string `json:"sha" gorm:"column:sha"`
	Key        string `json:"key" gorm:"column:key"`
	Status     string `json:"status" gorm:"column:status"`
	UploadId   string `json:"uploadId" gorm:"column:upload_id"`
	ChunkCount uint32 `json:"chunkCount" gorm:"column:chunk_count"`
	model.BaseModel
}

type UserFile struct {
	DirectoryId uint32 `json:"directoryId" gorm:"column:directory_id"`
	FileId      uint32 `json:"fileId" gorm:"column:file_id"`
	Name        string `json:"name" gorm:"column:name"`
	File        *File  `json:"file"`
	model.BaseTenantUserModel
}
