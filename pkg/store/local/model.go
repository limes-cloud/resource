package local

import (
	"github.com/limes-cloud/kratosx/types"
	"gorm.io/gorm"

	"github.com/limes-cloud/resource/internal/model"
)

type Chunk struct {
	types.CreateModel
	UploadID string      `json:"upload_id" gorm:"uniqueIndex:ui;not null;size:128;comment:上传id"`
	Index    int         `json:"index" gorm:"uniqueIndex:ui;not null;comment:切片下标"`
	Sha      string      `json:"sha" gorm:"not null;size:128;comment:切片sha"`
	Data     string      `json:"data" gorm:"not null;type:mediumblob;comment:切片数据"`
	Size     int         `json:"size" gorm:"not null;comment:切片大小"`
	File     *model.File `json:"file" gorm:"foreignKey:upload_id;references:upload_id;constraint:onDelete:cascade;"`
}

func (c *Chunk) Copy(db *gorm.DB, uploadId string, index int) error {
	nc := Chunk{
		Sha:      c.Sha,
		Data:     c.Data,
		Size:     c.Size,
		UploadID: uploadId,
		Index:    index,
	}
	return db.Create(nc).Error
}

func (c *Chunk) Add(db *gorm.DB) error {
	return db.Create(c).Error
}

func (c *Chunk) OneBySha(db *gorm.DB, sha string) error {
	return db.First(c, "sha=?", sha).Error
}

func (c *Chunk) Parts(db *gorm.DB, uploadId string) ([]*Chunk, error) {
	var chunks []*Chunk
	return chunks, db.Model(c).Order("`index`").Find(&chunks, "upload_id=?", uploadId).Error
}

func (c *Chunk) Delete(db *gorm.DB, uploadId string) error {
	return db.Delete(Chunk{}, "upload_id=?", uploadId).Error
}
