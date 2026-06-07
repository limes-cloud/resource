package dbs

import (
	"errors"
	"path/filepath"
	"strings"
	"sync"

	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
)

type File struct{}

var (
	fileIns  *File
	fileOnce sync.Once
)

func NewFile() *File {
	fileOnce.Do(func() {
		fileIns = &File{}
	})
	return fileIns
}

// GetFileBySha 获取指定数据
func (r File) GetFileBySha(ctx core.Context, store string, sha string) (*entity.File, error) {
	var (
		file = entity.File{}
		fs   = []string{"*"}
	)
	return &file, ctx.DB().Select(fs).Where("store = ? and sha = ?", store, sha).First(&file).Error
}

// GetFileByKey 获取指定数据
func (r File) GetFileByKey(ctx core.Context, key string) (*entity.File, error) {
	sha := strings.TrimSuffix(key, filepath.Ext(key))
	arr := strings.Split(key, "/")
	if len(arr) != 2 {
		return nil, errors.New("key is error")
	}
	return r.GetFileBySha(ctx, arr[0], sha)
}

func (r File) GetFileByUploadId(ctx core.Context, uid string) (*entity.File, error) {
	var (
		file = entity.File{}
		fs   = []string{"*"}
	)
	return &file, ctx.DB().Select(fs).Where("upload_id = ?", uid).First(&file).Error
}

// GetFile 获取指定的数据
func (r File) GetFile(ctx core.Context, id uint32) (*entity.File, error) {
	var (
		file = entity.File{}
		fs   = []string{"*"}
	)
	return &file, ctx.DB().Select(fs).First(&file, id).Error
}

// CreateFile 创建数据
func (r File) CreateFile(ctx core.Context, file *entity.File) (uint32, error) {
	return file.Id, ctx.DB().Create(file).Error
}

// UpdateFile 更新数据
func (r File) UpdateFile(ctx core.Context, file *entity.File) error {
	return ctx.DB().Where("id = ?", file.Id).Updates(file).Error
}

// DeleteFile 删除数据
func (r File) DeleteFile(ctx core.Context, ids []uint32, call func(file *entity.File)) (uint32, error) {
	var files []*entity.File
	if err := ctx.DB().Where("id in ?", ids).Find(&files).Error; err != nil {
		return 0, err
	}

	for _, item := range files {
		call(item)
	}

	db := ctx.DB().Where("id in ?", ids).Delete(entity.File{})
	return uint32(db.RowsAffected), db.Error
}

