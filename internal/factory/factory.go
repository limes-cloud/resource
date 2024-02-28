package factory

import (
	"fmt"
	"math"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/pkg/util"

	"github.com/limes-cloud/resource/api/errors"
	"github.com/limes-cloud/resource/internal/config"
	"github.com/limes-cloud/resource/internal/consts"
	store2 "github.com/limes-cloud/resource/internal/pkg/store"
	"github.com/limes-cloud/resource/internal/pkg/store/aliyun"
	"github.com/limes-cloud/resource/internal/pkg/store/local"
	"github.com/limes-cloud/resource/internal/pkg/store/tencent"
)

type Factory struct {
	conf *config.Config
}

func New(conf *config.Config) *Factory {
	return &Factory{conf: conf}
}

func (f *Factory) Storage() string {
	return f.conf.Storage
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
	if !util.InList(f.conf.AcceptTypes, tp) {
		return errors.UploadFileFormat("不支持的文件后缀")
	}
	return nil
}

// CheckSize 检查大小是否合法
func (f *Factory) CheckSize(size int64) error {
	if size > f.MaxChunkSize()*f.conf.MaxChunkCount {
		return errors.UploadFileFormat("超过传输文件大小")
	}
	return nil
}

// MaxSingularSize 获取单个文件的最大大小,单位KB
func (f *Factory) MaxSingularSize() int64 {
	return f.conf.MaxSingularSize * 1024
}

// MaxChunkSize 获取分片的大小 单位KB
func (f *Factory) MaxChunkSize() int64 {
	return f.conf.MaxChunkSize * 1024
}

func (f *Factory) FileSrcFormat() string {
	switch f.conf.Storage {
	case consts.STORE_ALIYUN:
		return "https://" + f.conf.Bucket + ".oss-cn-" + f.conf.Region + ".aliyuncs.com" + "/{src}"
	case consts.STORE_TENCENT:
		return "https://" + f.conf.Bucket + ".cos." + f.conf.Region + ".myqcloud.com" + "/{src}"
	case consts.STORE_LOCAL:
		return f.conf.ServerPath + "/{src}"
	}
	return "%s"
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
		Endpoint: f.conf.Endpoint,
		Key:      f.conf.Key,
		Secret:   f.conf.Secret,
		Bucket:   f.conf.Bucket,
		LocalDir: f.conf.LocalDir,
		DB:       ctx.DB(),
	}
	switch f.conf.Storage {
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
