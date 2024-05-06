package factory

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/pkg/util"
	"github.com/limes-cloud/kratosx/pkg/xlsx"

	"github.com/limes-cloud/resource/api/errors"
	"github.com/limes-cloud/resource/internal/biz/export"
	"github.com/limes-cloud/resource/internal/biz/file"
	"github.com/limes-cloud/resource/internal/config"
	"github.com/limes-cloud/resource/internal/consts"
	store2 "github.com/limes-cloud/resource/internal/pkg/store"
	"github.com/limes-cloud/resource/internal/pkg/store/aliyun"
	"github.com/limes-cloud/resource/internal/pkg/store/local"
	"github.com/limes-cloud/resource/internal/pkg/store/tencent"
)

type Factory struct {
	conf       *config.Config
	fileRepo   file.Repo
	exportRepo export.Repo
}

var (
	_ins *Factory
	once sync.Once
)

func New(conf *config.Config, fileRepo file.Repo, exportRepo export.Repo) *Factory {
	once.Do(func() {
		_ins = &Factory{conf: conf, fileRepo: fileRepo, exportRepo: exportRepo}
		go func() {
			_ins.ClearExportCache()
			time.Sleep(1 * time.Hour)
		}()
	})
	return _ins
}

func (f *Factory) Storage() string {
	return f.conf.Storage.Type
}

// ChunkCount 通过文件大小获取分片数量
func (f *Factory) ChunkCount(size int64) int {
	return int(math.Ceil(float64(size) / float64(f.MaxChunkSize())))
}

// GetType 获取文件类型
func (f *Factory) GetType(name string) string {
	index := strings.LastIndex(name, ".")
	suffix := ""
	if index != -1 {
		suffix = name[index+1:]
	}
	return suffix
}

// StoreKey 获取存储的key
func (f *Factory) StoreKey(sha, tp string) string {
	return fmt.Sprintf("%s.%s", sha, tp)
}

// CheckType 检查文件类型是否合法
func (f *Factory) CheckType(tp string) error {
	if !util.InList(f.conf.Storage.AcceptTypes, tp) {
		return errors.UploadFileFormat("不支持的文件后缀")
	}
	return nil
}

// CheckSize 检查大小是否合法
func (f *Factory) CheckSize(size int64) error {
	if size > f.MaxChunkSize()*f.conf.Storage.MaxChunkCount {
		return errors.UploadFileFormat("超过传输文件大小")
	}
	return nil
}

// MaxSingularSize 获取单个文件的最大大小,单位KB
func (f *Factory) MaxSingularSize() int64 {
	return f.conf.Storage.MaxSingularSize * 1024
}

// MaxChunkSize 获取分片的大小 单位KB
func (f *Factory) MaxChunkSize() int64 {
	return f.conf.Storage.MaxChunkSize * 1024
}

func (f *Factory) FileSrcFormat() string {
	switch f.Storage() {
	case consts.STORE_ALIYUN:
		return "https://" + f.conf.Storage.Bucket + ".oss-cn-" + f.conf.Storage.Region + ".aliyuncs.com" + "/{src}"
	case consts.STORE_TENCENT:
		return "https://" + f.conf.Storage.Bucket + ".cos." + f.conf.Storage.Region + ".myqcloud.com" + "/{src}"
	case consts.STORE_LOCAL:
		return f.conf.Storage.ServerPath + "/{src}"
	}
	return "%s"
}

func (f *Factory) ExportFileSrc(src string) string {
	prefix := f.conf.Export.LocalDir
	if index := strings.Index(f.conf.Export.LocalDir, "/"); index != -1 {
		prefix = f.conf.Export.LocalDir[index:]
	}
	return f.conf.Storage.ServerPath + prefix + "/" + src
}

func (f *Factory) FileSrc(src string) string {
	return strings.Replace(f.FileSrcFormat(), "{src}", src, 1)
}

// FileMime 获取文件的Mime
func (f *Factory) FileMime(body []byte) string {
	return mimetype.Detect(body).String()
}

func (f *Factory) Store(ctx kratosx.Context) (store2.Store, error) {
	c := &store2.Config{
		Endpoint: f.conf.Storage.Endpoint,
		Key:      f.conf.Storage.Key,
		Secret:   f.conf.Storage.Secret,
		Bucket:   f.conf.Storage.Bucket,
		LocalDir: f.conf.Storage.LocalDir,
		DB:       ctx.DB(),
	}
	switch f.Storage() {
	case consts.STORE_ALIYUN:
		return aliyun.New(c)
	case consts.STORE_TENCENT:
		return tencent.New(c)
	case consts.STORE_LOCAL:
		return local.New(c)
	default:
		return nil, errors.NoSupportStore()
	}
}

