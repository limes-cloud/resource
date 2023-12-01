package model

import (
	"github.com/limes-cloud/kratos"
)

type File struct {
	BaseModel
	DirectoryID uint32 `json:"directory_id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Size        uint32 `json:"size"`
	Sha         string `json:"sha"`
	Src         string `json:"src"`
	UploadID    string `json:"upload_id"`
	ChunkCount  uint32 `json:"chunk_count"`
	Storage     string `json:"storage"`
	Status      string `json:"status"`
	//SrcFormat   string `json:"-" gorm:"-"`
}

func (f *File) Copy(ctx kratos.Context, dir uint32, key string) error {
	if f.DirectoryID == dir {
		return nil
	}

	nf := *f
	nf.ID = 0
	nf.DirectoryID = dir
	nf.Name = key
	nf.CreatedAt = 0
	nf.UploadID = ""

	return ctx.DB().Create(&nf).Error
}

// Create 创建文件信息
func (f *File) Create(ctx kratos.Context) error {
	return ctx.DB().Model(f).Create(f).Error
}

// OneBySha 通过sha查询文件信息
func (f *File) OneBySha(ctx kratos.Context, sha string) error {
	return ctx.DB().First(f, "sha=?", sha).Error
}

// OneByUploadID 通过upload_id查询文件信息
func (f *File) OneByUploadID(ctx kratos.Context, ui string) error {
	return ctx.DB().First(f, "upload_id=?", ui).Error
}

// Page 查询分页数据
func (f *File) Page(ctx kratos.Context, options *PageOptions) ([]*File, uint32, error) {
	var list []*File
	total := int64(0)

	db := ctx.DB().Model(f)
	if options.Scopes != nil {
		db = db.Scopes(options.Scopes)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, uint32(total), err
	}

	db = db.Offset(int((options.Page - 1) * options.PageSize)).Limit(int(options.PageSize))

	return list, uint32(total), db.Find(&list).Error
}

// AllByDirectoryID 通过id查询文件信息
func (f *File) AllByDirectoryID(ctx kratos.Context, id uint32) ([]*File, error) {
	var list []*File
	return list, ctx.DB().Model(f).Find(&list, "directory_id=?", id).Error
}

// CountByName 通过name查询文件数量
func (f *File) CountByName(ctx kratos.Context, name string) (int64, error) {
	count := int64(0)
	return count, ctx.DB().Model(f).Where("name like ?", name+"%").Count(&count).Error
}

// CountByDirectoryID 通过id查询文件数量
func (f *File) CountByDirectoryID(ctx kratos.Context, id uint32) (int64, error) {
	count := int64(0)
	return count, ctx.DB().Model(f).Where("directory_id=?", id).Count(&count).Error
}

// OneByID 通过id查询文件信息
func (f *File) OneByID(ctx kratos.Context, id uint32) error {
	return ctx.DB().Model(f).First(f, id).Error
}

// OneByDirAndName 通过id查询文件信息
func (f *File) OneByDirAndName(ctx kratos.Context, id uint32, name string) error {
	return ctx.DB().Model(f).Where("directory_id=? and name=?", id, name).First(f).Error
}

// Update 更新文件信息
func (f *File) Update(ctx kratos.Context) error {
	return ctx.DB().Model(f).Updates(&f).Error
}

// DeleteByDirAndIds 通过id数组删除文件信息
func (f *File) DeleteByDirAndIds(ctx kratos.Context, did uint32, ids []uint32) error {
	return ctx.DB().Where("directory_id=? and id in ?", did, ids).Delete(File{}).Error
}
