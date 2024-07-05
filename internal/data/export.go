package data

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/pkg/filex"
	"github.com/limes-cloud/kratosx/pkg/valx"
	"github.com/limes-cloud/kratosx/pkg/xlsx"
	"google.golang.org/protobuf/proto"

	biz "github.com/limes-cloud/resource/internal/biz/export"
	"github.com/limes-cloud/resource/internal/conf"
	"github.com/limes-cloud/resource/internal/data/model"
	"github.com/limes-cloud/resource/internal/pkg/store"
)

type exportRepo struct {
	store    store.Store
	expStore store.Store
	conf     *conf.Config
}

func NewExportRepo(conf *conf.Config, store store.Store, expStore store.Store) biz.Repo {
	exp := &exportRepo{
		conf:     conf,
		store:    store,
		expStore: expStore,
	}
	go func() {
		for {
			// 清理临时文件
			exp.clearExportTmpCache()
			exp.clearExportFile()
			time.Sleep(10 * time.Minute)
		}
	}()
	return exp
}

// ToExportEntity model转entity
func (r exportRepo) ToExportEntity(m *model.Export) *biz.Export {
	e := &biz.Export{}
	_ = valx.Transform(m, e)
	e.URL = r.GenURL(m.Src)
	return e
}

// ToExportModel entity转model
func (r exportRepo) ToExportModel(e *biz.Export) *model.Export {
	m := &model.Export{}
	_ = valx.Transform(e, m)
	return m
}

func (r exportRepo) CopyExport(ctx kratosx.Context, export *biz.Export, req *biz.CopyExportRequest) (uint32, error) {
	exp := biz.Export{
		UserId:       req.UserId,
		DepartmentId: req.DepartmentId,
		Scene:        req.Scene,
		Name:         req.Name,
		Size:         export.Size,
		Sha:          export.Sha,
		Src:          export.Src,
		Status:       export.Status,
		ExpiredAt:    time.Now().Unix() + int64(r.conf.Export.Expire.Seconds()),
	}
	return r.CreateExport(ctx, &exp)
}

func (r exportRepo) GetKeyByValue(ctx kratosx.Context, value string) (string, error) {
	key := value
	if strings.Contains(value, "/") {
		key = value[strings.Index(value, "/")+1:]
	} else if !strings.Contains(value, ".") {
		if err := ctx.DB().Model(model.File{}).Select("key").Where("sha=?", value).Scan(&key).Error; err != nil {
			return "", err
		}
	}
	return key, nil
}

func (r exportRepo) GetFileByValue(ctx kratosx.Context, value string) (*os.File, error) {
	key, err := r.GetKeyByValue(ctx, value)
	if err != nil {
		return nil, err
	}
	fileName := r.conf.Export.LocalDir + "/tmp/" + key
	if filex.IsExistFile(fileName) {
		return os.Open(fileName)
	}

	reader, err := r.store.Get(key)
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	if _, err = io.Copy(file, reader); err != nil {
		return nil, err
	}
	return file, nil
}

func (r exportRepo) ExportExcel(ctx kratosx.Context, src string, rows [][]*biz.ExportExcelCol) (uint32, error) {
	path := r.conf.Export.LocalDir + "/" + src
	xlsxFile := xlsx.New(path).Writer()
	for _, cols := range rows {
		var temp []any
		for _, item := range cols {
			switch item.Type {
			case "image":
				if item.Value == "" {
					continue
				}
				file, err := r.GetFileByValue(ctx, item.Value)
				if err != nil {
					ctx.Logger().Errorw("msg", "get file error", "err", err.Error())
					continue
				}
				temp = append(temp, file)
			default:
				temp = append(temp, item.Value)
			}
		}
		if err := xlsxFile.WriteRow(temp); err != nil {
			ctx.Logger().Errorw("msg", "write xlsx row error", "err", err.Error())
		}
	}
	if err := xlsxFile.Save(); err != nil {
		return 0, err
	}
	stat, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return uint32(stat.Size() / 1000), nil
}

func (r exportRepo) ExportFile(ctx kratosx.Context, src string, list []*biz.ExportFileItem) (uint32, error) {
	var exports = make(map[string]string)
	for _, item := range list {
		key, err := r.GetKeyByValue(ctx, item.Value)
		if err != nil {
			ctx.Logger().Errorw("msg", "get file key error", "err", err.Error())
			continue
		}
		exports[key] = key
		if item.Rename != "" {
			exports[key] = item.Rename + filepath.Ext(key)
		}
	}

	var oriExports = make(map[string]string)
	for key, rename := range exports {
		path := r.conf.Export.LocalDir + "/tmp/" + key
		if filex.IsExistFile(path) {
			oriExports[path] = rename
			continue
		}

		fd, err := os.Create(path)
		if err != nil {
			ctx.Logger().Errorw("msg", "create file err", "path", path, "err", err.Error())
			continue
		}

		reader, err := r.store.Get(key)
		if err != nil {
			ctx.Logger().Errorw("msg", "get remote file error", "key", key, "err", err.Error())
			continue
		}

		if _, err := io.Copy(fd, reader); err != nil {
			ctx.Logger().Errorw("msg", "save remote file error", "key", key, "download err", err.Error())
			continue
		}

		oriExports[path] = rename
	}

	src = r.conf.Export.LocalDir + "/" + src
	if err := filex.ZipFiles(src, oriExports); err != nil {
		return 0, err
	}

	stat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}
	return uint32(stat.Size() / 1024), nil
}

