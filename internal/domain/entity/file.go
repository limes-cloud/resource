package entity

import (
	"github.com/limes-cloud/kratosx/types"
)

type File struct {
	DirectoryId uint32 `json:"directoryId" gorm:"column:directory_id"`
	Store       string `json:"store" gorm:"column:store"`
	Name        string `json:"name" gorm:"column:name"`
	Type        string `json:"type" gorm:"column:type"`
	Size        uint32 `json:"size" gorm:"column:size"`
	Sha         string `json:"sha" gorm:"column:sha"`
	Key         string `json:"key" gorm:"column:key"`
	Src         string `json:"src" gorm:"column:src"`
	Status      string `json:"status" gorm:"column:status"`
	UploadId    string `json:"uploadId" gorm:"column:upload_id"`
	ChunkCount  uint32 `json:"chunkCount" gorm:"column:chunk_count"`
	Url         string `json:"url" gorm:"-"`
	types.BaseModel
}
