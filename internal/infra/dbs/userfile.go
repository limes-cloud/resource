package dbs

import (
	"fmt"
	"sync"

	"google.golang.org/protobuf/proto"

	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/types"
)

type UserFile struct{}

var (
	userFileIns  *UserFile
	userFileOnce sync.Once
)

func NewUserFile() *UserFile {
	userFileOnce.Do(func() {
		userFileIns = &UserFile{}
	})
	return userFileIns
}

func (r UserFile) CreateUserFile(ctx core.Context, uf *entity.UserFile) (uint32, error) {
	return uf.Id, ctx.DB().Create(uf).Error
}

func (r UserFile) UpdateUserFile(ctx core.Context, uf *entity.UserFile) error {
	return ctx.DB().Where("id = ?", uf.Id).Updates(uf).Error
}

func (r UserFile) DeleteUserFile(ctx core.Context, ids []uint32, call func(file *entity.File)) (uint32, error) {
	var fileIds []uint32
	if err := ctx.DB().Model(entity.UserFile{}).Select("file_id").
		Where("id in ?", ids).
		Scan(&fileIds).Error; err != nil {
		return 0, err
	}

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

	var delIds []uint32
	for _, item := range results {
		if item.Count <= 1 {
			delIds = append(delIds, item.FileId)
		}
	}

	var files []*entity.File
	if err := ctx.DB().Where("id in ?", delIds).Find(&files).Error; err != nil {
		return 0, err
	}

	err := ctx.Transaction(func(ctx core.Context) error {
		if err := ctx.DB().Where("id in ?", ids).Delete(entity.UserFile{}).Error; err != nil {
			return err
		}
		return ctx.DB().Where("id in ?", delIds).Delete(entity.File{}).Error
	})
	if err != nil {
		return 0, err
	}

	for _, item := range files {
		call(item)
	}
	return uint32(len(delIds)), nil
}

func (r UserFile) IsExistUserFile(ctx core.Context, uid, fid uint32) (bool, error) {
	var id uint32
	if err := ctx.DB().Model(entity.UserFile{}).
		Select("id").
		Where("user_id=? and file_id=?", uid, fid).
		Scan(&id).Error; err != nil {
		return false, err
	}
	return id > 0, nil
}

func (r UserFile) GetUserFile(ctx core.Context, req *types.GetUserFileRequest) (*entity.UserFile, error) {
	uf := entity.UserFile{}
	db := ctx.DB().Where("user_id = ?", req.UserId).Where("file_id = ?", req.FileId)
	if req.DirectoryId != 0 {
		db = db.Where("directory_id = ?", req.DirectoryId)
	}
	return &uf, db.First(&uf).Error
}

func (r UserFile) ListUserFile(ctx core.Context, req *types.ListFileRequest) ([]*entity.UserFile, uint32, error) {
	var (
		list  []*entity.UserFile
		total int64
	)

	db := ctx.DB().Model(entity.UserFile{}).Preload("File")

	if req.DirectoryId != nil {
		db = db.Where("directory_id = ?", *req.DirectoryId)
	}
	if req.Name != nil && *req.Name != "" {
		db = db.Where("name like ?", *req.Name+"%")
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
