package local

import (
	"github.com/limes-cloud/kratosx/types"
	"gorm.io/gorm"
)

type Chunk struct {
	types.CreateModel
	UploadID string `json:"upload_id"`
	Index    int    `json:"index"`
	Sha      string `json:"sha"`
	Data     string `json:"data"`
	Size     int    `json:"size"`
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
