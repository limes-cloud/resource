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

	thttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/library/db/gormtranserror"
	"github.com/limes-cloud/kratosx/pkg/crypto"
	"github.com/limes-cloud/kratosx/pkg/filex"
	ktypes "github.com/limes-cloud/kratosx/types"
	"github.com/xuri/excelize/v2"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"

	"github.com/limes-cloud/resource/api/resource/errors"
	pb "github.com/limes-cloud/resource/api/resource/file/v1"
	"github.com/limes-cloud/resource/internal/conf"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/repository"
	"github.com/limes-cloud/resource/internal/pkg"
	"github.com/limes-cloud/resource/internal/types"
)

const (
	EXPORT_STATUS_FAIL = "FAIL"

	// EXPORT_STATUS_PROGRESS  = "PROGRESS"
	EXPORT_STATUS_COMPLETED = "COMPLETED"
	EXPORT_STATUS_EXPIRED   = "EXPIRED"
)

type Export struct {
	conf  *conf.Config
	repo  repository.Export
	file  repository.File
	store repository.Store
}

func NewExport(
	conf *conf.Config,
	repo repository.Export,
	file repository.File,
	store repository.Store,
) *Export {
	export := &Export{
		conf:  conf,
		repo:  repo,
		file:  file,
		store: store,
	}
	go func() {
		ctx := kratosx.MustContext(context.Background())
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
func (u *Export) ListExport(ctx kratosx.Context, req *types.ListExportRequest) ([]*entity.Export, uint32, error) {
	list, total, err := u.repo.ListExport(ctx, req)
	if err != nil {
		ctx.Logger().Warnw("msg", "list directory error", "err", err.Error())
		return nil, 0, errors.ListError(err.Error())
	}
	for ind, item := range list {
		url, err := u.store.GenTemporaryURL(item.Src)
		if err != nil {
			continue
		}
		list[ind].Url = url
	}
	return list, total, nil
}

// ExportExcel 创建导出表格
func (u *Export) ExportExcel(ctx kratosx.Context, req *types.ExportExcelRequest) (*types.ExportExcelReply, error) {
	b, _ := json.Marshal(req.Rows)
	sha := crypto.MD5(b)
	export, err := u.repo.GetExportBySha(ctx, sha)
	if err != nil && !gormtranserror.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.SystemError(err.Error())
	}

	if err == nil {
		if export.Status == STATUS_PROGRESS && export.UserId == req.UserId {
			return nil, errors.ExportTaskProcessError()
		}
		// 复制正在进行中的导入数据
		id, err := u.repo.CopyExport(ctx, export, &types.CopyExportRequest{
			UserId:       req.UserId,
			DepartmentId: req.DepartmentId,
			Scene:        req.Scene,
			Name:         req.Name,
		})
		if err != nil {
			return nil, err
		}
		return &types.ExportExcelReply{Id: id, Sha: sha}, nil
	}

	id, err := u.repo.CreateExport(ctx, &entity.Export{
		UserId:       req.UserId,
		DepartmentId: req.DepartmentId,
		Scene:        req.Scene,
		Name:         req.Name,
		Sha:          sha,
		Src:          fmt.Sprintf("%s.zip", sha),
		Status:       STATUS_PROGRESS,
	})
	if err != nil {
		ctx.Logger().Warnw("msg", "create export error", "err", err.Error())
		return nil, errors.DatabaseError(err.Error())
	}

	go func() {
		kCtx := ctx.Clone()
		size, err := u.exportExcel(kCtx, sha, req)
		exp := &entity.Export{
			BaseModel: ktypes.BaseModel{Id: id},
			Status:    STATUS_COMPLETED,
			Size:      size,
			ExpiredAt: time.Now().Unix() + int64(u.conf.Export.Expire.Seconds()),
		}
		if err != nil {
			exp.Status = EXPORT_STATUS_FAIL
			exp.Reason = proto.String(err.Error())
		}

		if err := u.repo.UpdateExport(kCtx, exp); err != nil {
			ctx.Logger().Errorw("msg", "update export status error", "err", err.Error())
		}
	}()

	return &types.ExportExcelReply{Id: id, Sha: sha, Src: fmt.Sprintf("%s.zip", sha)}, nil
}

// nolint
func (u *Export) getFileByValue(ctx kratosx.Context, value string) (*os.File, error) {
	key := value
	if strings.Contains(value, "/") {
		key = value[strings.Index(value, "/")+1:]
	} else if !strings.Contains(value, ".") {
		file, err := u.file.GetFileBySha(ctx, value)
		if err != nil {
			return nil, err
		}
		key = file.Key
	}

	fileName := u.conf.Export.LocalDir + "/tmp/" + key
	if filex.IsExistFile(fileName) {
		return os.Open(fileName)
	}

	reader, err := u.store.Get(key)
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

func (u *Export) exportExcel(ctx kratosx.Context, sha string, req *types.ExportExcelRequest) (uint32, error) {
	// 存储地址
	path := u.conf.Export.LocalDir + "/" + fmt.Sprintf("%s.zip", sha)

	// 表格保存地址
	excelPath := u.conf.Export.LocalDir + "/" + fmt.Sprintf("%s.xlsx", sha)
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
	exports, err := u.fetchFile(ctx, req.Files)
	if err != nil {
		return 0, err
	}

	// 挂载xlsx
	exports[excelPath] = req.Name + ".xlsx"

	// 打包文件
	if err := filex.ZipFiles(path, exports); err != nil {
		return 0, err
	}

	// 获取文件大小
	stat, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return uint32(stat.Size() / 1000), nil
}

// fetchFile 拉取文件, 返回文件路径和文件名
func (u *Export) fetchFile(ctx kratosx.Context, list []*types.ExportFileItem) (map[string]string, error) {
	getKeyFunc := func(ctx kratosx.Context, value string) (string, error) {
		var key = value
		if strings.Contains(value, "/") {
			key = value[strings.Index(value, "/")+1:]
		} else if !strings.Contains(value, ".") {
			file, err := u.file.GetFileBySha(ctx, value)
			if err != nil {
				return "", err
			}
			key = file.Key
		}
		return key, nil
	}

	var exports = make(map[string]string)
	for _, item := range list {
		key, err := getKeyFunc(ctx, item.Value)
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
		path := u.conf.Export.LocalDir + "/tmp/" + key
		if filex.IsExistFile(path) {
			oriExports[path] = rename
			continue
		}

		fd, err := os.Create(path)
		if err != nil {
			ctx.Logger().Errorw("msg", "create file err", "path", path, "err", err.Error())
			continue
		}

		reader, err := u.store.Get(key)
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

func (u *Export) exportFile(ctx kratosx.Context, src string, list []*types.ExportFileItem) (uint32, error) {
	oriExports, err := u.fetchFile(ctx, list)
	if err != nil {
		return 0, err
	}

	src = u.conf.Export.LocalDir + "/" + src
	if err := filex.ZipFiles(src, oriExports); err != nil {
		return 0, err
	}

	stat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}
	return uint32(stat.Size() / 1024), nil
}

// clearExportTmpCache 清理临时文件夹
func (u *Export) clearExportTmpCache() {
	dir := u.conf.Export.LocalDir + "/tmp"
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
		if d.Seconds() >= u.conf.Export.Expire.Seconds() {
			_ = os.RemoveAll(path)
		}
		return err
	})
}

// clearExportFile 清理导出的过期的大文件
func (u *Export) clearExportFile(ctx kratosx.Context) {
	dir := u.conf.Export.LocalDir
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
				BaseModel: ktypes.BaseModel{Id: item.Id},
				Status:    EXPORT_STATUS_EXPIRED,
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
				if err = os.RemoveAll(dir + "/" + item.Src); err != nil {
					ctx.Logger().Warnw("msg", "remove export file status error", "err", err.Error())
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
func (u *Export) ExportFile(ctx kratosx.Context, req *types.ExportFileRequest) (*types.ExportFileReply, error) {
	b, _ := json.Marshal(req.Files)
	ids, _ := json.Marshal(req.Ids)
	sha := crypto.MD5(append(b, ids...))
	export, err := u.repo.GetExportBySha(ctx, sha)
	if err != nil && !gormtranserror.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err == nil {
		if export.Status == STATUS_PROGRESS && export.UserId == req.UserId {
			return nil, errors.ExportTaskProcessError()
		}
		// 复制正在进行中的导入数据
		id, err := u.repo.CopyExport(ctx, export, &types.CopyExportRequest{
			UserId:       req.UserId,
			DepartmentId: req.DepartmentId,
			Scene:        req.Scene,
			Name:         req.Name,
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

	src := fmt.Sprintf("%s.zip", sha)
	id, err := u.repo.CreateExport(ctx, &entity.Export{
		UserId:       req.UserId,
		DepartmentId: req.DepartmentId,
		Scene:        req.Scene,
		Name:         req.Name,
		Sha:          sha,
		Src:          src,
		Status:       STATUS_PROGRESS,
	})
	if err != nil {
		return nil, errors.DatabaseError(err.Error())
	}

	go func() {
		kCtx := ctx.Clone()
		size, err := u.exportFile(kCtx, src, req.Files)
		exp := &entity.Export{
			BaseModel: ktypes.BaseModel{Id: id},
			Status:    STATUS_COMPLETED,
			Size:      size,
			ExpiredAt: time.Now().Unix() + int64(u.conf.Export.Expire.Seconds()),
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
func (u *Export) DeleteExport(ctx kratosx.Context, ids []uint32) (uint32, error) {
	total, err := u.repo.DeleteExport(ctx, ids)
	if err != nil {
		return 0, errors.DeleteError(err.Error())
	}
	return total, nil
}

// GetExport 获取指定的导出信息
func (u *Export) GetExport(ctx kratosx.Context, req *types.GetExportRequest) (*entity.Export, error) {
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

	res.Url, _ = u.store.GenTemporaryURL(res.Src)
	return res, nil
}

// VerifyURL 验证url
func (u *Export) VerifyURL(key, expire, sign string) error {
	return u.store.VerifyTemporaryURL(key, expire, sign)
}

func (s *Export) LocalPath(next http.Handler, src string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = src
		next.ServeHTTP(w, r)
	})
}

func (s *Export) Download() thttp.HandlerFunc {
	return func(ctx thttp.Context) error {
		go func() {
			<-ctx.Done()
			fmt.Println("download file", time.Now().UnixNano())
		}()
		var req pb.DownloadFileRequest
		if err := ctx.BindQuery(&req); err != nil {
			return err
		}
		if err := ctx.BindVars(&req); err != nil {
			return err
		}

		if err := s.VerifyURL(req.Src, req.Expire, req.Sign); err != nil {
			return err
		}

		blw := pkg.NewWriter()
		fs := http.FileServer(http.Dir(s.conf.Export.LocalDir))
		fs = s.LocalPath(fs, req.Src)
		fs.ServeHTTP(blw, ctx.Request())

		header := ctx.Response().Header()
		fn := req.Src
		if req.SaveName != "" {
			fn = req.SaveName + filepath.Ext(req.Src)
		}
		header.Set("Content-Type", "application/octet-stream")
		header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fn))
		ctx.Response().WriteHeader(blw.Code())
		if _, err := ctx.Response().Write(blw.Body()); err != nil {
			return errors.SystemError()
		}

		return nil
	}
}
