package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/limes-cloud/resource/internal/infra/store"
	storetypes "github.com/limes-cloud/resource/internal/infra/store/types"

	"github.com/google/uuid"
	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/model"
	"github.com/limes-cloud/resource/api/errors"
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/pkg/filex"

	"github.com/limes-cloud/kratosx/library/db/gormtranserror"
	"github.com/limes-cloud/kratosx/pkg/crypto"
	"github.com/xuri/excelize/v2"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"

	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/repository"
	"github.com/limes-cloud/resource/internal/types"
)

const (
	EXPORT_STATUS_FAIL = "FAIL"

	// EXPORT_STATUS_PROGRESS  = "PROGRESS"
	EXPORT_STATUS_COMPLETED = "COMPLETED"
	EXPORT_STATUS_EXPIRED   = "EXPIRED"
)

type Export struct {
	repo repository.Export
	file repository.File
}

func NewExport(
	repo repository.Export,
	file repository.File,
) *Export {
	export := &Export{
		repo: repo,
		file: file,
	}
	go func() {
		ctx := core.MustContext(context.Background(), kratosx.WithSkipDBHook())
		for {
			// 清理临时文件
			export.clearExportFile(ctx)
			export.clearExportTmpCache()
			time.Sleep(10 * time.Minute)
		}
	}()
	return export
}

// ListExport 获取导出信息列表
func (u *Export) ListExport(ctx core.Context, req *types.ListExportRequest) ([]*entity.Export, uint32, error) {
	list, total, err := u.repo.ListExport(ctx, req)
	if err != nil {
		ctx.Logger().Warnw("msg", "list directory error", "err", err.Error())
		return nil, 0, errors.ListError(err.Error())
	}
	return list, total, nil
}

// ExportExcel 创建导出表格
func (u *Export) ExportExcel(ctx core.Context, req *types.ExportExcelRequest) (*types.ExportExcelReply, error) {
	b, _ := json.Marshal(req.Rows)
	sha := crypto.MD5(b)
	export, err := u.repo.GetExportBySha(ctx, sha)
	if err != nil && !gormtranserror.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.SystemError(err.Error())
	}

	if err == nil {
		//
		if export.Status == STATUS_PROGRESS && export.UserId == ctx.Auth().UserId {
			return nil, errors.ExportTaskProcessError()
		}
		// 复制正在进行中的导入数据
		id, err := u.repo.CopyExport(ctx, export, &types.CopyExportRequest{
			Name: req.Name,
		})
		if err != nil {
			return nil, err
		}
		return &types.ExportExcelReply{Id: id, Sha: sha}, nil
	}

	id, err := u.repo.CreateExport(ctx, &entity.Export{
		Name:   req.Name,
		Sha:    sha,
		Key:    fmt.Sprintf("%s.zip", sha),
		Status: STATUS_PROGRESS,
	})
	if err != nil {
		ctx.Logger().Warnw("msg", "create export error", "err", err.Error())
		return nil, errors.DatabaseError(err.Error())
	}

	go func() {
		kCtx := ctx.Clone()
		conf := kCtx.Config()
		size, err := u.exportExcel(kCtx, sha, req)
		exp := &entity.Export{
			BaseTenantUserModel: model.BaseTenantUserModel{Id: id},
			Status:              STATUS_COMPLETED,
			Size:                size,
			ExpiredAt:           time.Now().Unix() + int64(conf.Export.Expire.Seconds()),
		}
		if err != nil {
			exp.Status = EXPORT_STATUS_FAIL
			exp.Reason = proto.String(err.Error())
		}

		// todo 上传到存储系统

		if err := u.repo.UpdateExport(kCtx, exp); err != nil {
			ctx.Logger().Errorw("msg", "update export status error", "err", err.Error())
		}
	}()

	return &types.ExportExcelReply{Id: id, Sha: sha, Key: fmt.Sprintf("%s.zip", sha)}, nil
}

func (u *Export) getStoreByKey(key string) (storetypes.Store, error) {
	arr := strings.Split(key, "/")
	if len(arr) == 2 {
		return store.NewStore(arr[0])
	}
	return store.NewStore()
}

// nolint
func (u *Export) getFileByValue(ctx core.Context, key string) (*os.File, error) {
	conf := ctx.Config()
	fileName := conf.Export.LocalDir + "/tmp/" + key
	if filex.IsExistFile(fileName) {
		return os.Open(fileName)
	}

	store, err := u.getStoreByKey(key)
	if err != nil {
		return nil, err
	}

	reader, err := store.Get(key)
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}
	if _, err = io.Copy(file, reader); err != nil {
		return nil, err
	}
	return file, nil
}