// ListExport 获取列表
func (r exportRepo) ListExport(ctx kratosx.Context, req *biz.ListExportRequest) ([]*biz.Export, uint32, error) {
	var (
		bs    []*biz.Export
		ms    []*model.Export
		total int64
		fs    = []string{"*"}
	)

	db := ctx.DB().Model(model.Export{}).Select(fs)

	if !req.All {
		if req.UserIds != nil {
			db = db.Where("user_id in ?", req.UserIds)
		}
		if req.DepartmentIds != nil {
			db = db.Where("department_ids in ?", req.DepartmentIds)
		}
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
		bs = append(bs, r.ToExportEntity(m))
	}
	return bs, uint32(total), nil
}

// CreateExport 创建数据
func (r exportRepo) CreateExport(ctx kratosx.Context, req *biz.Export) (uint32, error) {
	m := r.ToExportModel(req)
	return m.Id, ctx.DB().Create(m).Error
}

// DeleteExport 删除数据
func (r exportRepo) DeleteExport(ctx kratosx.Context, ids []uint32) (uint32, error) {
	db := ctx.DB().Where("id in ?", ids).Delete(&model.Export{})
	return uint32(db.RowsAffected), db.Error
}

// GetExportBySha 获取指定数据
func (r exportRepo) GetExportBySha(ctx kratosx.Context, sha string) (*biz.Export, error) {
	var (
		m  = model.Export{}
		fs = []string{"*"}
	)
	db := ctx.DB().Select(fs)
	if err := db.Where("sha = ?", sha).First(&m).Error; err != nil {
		return nil, err
	}

	return r.ToExportEntity(&m), nil
}

// GetExport 获取指定的数据
func (r exportRepo) GetExport(ctx kratosx.Context, id uint32) (*biz.Export, error) {
	var (
		m  = model.Export{}
		fs = []string{"*"}
	)
	db := ctx.DB().Select(fs)
	if err := db.First(&m, id).Error; err != nil {
		return nil, err
	}

	return r.ToExportEntity(&m), nil
}

// UpdateExport 更新数据
func (r exportRepo) UpdateExport(ctx kratosx.Context, req *biz.Export) error {
	return ctx.DB().Updates(r.ToExportModel(req)).Error
}

// clearExportTmpCache 清理临时文件夹
func (r exportRepo) clearExportTmpCache() {
	dir := r.conf.Export.LocalDir + "/tmp"
	if !filex.IsExistFolder(dir) {
		return
	}
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if path == dir {
			return nil
		}
		if err != nil {
			return err
		}
		d := time.Since(info.ModTime())
		if d.Seconds() >= r.conf.Export.Expire.Seconds() {
			_ = os.RemoveAll(path)
		}
		return err
	})
}

// clearExportFile 清理导出的过期的大文件
func (r exportRepo) clearExportFile() {
	dir := r.conf.Export.LocalDir
	if !filex.IsExistFolder(dir) {
		return
	}

	ctx := kratosx.MustContext(context.Background())

	// 获取已经超时的文件
	var files []*model.Export
	if err := ctx.DB().Where("expired_at <= ?", time.Now().Unix()).Find(&files).Error; err != nil {
		ctx.Logger().Warnw("msg", "get expire export file error", "err", err.Error())
		return
	}

	for _, item := range files {
		var count int64
		if err := ctx.DB().Model(model.File{}).Where("sha=?", item.Sha).Count(&count).Error; err != nil {
			ctx.Logger().Warnw("msg", "get expire export file count error", "err", err.Error())
			return
		}

		if err := ctx.DB().Model(model.Export{}).
			Where("id=?", item.Id).
			UpdateColumn("status", biz.STATUS_EXPIRED).Error; err != nil {
			ctx.Logger().Warnw("msg", "update expire export file status error", "err", err.Error())
			return
		}

		if count == 1 {
			_ = os.RemoveAll(dir + "/" + item.Sha)
		}
	}
}

func (r exportRepo) GetExportFileKeyById(ctx kratosx.Context, id uint32) (string, error) {
	var key string
	if err := ctx.DB().Model(model.File{}).Select("key").Where("id=?", id).Scan(&key).Error; err != nil {
		return "", err
	}
	return key, nil
}

func (r exportRepo) VerifyURL(key string, expire string, sign string) error {
	return r.expStore.VerifyTemporaryURL(key, expire, sign)
}

func (r exportRepo) GenURL(key string) string {
	url, err := r.expStore.GenTemporaryURL(key)
	if err != nil {
		return ""
	}
	return url
}
