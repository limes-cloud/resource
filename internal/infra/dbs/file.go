package dbs

import (
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/limes-cloud/kratosx"
	"google.golang.org/protobuf/proto"

	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/types"
)

type File struct {
}

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
func (r File) GetFileBySha(ctx kratosx.Context, sha string) (*entity.File, error) {
	var (
		file = entity.File{}
		fs   = []string{"*"}
	)
	return &file, ctx.DB().Select(fs).Where("sha = ?", sha).First(&file).Error
}

// GetFileBySrc 获取指定数据
func (r File) GetFileBySrc(ctx kratosx.Context, src string) (*entity.File, error) {
	var (
		file = entity.File{}
		fs   = []string{"*"}
	)
	return &file, ctx.DB().Select(fs).Where("src = ?", src).First(&file).Error
}

func (r File) GetFileByUploadId(ctx kratosx.Context, uid string) (*entity.File, error) {
	var (
		file = entity.File{}
		fs   = []string{"*"}
	)
	return &file, ctx.DB().Select(fs).Where("upload_id = ?", uid).First(&file).Error
}

// GetFile 获取指定的数据
func (r File) GetFile(ctx kratosx.Context, id uint32) (*entity.File, error) {
	var (
		file = entity.File{}
		fs   = []string{"*"}
	)
	return &file, ctx.DB().Select(fs).First(&file, id).Error
}

// ListFile 获取列表
func (r File) ListFile(ctx kratosx.Context, req *types.ListFileRequest) ([]*entity.File, uint32, error) {
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
	if len(req.ShaList) != 0 {
		db = db.Where("sha in ?", req.ShaList)
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
func (r File) CreateFile(ctx kratosx.Context, file *entity.File) (uint32, error) {
	file.Src = fmt.Sprintf("%d/%s", file.DirectoryId, file.Key)
	return file.Id, ctx.DB().Create(file).Error
}

// UpdateFile 更新数据
func (r File) UpdateFile(ctx kratosx.Context, file *entity.File) error {
	return ctx.DB().Where("id = ?", file.Id).Updates(file).Error
}

// DeleteFile 删除数据
func (r File) DeleteFile(ctx kratosx.Context, ids []uint32, call func(file *entity.File)) (uint32, error) {
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

func (r File) CopyFile(ctx kratosx.Context, src *entity.File, directoryId uint32, fileName string) error {
	if src.DirectoryId == directoryId {
		return nil
	}
	uids := strings.Split(uuid.NewString(), "-")
	file := entity.File{
		DirectoryId: directoryId,
		Key:         src.Key,
		Src:         fmt.Sprintf("%d/%s", directoryId, src.Key),
		Name:        fileName,
		Status:      src.Status,
		UploadId:    src.UploadId + "_copy_" + uids[0],
		Type:        src.Type,
		Size:        src.Size,
		ChunkCount:  src.ChunkCount,
		Sha:         src.Sha,
	}

	if err := ctx.DB().Create(&file).Error; err != nil {
		ctx.Logger().Warnw("msg", "copy file error", "err", err)
	}
	return nil
}