func (u *Export) exportExcel(ctx core.Context, sha string, req *types.ExportExcelRequest) (uint32, error) {
	conf := ctx.Config()

	uid := uuid.New().String()

	key := fmt.Sprintf("%s.zip", sha)
	// 存储地址
	path := conf.Export.LocalDir + "/tmp/" + key

	// 表格保存地址
	excelPath := conf.Export.LocalDir + "/tmp/" + uid + "/" + fmt.Sprintf("%s.xlsx", sha)
	defer func() {
		// 移除生成的excel文件
		_ = os.Remove(excelPath)
	}()

	// 创建excel文件
	file := excelize.NewFile()
	writer, err := file.NewStreamWriter("sheet1")
	if err != nil {
		return 0, err
	}

	// headers 转 row
	transHeader := func(rows []string) []any {
		var res []any
		for _, item := range rows {
			res = append(res, item)
		}
		return res
	}

	// 写入标题
	if err := writer.SetRow("A1", transHeader(req.Headers)); err != nil {
		return 0, err
	}

	// 写入行数据
	for ind, item := range req.Rows {
		var rows []any
		for _, col := range item {
			rows = append(rows, col.Value)
		}
		if err := writer.SetRow(fmt.Sprintf("A%d", ind+2), rows); err != nil {
			return 0, err
		}
	}

	// 保存数据
	if err := writer.Flush(); err != nil {
		return 0, err
	}

	// 存储到磁盘
	if err := file.SaveAs(excelPath); err != nil {
		return 0, err
	}

	// 打包文件
	exports, err := u.fetchFile(ctx, uid, req.Files)
	if err != nil {
		return 0, err
	}

	// 挂载xlsx
	exports[excelPath] = req.Name + ".xlsx"

	// 打包文件
	if err := filex.ZipFiles(path, exports); err != nil {
		return 0, err
	}

	// 上传到存储系统
	export, err := store.NewExportStore()
	if err != nil {
		return 0, err
	}
	if err := export.PutFromLocal(key, path); err != nil {
		return 0, err
	}

	// 删除本地文件
	_ = os.Remove(path)
	_ = os.Remove(conf.Export.LocalDir + "/tmp/" + uid)

	// 获取文件大小
	stat, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return uint32(stat.Size() / 1000), nil
}

// fetchFile 拉取文件, 返回文件路径和文件名
func (u *Export) fetchFile(ctx core.Context, uid string, list []*types.ExportFileItem) (map[string]string, error) {
	exports := make(map[string]string)
	for _, item := range list {
		exports[item.Value] = item.Value
		if item.Rename != "" {
			exports[item.Value] = item.Rename + filepath.Ext(item.Value)
		}
	}

	var (
		oriExports = make(map[string]string)
		conf       = ctx.Config()
		dir        = conf.Export.LocalDir + "/tmp/" + uid + "/"
	)
	if !filex.IsExistFolder(dir) {
		_ = os.MkdirAll(dir, 0o750)
	}
	for key, rename := range exports {
		path := dir + key
		if filex.IsExistFile(path) {
			oriExports[path] = rename
			continue
		}

		fd, err := os.Create(path)
		if err != nil {
			ctx.Logger().Errorw("msg", "create file err", "path", path, "err", err.Error())
			continue
		}

		store, err := u.getStoreByKey(key)
		if err != nil {
			return nil, err
		}

		reader, err := store.Get(key)
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

	return oriExports, nil
}

func (u *Export) exportFile(ctx core.Context, key string, list []*types.ExportFileItem) (uint32, error) {
	uid := uuid.New().String()
	oriExports, err := u.fetchFile(ctx, uid, list)
	if err != nil {
		return 0, err
	}

	conf := ctx.Config()
	path := conf.Export.LocalDir + "/tmp/" + key
	if err := filex.ZipFiles(path, oriExports); err != nil {
		return 0, err
	}

	stat, err := os.Stat(path)
	if err != nil {
		return 0, err
	}

	// 上传到存储系统
	export, err := store.NewExportStore()
	if err != nil {
		return 0, err
	}

	if err := export.PutFromLocal(key, path); err != nil {
		return 0, err
	}

	// 删除本地文件
	_ = os.Remove(path)
	_ = os.RemoveAll(conf.Export.LocalDir + "/tmp/" + uid)

	return uint32(stat.Size() / 1024), nil
}

// clearExportTmpCache 清理临时文件夹
func (u *Export) clearExportTmpCache() {
	conf := core.MustContext(context.Background()).Config()
	dir := conf.Export.LocalDir + "/tmp"
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
		if d.Seconds() >= conf.Export.Expire.Seconds() {
			_ = os.RemoveAll(path)
		}
		return err
	})
}

