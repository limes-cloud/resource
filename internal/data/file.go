package data

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/pkg/valx"
	"google.golang.org/protobuf/proto"

	biz "github.com/limes-cloud/resource/internal/biz/file"
	"github.com/limes-cloud/resource/internal/conf"
	"github.com/limes-cloud/resource/internal/data/model"
	"github.com/limes-cloud/resource/internal/pkg/store"
)

type fileRepo struct {
	conf  *conf.Config
	store store.Store
}

func NewFileRepo(conf *conf.Config, st store.Store) biz.Repo {
	return &fileRepo{
		conf:  conf,
		store: st,
	}
}

// ToFileEntity model转entity
func (r fileRepo) ToFileEntity(ctx kratosx.Context, m *model.File) *biz.File {
	e := &biz.File{
		Id:          m.Id,
		DirectoryId: m.DirectoryId,
		Name:        m.Name,
		Type:        m.Type,
		Size:        m.Size,
		Sha:         m.Sha,
		Src:         m.Src,
		Key:         m.Key,
		Status:      m.Status,
		UploadId:    m.UploadId,
		ChunkCount:  m.ChunkCount,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
	accessURL, err := r.store.GenTemporaryURL(m.Key)
	if err != nil {
		ctx.Logger().Warnf("gen template url error:%s", err.Error())
	} else {
		e.URL = accessURL
	}
	return e
}

// ToFileModel entity转model
func (r fileRepo) ToFileModel(e *biz.File) *model.File {
	m := &model.File{}
	_ = valx.Transform(e, m)
	return m
}

// GetFileBySha 获取指定数据
func (r fileRepo) GetFileBySha(ctx kratosx.Context, sha string) (*biz.File, error) {
	var (
		m  = model.File{}
		fs = []string{"*"}
	)
	db := ctx.DB().Select(fs)
	if err := db.Where("sha = ?", sha).First(&m).Error; err != nil {
		return nil, err
	}

	return r.ToFileEntity(ctx, &m), nil
}

// GetFileBySrc 获取指定数据
func (r fileRepo) GetFileBySrc(ctx kratosx.Context, src string) (*biz.File, error) {
	var (
		m  = model.File{}
		fs = []string{"*"}
	)
	db := ctx.DB().Select(fs)
	if err := db.Where("src = ?", src).First(&m).Error; err != nil {
		return nil, err
	}

	return r.ToFileEntity(ctx, &m), nil
}

// GetFile 获取指定的数据
func (r fileRepo) GetFile(ctx kratosx.Context, id uint32) (*biz.File, error) {
	var (
		m  = model.File{}
		fs = []string{"*"}
	)
	db := ctx.DB().Select(fs)
	if err := db.First(&m, id).Error; err != nil {
		return nil, err
	}

	return r.ToFileEntity(ctx, &m), nil
}

// ListFile 获取列表
func (r fileRepo) ListFile(ctx kratosx.Context, req *biz.ListFileRequest) ([]*biz.File, uint32, error) {
	var (
		bs    []*biz.File
		ms    []*model.File
		total int64
		fs    = []string{"*"}
	)

	db := ctx.DB().Model(model.File{}).Select(fs)

	if req.DirectoryId != nil {
		db = db.Where("directory_id = ?", *req.DirectoryId)
	}
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
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

	if err := db.Find(&ms).Error; err != nil {
		return nil, 0, err
	}

	for _, m := range ms {
		bs = append(bs, r.ToFileEntity(ctx, m))
	}
	return bs, uint32(total), nil
}

// CreateFile 创建数据
func (r fileRepo) CreateFile(ctx kratosx.Context, req *biz.File) (uint32, error) {
	m := r.ToFileModel(req)
	m.Src = fmt.Sprintf("%d/%s", m.DirectoryId, m.Key)
	return m.Id, ctx.DB().Create(m).Error
}

// UpdateFile 更新数据
func (r fileRepo) UpdateFile(ctx kratosx.Context, req *biz.File) error {
	m := r.ToFileModel(req)
	m.Src = fmt.Sprintf("%d/%s", m.DirectoryId, m.Key)
	return ctx.DB().Updates(m).Error
}

// DeleteFile 删除数据
func (r fileRepo) DeleteFile(ctx kratosx.Context, ids []uint32) (uint32, error) {
	var files []*model.File
	if err := ctx.DB().Where("id in ?", ids).Find(&files).Error; err != nil {
		return 0, err
	}

	for _, item := range files {
		if item.Status == biz.STATUS_COMPLETED {
			_ = r.store.Delete(item.Key)
		} else {
			chunk, err := r.store.NewPutChunkByUploadID(item.Key, item.UploadId)
			if err == nil {
				_ = chunk.Abort()
			}
		}
	}

	db := ctx.DB().Where("id in ?", ids).Delete(model.File{})
	return uint32(db.RowsAffected), db.Error
}

func (r fileRepo) CopyFile(ctx kratosx.Context, src *biz.File, directoryId uint32, fileName string) error {
	if src.DirectoryId == directoryId {
		return nil
	}
	uids := strings.Split(uuid.NewString(), "-")
	file := model.File{
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

func (r fileRepo) UpdateFileStatus(ctx kratosx.Context, id uint32, status string) error {
	return ctx.DB().Model(model.File{}).Where("id=?", id).Update("status", status).Error
}

func (r fileRepo) GetFileByUploadId(ctx kratosx.Context, uid string) (*biz.File, error) {
	var m model.File
	if err := ctx.DB().Where("upload_id = ?", uid).First(&m).Error; err != nil {
		return nil, err
	}

	return r.ToFileEntity(ctx, &m), nil
}

func (r fileRepo) GetDirectoryLimitByPath(ctx kratosx.Context, paths []string) (*biz.DirectoryLimit, error) {
	var (
		dir    = model.Directory{}
		parent = uint32(0)
	)

	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		nr := model.Directory{}
		if err := ctx.DB().Where(model.Directory{
			ParentId: parent,
			Name:     path,
		}).Attrs(model.Directory{
			Accept:  strings.Join(r.conf.DefaultAcceptTypes, ","),
			MaxSize: r.conf.DefaultMaxSize,
		}).FirstOrCreate(&nr).Error; err != nil {
			return nil, err
		}
		parent = nr.Id
		dir = nr
	}

	return &biz.DirectoryLimit{
		DirectoryId: dir.Id,
		Accepts:     strings.Split(dir.Accept, ","),
		MaxSize:     dir.MaxSize,
	}, nil
}

func (r fileRepo) GetDirectoryLimitById(ctx kratosx.Context, id uint32) (*biz.DirectoryLimit, error) {
	var dir = model.Directory{}
	if err := ctx.DB().Where("id=?", id).First(&dir).Error; err != nil {
		return nil, err
	}

	return &biz.DirectoryLimit{
		DirectoryId: dir.Id,
		Accepts:     strings.Split(dir.Accept, ","),
		MaxSize:     dir.MaxSize,
	}, nil
}

func (r fileRepo) GetStore() store.Store {
	return r.store
}