// ExportFile 导出指定的文件列表
func (f *Factory) ExportFile(ctx kratosx.Context, in *export.AddExportRequest) (int64, error) {
	if util.IsExistFile(in.Name) {
		_ = os.Chtimes(in.Name, time.Now(), time.Now())
		stat, err := os.Stat(in.Name)
		if err != nil {
			return 0, err
		}
		return stat.Size(), nil
	}

	dir := f.conf.Export.LocalDir
	if !util.IsExistFolder(dir) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return 0, err
		}
	}

	var exports = make(map[string]string)

	if len(in.Ids) != 0 {
		for _, id := range in.Ids {
			fe, err := f.fileRepo.GetFileByID(ctx, id)
			if err != nil {
				ctx.Logger().Errorw("msg", "get file error", "err", err.Error())
				continue
			}
			exports[fe.Src] = fe.Src
		}
	} else {
		for _, item := range in.Files {
			if item.Sha == "" {
				continue
			}
			fe, err := f.fileRepo.GetFileBySha(ctx, item.Sha)
			if err != nil {
				ctx.Logger().Errorw("msg", "get file error", "err", err.Error())
				continue
			}
			exports[fe.Src] = fe.Src
			if item.Rename != "" {
				exports[fe.Src] = item.Rename + filepath.Ext(fe.Src)
			}
		}
	}

	store, err := f.Store(ctx)
	if err != nil {
		return 0, err
	}
	if f.Storage() != consts.STORE_LOCAL {
		var remoteExports = make(map[string]string)
		for src, rename := range exports {
			path := dir + "/" + rename
			if util.IsExistFile(path) {
				continue
			}

			fd, err := os.Create(path)
			if err != nil {
				ctx.Logger().Errorw("msg", "create file err", "path", path, "err", err.Error())
				continue
			}

			reader, err := store.Get(src)
			if err != nil {
				ctx.Logger().Errorw("msg", "get remote file error", "path", path, "err", err.Error())
				continue
			}

			if _, err := io.Copy(fd, reader); err != nil {
				ctx.Logger().Errorw("msg", "save remote file error", "path", path, "download err", err.Error())
				continue
			}

			remoteExports[path] = rename + filepath.Ext(path)
		}
		exports = remoteExports
	} else {
		var localExports = make(map[string]string)
		for src, rename := range exports {
			path := strings.ReplaceAll(f.conf.Storage.LocalDir+"/"+src, "//", "/")
			localExports[path] = rename
		}
		exports = localExports
	}
	if err := util.ZipFiles(in.Name, exports); err != nil {
		return 0, err
	}

	stat, err := os.Stat(in.Name)
	if err != nil {
		return 0, err
	}
	return stat.Size(), nil
}

// ExportExcel 导出指定的数据列表为excel
func (f *Factory) ExportExcel(ctx kratosx.Context, in *export.AddExportExcelRequest) (int64, error) {
	if util.IsExistFile(in.Name) {
		_ = os.Chtimes(in.Name, time.Now(), time.Now())
		stat, err := os.Stat(in.Name)
		if err != nil {
			return 0, err
		}
		return stat.Size(), nil
	}
	dir := f.conf.Export.LocalDir
	if !util.IsExistFolder(dir) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return 0, err
		}
	}

	store, err := f.Store(ctx)
	if err != nil {
		return 0, err
	}

	xlsxFile := xlsx.New(in.Name).Writer()
	for _, list := range in.Rows {
		var temp []any
		for _, item := range list {
			switch item.Type {
			case "image":
				if item.Value == "" {
					continue
				}
				fe, err := f.fileRepo.GetFileBySha(ctx.Clone(), item.Value)
				if err != nil {
					ctx.Logger().Errorw("msg", "get file error", "err", err.Error())
					continue
				}
				if f.Storage() != consts.STORE_LOCAL {
					path := dir + "/" + fe.Src
					if util.IsExistFile(path) {
						if fd, err := os.Open(path); err == nil {
							temp = append(temp, fd)
							continue
						}
					}

					fd, err := os.Create(path)
					if err != nil {
						ctx.Logger().Errorw("path", path, "create file err", err.Error())
						continue
					}

					reader, err := store.Get(fe.Src)
					if err != nil {
						ctx.Logger().Errorw("path", path, "get err", err.Error())
						continue
					}

					if _, err := io.Copy(fd, reader); err != nil {
						ctx.Logger().Errorw("path", path, "download err", err.Error())
						continue
					}
					temp = append(temp, fd)
				} else {
					path := strings.ReplaceAll(f.conf.Storage.LocalDir+"/"+fe.Src, "//", "/")
					fd, err := os.Open(path)
					if err != nil {
						ctx.Logger().Errorw("path", path, "download err", err.Error())
						continue
					}
					temp = append(temp, fd)
				}

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
	stat, err := os.Stat(in.Name)
	if err != nil {
		return 0, err
	}
	return stat.Size(), nil
}

func (f *Factory) ClearExportCache() {
	dir := f.conf.Export.LocalDir
	if !util.IsExistFolder(dir) {
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
		if d.Seconds() >= f.conf.Export.Expire.Seconds() {
			_ = os.RemoveAll(path)
		}
		return err
	})
	_ = f.exportRepo.UpdateExportExpire(
		kratosx.MustContext(context.Background()),
		time.Now().Unix()-int64(f.conf.Export.Expire.Seconds()),
	)
}