// clearExportFile 清理导出的过期的大文件
func (u *Export) clearExportFile(ctx core.Context) {
	conf := ctx.Config()
	dir := conf.Export.LocalDir
	if !filex.IsExistFolder(dir) {
		return
	}

	var (
		page     uint32 = 1
		pageSize uint32 = 100
	)

	for {
		// 获取已经超时的文件
		files, _, err := u.repo.ListExport(ctx, &types.ListExportRequest{
			Page:      page,
			PageSize:  pageSize,
			Status:    proto.String(EXPORT_STATUS_COMPLETED),
			ExpiredAt: proto.Int64(time.Now().Unix()),
		})
		if err != nil {
			ctx.Logger().Warnw("msg", "get expire export file error", "err", err.Error())
			return
		}

		for _, item := range files {
			if err := u.repo.UpdateExport(ctx, &entity.Export{
				BaseTenantUserModel: model.BaseTenantUserModel{Id: item.Id},
				Status:              EXPORT_STATUS_EXPIRED,
			}); err != nil {
				ctx.Logger().Warnw("msg", "update expire export file status error", "err", err.Error())
				return
			}

			count, err := u.repo.GetExportFileCount(ctx, &types.GetExportFileCountRequest{
				Sha:    item.Sha,
				Status: EXPORT_STATUS_COMPLETED,
			})
			if err != nil {
				ctx.Logger().Warnw("msg", "get export file count error", "err", err.Error())
			}
			if count == 0 {
				if err = os.RemoveAll(dir + "/" + item.Key); err != nil {
					ctx.Logger().Warnw("msg", "remove export file status error", "err", err.Error())
				}

				store, err := u.getStoreByKey(item.Key)
				if err != nil {
					continue
				}

				if err = store.Delete(item.Key); err != nil {
					ctx.Logger().Warnw("msg", "delete export file status error", "err", err.Error())
				}
			}
		}

		// 判断是否还有数据
		if uint32(len(files)) < pageSize {
			break
		}
		page++
	}
}

// ExportFile 创建导出表格
func (u *Export) ExportFile(ctx core.Context, req *types.ExportFileRequest) (*types.ExportFileReply, error) {
	b, _ := json.Marshal(req.Files)
	ids, _ := json.Marshal(req.Ids)
	sha := crypto.MD5(append(b, ids...))
	export, err := u.repo.GetExportBySha(ctx, sha)
	if err != nil && !gormtranserror.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err == nil {
		if export.Status == STATUS_PROGRESS && export.UserId == ctx.Auth().UserId {
			return nil, errors.ExportTaskProcessError()
		}
		// 复制正在进行中的导入数据
		id, err := u.repo.CopyExport(ctx, export, &types.CopyExportRequest{
			Name: req.Name,
		})
		if err != nil {
			return nil, err
		}
		return &types.ExportFileReply{Id: id}, nil
	}

	if len(req.Ids) != 0 {
		for _, id := range req.Ids {
			file, err := u.file.GetFile(ctx, id)
			if err != nil {
				return nil, errors.DatabaseError(err.Error())
			}
			req.Files = append(req.Files, &types.ExportFileItem{Value: file.Key})
		}
	}

	key := fmt.Sprintf("%s.zip", sha)
	id, err := u.repo.CreateExport(ctx, &entity.Export{
		Name:   req.Name,
		Sha:    sha,
		Key:    fmt.Sprintf("%s.zip", sha),
		Status: STATUS_PROGRESS,
	})
	if err != nil {
		return nil, errors.DatabaseError(err.Error())
	}

	go func() {
		kCtx := ctx.Clone()
		conf := kCtx.Config()
		size, err := u.exportFile(kCtx, key, req.Files)
		exp := &entity.Export{
			BaseTenantUserModel: model.BaseTenantUserModel{Id: id},
			Status:              STATUS_COMPLETED,
			Size:                size,
			ExpiredAt:           time.Now().Unix() + int64(conf.Export.Expire.Seconds()),
		}
		if err != nil {
			exp.Status = EXPORT_STATUS_FAIL
			exp.Reason = proto.String(err.Error())
		}
		if err := u.repo.UpdateExport(kCtx, exp); err != nil {
			ctx.Logger().Errorw("msg", "update export status error", "err", err.Error())
		}
	}()

	return &types.ExportFileReply{Id: id}, nil
}

// DeleteExport 删除导出信息
func (u *Export) DeleteExport(ctx core.Context, ids []uint32) (uint32, error) {
	total, err := u.repo.DeleteExport(ctx, ids)
	if err != nil {
		return 0, errors.DeleteError(err.Error())
	}
	return total, nil
}

// GetExport 获取指定的导出信息
func (u *Export) GetExport(ctx core.Context, req *types.GetExportRequest) (*entity.Export, error) {
	var (
		res *entity.Export
		err error
	)

	if req.Id != nil {
		res, err = u.repo.GetExport(ctx, *req.Id)
	} else if req.Sha != nil {
		res, err = u.repo.GetExportBySha(ctx, *req.Sha)
	} else {
		return nil, errors.ParamsError()
	}
	if err != nil {
		return nil, errors.GetError(err.Error())
	}
	return res, nil
}

// VerifyURL 验证url
func (u *Export) VerifyURL(key, expire, sign string) error {
	store, err := u.getStoreByKey(key)
	if err != nil {
		return err
	}
	return store.VerifyTemporaryURL(key, expire, sign)
}

func (s *Export) LocalPath(next http.Handler, key string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = key
		next.ServeHTTP(w, r)
	})
}
