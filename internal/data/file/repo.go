package file

import (
	"github.com/limes-cloud/kratosx"

	"github.com/limes-cloud/resource/internal/biz/file"
	"github.com/limes-cloud/resource/internal/consts"
)

type repo struct {
}

func NewRepo() file.Repo {
	return &repo{}
}

func (r repo) AddDirectory(ctx kratosx.Context, in *file.Directory) (uint32, error) {
	return in.ID, ctx.DB().Create(in).Error
}

func (r repo) GetDirectoryByID(ctx kratosx.Context, id uint32) (*file.Directory, error) {
	dir := file.Directory{}
	return &dir, ctx.DB().First(&dir, "id=?", id).Error
}

func (r repo) GetDirectoryByName(ctx kratosx.Context, id uint32, name string) (*file.Directory, error) {
	dir := file.Directory{}
	return &dir, ctx.DB().First(&dir, "parent_id=? and name=?", id, name).Error
}

func (r repo) GetDirectoryByPaths(ctx kratosx.Context, app string, paths []string) (*file.Directory, error) {
	dir := file.Directory{}
	parent := uint32(0)
	for _, path := range paths {
		nr := file.Directory{}
		if err := ctx.DB().Where(file.Directory{
			App:      app,
			ParentID: parent,
			Name:     path,
		}).FirstOrCreate(&nr).Error; err != nil {
			return nil, err
		}
		parent = nr.ID
		dir = nr
	}
	return &dir, nil
}

func (r repo) UpdateDirectory(ctx kratosx.Context, in *file.Directory) error {
	return ctx.DB().Model(file.Directory{}).Updates(in).Error
}

func (r repo) DeleteDirectory(ctx kratosx.Context, id uint32) error {
	return ctx.DB().Where("id=?", id).Delete(file.Directory{}).Error
}

func (r repo) AllDirectoryByParentID(ctx kratosx.Context, pid uint32, app string) ([]*file.Directory, error) {
	var list []*file.Directory
	return list, ctx.DB().Model(file.Directory{}).Find(&list, "parent_id=? and app=?", pid, app).Error
}

func (r repo) DirectoryCountByParentID(ctx kratosx.Context, id uint32) (int64, error) {
	var count int64
	return count, ctx.DB().Model(file.Directory{}).Where("parent_id=?", id).Count(&count).Error
}

func (r repo) CopyFile(ctx kratosx.Context, src *file.File, did uint32, name string) error {
	if src.DirectoryID == did {
		return nil
	}

	nf := *src
	nf.ID = 0
	nf.DirectoryID = did
	nf.Name = name
	nf.CreatedAt = 0
	nf.UpdatedAt = 0
	nf.UploadID = nil

	return ctx.DB().Create(&nf).Error
}

func (r repo) FileCountByName(ctx kratosx.Context, did uint32, name string) (int64, error) {
	count := int64(0)
	return count, ctx.DB().Model(file.File{}).Where("directory_id=? and name like ?", did, name+"%").Count(&count).Error
}

func (r repo) FileCountByDirectoryID(ctx kratosx.Context, id uint32) (int64, error) {
	count := int64(0)
	return count, ctx.DB().Model(file.File{}).Where("directory_id=? ", id).Count(&count).Error
}

func (r repo) GetFileByID(ctx kratosx.Context, id uint32) (*file.File, error) {
	fe := file.File{}
	return &fe, ctx.DB().First(&fe, "id=?", id).Error
}

func (r repo) GetFileBySha(ctx kratosx.Context, sha string) (*file.File, error) {
	fe := file.File{}
	return &fe, ctx.DB().First(&fe, "sha=?", sha).Error
}

func (r repo) GetFileByUploadID(ctx kratosx.Context, uid string) (*file.File, error) {
	fe := file.File{}
	return &fe, ctx.DB().First(&fe, "upload_id=?", uid).Error
}

func (r repo) PageFile(ctx kratosx.Context, in *file.PageFileRequest) ([]*file.File, uint32, error) {
	var list []*file.File
	total := int64(0)

	db := ctx.DB().Model(file.File{})
	if in.Name != "" {
		db = db.Where("name=?", in.Name)
	}
	if in.DirectoryId != 0 {
		db = db.Where("directory_id=?", in.DirectoryId)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, uint32(total), err
	}

	db = db.Offset(int((in.Page - 1) * in.PageSize)).Limit(int(in.PageSize))

	return list, uint32(total), db.Find(&list).Error
}

func (r repo) AddFile(ctx kratosx.Context, file *file.File) error {
	return ctx.DB().Create(file).Error
}

func (r repo) UpdateFile(ctx kratosx.Context, file *file.File) error {
	return ctx.DB().Where("id=?", file.ID).Updates(file).Error
}

func (r repo) UpdateFileSuccess(ctx kratosx.Context, id uint32) error {
	return ctx.DB().Model(file.File{}).Where("id=?", id).UpdateColumn("status", consts.STATUS_COMPLETED).Error
}

func (r repo) DeleteFile(ctx kratosx.Context, id uint32) error {
	return ctx.DB().Where("id=?", id).Delete(file.File{}).Error
}

func (r repo) DeleteFiles(ctx kratosx.Context, did uint32, ids []uint32) error {
	return ctx.DB().Where("directory_id=? and id in ?", did, ids).Delete(file.File{}).Error
}
