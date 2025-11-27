package dbs

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/limes-cloud/resource/internal/core"

	"google.golang.org/protobuf/proto"

	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/types"
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

// ListFile 获取列表
func (r File) ListFile(ctx core.Context, req *types.ListFileRequest) ([]*entity.File, uint32, error) {
	var (
		list  []*entity.File
		total int64
		fs    = []string{"*"}
	)

	db := ctx.DB().Model(entity.File{}).Select(fs)

	if req.DirectoryId != nil {
		db = db.Where("directory_id = ?", *req.DirectoryId)
	}
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}
	if req.Name != nil && *req.Name != "" {
		db = db.Where("name like ", *req.Name+"%")
	}
	if len(req.KeyList) != 0 {
		shaList := make([]string, len(req.KeyList))
		// 去除文件后缀
		for i, key := range req.KeyList {
			shaList[i] = strings.TrimSuffix(key, filepath.Ext(key))
		}
		db = db.Where("sha in ?", shaList)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	db = db.Offset(int((req.Page - 1) * req.PageSize)).Limit(int(req.PageSize))

	if req.OrderBy == nil || *req.OrderBy == "" {
		req.OrderBy = proto.String("id")
	}
	if req.Order == nil || *req.Order == "" {
		req.Order = proto.String("asc")
	}
	db = db.Order(fmt.Sprintf("%s %s", *req.OrderBy, *req.Order))
	if *req.OrderBy != "id" {
		db = db.Order("id asc")
	}

	return list, uint32(total), db.Find(&list).Error
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

func (r File) CreateUserFile(ctx core.Context, uf *entity.UserFile) (uint32, error) {
	return uf.Id, ctx.DB().Create(uf).Error
}

func (r File) UpdateUserFile(ctx core.Context, uf *entity.UserFile) error {
	return ctx.DB().Where("id = ?", uf.Id).Updates(uf).Error
}

func (r File) DeleteUserFile(ctx core.Context, ids []uint32, call func(UserFile *entity.File)) (uint32, error) {
	// 查询删除的文件所属的文件id
	var fileIds []uint32
	if err := ctx.DB().Model(entity.UserFile{}).Select("file_id").
		Where("id in ?", ids).
		Scan(&fileIds).Error; err != nil {
		return 0, err
	}

	// 查询当前的文件被引用的次数
	var results []struct {
		FileId uint32 `json:"file_id"`
		Count  int64  `json:"count"`
	}
	if err := ctx.DB().Model(entity.UserFile{}).Select("file_id", "count(*) count").
		Where("file_id in ?", fileIds).
		Group("file_id").
		Scan(&results).Error; err != nil {
		return 0, err
	}

	// 筛选需要删除的file_id
	var delIds []uint32
	for _, item := range results {
		if item.Count <= 1 {
			delIds = append(delIds, item.FileId)
		}
	}

	// 查询所有被删除的文件
	var files []*entity.File
	if err := ctx.DB().Where("id in ?", delIds).Find(&files).Error; err != nil {
		return 0, err
	}

	err := ctx.Transaction(func(ctx core.Context) error {
		if err := ctx.DB().Where("id in ?", ids).Delete(entity.UserFile{}).Error; err != nil {
			return err
		}
		if err := ctx.DB().Where("id in ?", delIds).Delete(entity.File{}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	for _, item := range files {
		call(item)
	}

	return uint32(len(delIds)), nil
}

func (r File) IsExistUserFile(ctx core.Context, uid, fid uint32) (bool, error) {
	var id uint32
	if err := ctx.DB().Model(entity.UserFile{}).
		Select("id").
		Where("user_id=? and file_id=?", uid, fid).
		Scan(&id).Error; err != nil {
		return false, err
	}
	return id > 0, nil
}

// GetUserFile 获取指定的数据
func (r File) GetUserFile(ctx core.Context, req *types.GetUserFileRequest) (*entity.UserFile, error) {
	file := entity.UserFile{}
	db := ctx.DB().Where("user_id = ?", req.UserId).Where("file_id = ?", req.FileId)
	if req.DirectoryId != 0 {
		db = db.Where("directory_id = ?", req.DirectoryId)
	}
	return &file, db.First(&file).Error
}

func (r File) ListUserFile(ctx core.Context, req *types.ListFileRequest) ([]*entity.UserFile, uint32, error) {
	var (
		list  []*entity.UserFile
		total int64
	)

	db := ctx.DB().Model(entity.UserFile{}).Preload("File")

	if req.DirectoryId != nil {
		db = db.Where("directory_id = ?", *req.DirectoryId)
	}
	if req.Name != nil && *req.Name != "" {
		db = db.Where("name like ", *req.Name+"%")
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	db = db.Offset(int((req.Page - 1) * req.PageSize)).Limit(int(req.PageSize))

	if req.OrderBy == nil || *req.OrderBy == "" {
		req.OrderBy = proto.String("id")
	}
	if req.Order == nil || *req.Order == "" {
		req.Order = proto.String("asc")
	}
	db = db.Order(fmt.Sprintf("%s %s", *req.OrderBy, *req.Order))
	if *req.OrderBy != "id" {
		db = db.Order("id asc")
	}

	return list, uint32(total), db.Find(&list).Error
}
