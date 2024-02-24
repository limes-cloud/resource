package logic

import (
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"mime"
	"strings"
	"sync"

	"github.com/gabriel-vasile/mimetype"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/limes-cloud/kratosx"
	ktypes "github.com/limes-cloud/kratosx/types"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"

	v1 "github.com/limes-cloud/resource/api/v1"
	"github.com/limes-cloud/resource/config"
	"github.com/limes-cloud/resource/consts"
	"github.com/limes-cloud/resource/internal/model"
	"github.com/limes-cloud/resource/internal/types"
	"github.com/limes-cloud/resource/pkg/image"
	"github.com/limes-cloud/resource/pkg/store"
	"github.com/limes-cloud/resource/pkg/store/aliyun"
	"github.com/limes-cloud/resource/pkg/store/local"
	"github.com/limes-cloud/resource/pkg/store/tencent"
	"github.com/limes-cloud/resource/pkg/util"
)

type File struct {
	conf    *config.Config
	muiOnce map[string]*sync.Once
	rw      sync.RWMutex
}

func NewFile(conf *config.Config) *File {
	return &File{
		conf:    conf,
		muiOnce: make(map[string]*sync.Once),
		rw:      sync.RWMutex{},
	}
}

// GetFileBySha 查询文件
func (f *File) GetFileBySha(ctx kratosx.Context, in *v1.GetFileByShaRequest) (*v1.File, error) {
	file := model.File{}
	if err := file.OneBySha(ctx, in.Sha); err != nil {
		return nil, v1.NotExistFileError()
	}

	file.Src = f.fileSrc(file.Src)

	reply := v1.File{}
	// 进行数据转换
	if err := util.Transform(file, &reply); err != nil {
		return nil, v1.TransformErrorFormat(err.Error())
	}

	return &reply, nil
}

// PageFile 获取分页文件
func (f *File) PageFile(ctx kratosx.Context, in *v1.PageFileRequest) (*v1.PageFileReply, error) {
	file := model.File{}
	list, total, err := file.Page(ctx, &ktypes.PageOptions{
		Page:     in.Page,
		PageSize: in.PageSize,
		Scopes: func(db *gorm.DB) *gorm.DB {
			db = db.Where("directory_id=? and status=?", in.DirectoryId, consts.STATUS_COMPLETED)
			if in.Name != nil {
				db = db.Where("name like ?", *in.Name+"%")
			}
			return db
		},
	})

	if err != nil {
		return nil, v1.DatabaseErrorFormat(err.Error())
	}

	for ind, item := range list {
		list[ind].Src = f.fileSrc(item.Src)
	}

	reply := v1.PageFileReply{Total: &total}
	// 进行数据转换
	if err = util.Transform(list, &reply.List); err != nil {
		return nil, v1.TransformErrorFormat(err.Error())
	}

	return &reply, nil
}

// UpdateFile 修改文件
func (f *File) UpdateFile(ctx kratosx.Context, in *v1.UpdateFileRequest) (*empty.Empty, error) {
	dir := model.Directory{}
	if err := dir.OneByID(ctx, in.DirectoryId); err != nil {
		return nil, v1.NotExistDirectoryError()
	}

	// 需要通过app鉴权，所以这里检测一遍
	if dir.App != in.App {
		return nil, v1.SystemError()
	}

	oldFile := model.File{}
	if err := oldFile.OneByID(ctx, in.Id); err != nil {
		return nil, v1.NotExistFileError()
	}
	// 查询文件名称是否已经存在
	if err := oldFile.OneByDirAndName(ctx, oldFile.DirectoryID, in.Name); err == nil {
		return nil, v1.AlreadyExistFileNameError()
	}

	file := model.File{
		BaseModel: ktypes.BaseModel{ID: in.Id},
		Name:      in.Name,
	}

	if err := file.Update(ctx); err != nil {
		return nil, v1.DatabaseErrorFormat(err.Error())
	}
	return nil, nil
}

// DeleteFile 删除文件
func (f *File) DeleteFile(ctx kratosx.Context, in *v1.DeleteFileRequest) (*empty.Empty, error) {
	dir := model.Directory{}
	if err := dir.OneByID(ctx, in.DirectoryId); err != nil {
		return nil, v1.NotExistDirectoryError()
	}

	// 需要通过app鉴权，所以这里检测一遍
	if dir.App != in.App {
		return nil, v1.SystemError()
	}

	file := model.File{}
	if err := file.DeleteByDirAndIds(ctx, in.DirectoryId, in.Ids); err != nil {
		return nil, v1.DatabaseErrorFormat(err.Error())
	}
	return nil, nil
}

// PrepareUploadFile 预上传文件
func (f *File) PrepareUploadFile(ctx kratosx.Context, in *v1.PrepareUploadFileRequest) (*v1.PrepareUploadFileReply, error) {
	if in.DirectoryPath == "" && in.DirectoryId == 0 {
		return nil, v1.ParamsError()
	}

	dir := model.Directory{}
	if in.DirectoryId != 0 {
		if err := dir.OneByID(ctx, in.DirectoryId); err != nil {
			return nil, v1.UploadFileErrorFormat("不存在文件夹")
		}
		if dir.App != in.App {
			return nil, v1.SystemError()
		}
	} else {
		paths := strings.Split(in.DirectoryPath, "/")
		if err := dir.OneByPaths(ctx, in.App, paths); err != nil {
			return nil, err
		}
	}

	// 获取文件个数，重置文件名
	file := model.File{}
	if count, _ := file.CountByName(ctx, file.Name); count != 0 {
		file.Name = fmt.Sprintf("%s（%d）", file.Name, count-1)
	}

	// 判断是否存在文件，存在则进行秒传
	if err := file.OneBySha(ctx, in.Sha); err == nil {
		// 存在且已经上传完成
		if file.Status == consts.STATUS_COMPLETED {
			_ = file.Copy(ctx, dir.BaseModel.ID, in.Name)
			return &v1.PrepareUploadFileReply{
				Uploaded: proto.Bool(true),
				Src:      proto.String(f.fileSrc(file.Src)),
				Sha:      proto.String(file.Sha),
			}, nil
		}

		// 存在但是处于上传中
		uploadChunks := []uint32{}
		if file.ChunkCount > 1 && file.UploadID != nil {
			store, err := f.NewStore(ctx)
			if err != nil {
				return nil, v1.InitStoreError()
			}
			chunk, err := store.NewPutChunkByUploadID(file.Src, *file.UploadID)
			if err != nil {
				return nil, v1.ChunkUploadError()
			}
			_ = util.Transform(chunk.UploadedChunkIndex(), &uploadChunks)
		}
		return &v1.PrepareUploadFileReply{
			Uploaded:     proto.Bool(false),
			UploadId:     file.UploadID,
			ChunkSize:    proto.Uint32(uint32(f.maxChunkSize())),
			ChunkCount:   proto.Uint32(file.ChunkCount),
			UploadChunks: uploadChunks,
			Sha:          proto.String(file.Sha),
		}, nil
	}

	// 检查文件大小
	if err := f.checkSize(int64(in.Size)); err != nil {
		return nil, err
	}

	// 检查文件后缀
	if err := f.checkType(f.getType(in.Name)); err != nil {
		return nil, err
	}

	fileType := f.getType(in.Name)
	file = model.File{
		DirectoryID: dir.BaseModel.ID,
		Size:        in.Size,
		Sha:         in.Sha,
		Name:        in.Name,
		Src:         f.storeKey(in.Sha, fileType),
		Status:      consts.STATUS_PROGRESS,
		Storage:     f.conf.Storage,
		Type:        fileType,
		UploadID:    proto.String(uuid.NewString()),
		ChunkCount:  1,
	}

	// 判断是否需要切片
	if f.maxSingularSize() < int64(in.Size) {
		store, err := f.NewStore(ctx)
		if err != nil {
			return nil, err
		}

		pc, err := store.NewPutChunk(file.Src)
		if err != nil {
			return nil, v1.ChunkUploadError()
		}
		file.UploadID = proto.String(pc.UploadID())
		file.ChunkCount = uint32(f.chunkCount(int64(in.Size)))
	}

	if err := file.Create(ctx); err != nil {
		return nil, v1.DatabaseError()
	}

	return &v1.PrepareUploadFileReply{
		Uploaded:     proto.Bool(false),
		UploadId:     file.UploadID,
		ChunkSize:    proto.Uint32(uint32(f.maxChunkSize())),
		ChunkCount:   proto.Uint32(file.ChunkCount),
		UploadChunks: []uint32{},
	}, nil
}

// UploadFile 上传文件
func (f *File) UploadFile(ctx kratosx.Context, in *v1.UploadFileRequest) (*v1.UploadFileReply, error) {
	f.rw.Lock()
	if f.muiOnce[in.UploadId] == nil {
		f.muiOnce[in.UploadId] = &sync.Once{}
	}
	f.rw.Unlock()

	file := model.File{}
	if err := file.OneByUploadID(ctx, in.UploadId); err != nil {
		return nil, v1.UploadFileErrorFormat("上传id不存在")
	}

	fileByte, err := base64.StdEncoding.DecodeString(in.Data)
	if err != nil {
		return nil, v1.FileFormatError()
	}

	if file.Status == consts.STATUS_COMPLETED {
		return nil, v1.UploadFileErrorFormat("请勿重复上传")
	}

	file.Status = consts.STATUS_COMPLETED

	store, err := f.NewStore(ctx)
	if err != nil {
		return nil, err
	}

	// 直接上传
	if file.ChunkCount == 1 {
		if err := store.PutBytes(file.Src, fileByte); err != nil {
			return nil, v1.UploadFileError()
		}
		_ = file.Update(ctx)
		return &v1.UploadFileReply{
			Src: f.fileSrc(file.Src),
			Sha: file.Sha,
		}, nil
	}

	// 切片上传
	chunk, err := store.NewPutChunkByUploadID(file.Src, in.UploadId)
	if err != nil {
		return nil, v1.ChunkUploadError()
	}

	if err := chunk.AppendBytes(fileByte, int(in.Index)); err != nil {
		return nil, v1.ChunkUploadError()
	}

	if chunk.ChunkCount() == int(file.ChunkCount) {
		f.rw.RLock()
		if f.muiOnce[in.UploadId] != nil {
			f.muiOnce[in.UploadId].Do(func() {
				_ = chunk.Complete()
				_ = file.Update(ctx)
			})
		}
		delete(f.muiOnce, in.UploadId)
		f.rw.RUnlock()
	}

	return &v1.UploadFileReply{
		Src: f.fileSrc(file.Src),
		Sha: file.Sha,
	}, nil
}

// GetFile 上传文件
func (f *File) GetFile(ctx kratosx.Context, in *types.GetFileRequest) (*types.GetFileResponse, error) {
	store, err := f.NewStore(ctx)
	if err != nil {
		return nil, v1.SystemError()
	}
	reader, err := store.Get(in.Src)
	if err != nil {
		return nil, v1.NotExistResourceError()
	}
	rb, _ := io.ReadAll(reader)

	fileMime := mime.TypeByExtension("." + f.getType(in.Src))
	if fileMime == "" {
		fileMime = f.fileMime(rb)
	}
	// 如果是图片，则进行裁剪
	if strings.Contains(fileMime, "image/") && in.Width > 0 && in.Height > 0 {
		tp := strings.Split(fileMime, "/")[1]
		if img, err := image.New(tp, rb); err == nil {
			if in.Mode == "" {
				in.Mode = image.AspectFill
			}
			if nrb, err := img.Resize(in.Width, in.Height, in.Mode); err == nil {
				rb = nrb
			}
		}
	}

	return &types.GetFileResponse{
		Data: rb,
		Mime: fileMime,
	}, nil
}

// NewStore 新建存储引擎
func (f *File) NewStore(ctx kratosx.Context) (store.Store, error) {
	c := &store.Config{
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
		return nil, v1.NoSupportStoreError()
	}
}

// chunkCount 通过文件大小获取分片数量
func (f *File) chunkCount(size int64) int {
	return int(math.Ceil(float64(size) / float64(f.maxChunkSize())))
}

// getType 获取文件类型
func (f *File) getType(name string) string {
	index := strings.LastIndex(name, ".")
	suffix := ""
	if index != -1 {
		suffix = name[index+1:]
	}
	return suffix
}

// storeKey 获取存储的key
func (f *File) storeKey(sha, tp string) string {
	return fmt.Sprintf("%s.%s", sha, tp)
}

// checkType 检查文件类型是否合法
func (f *File) checkType(tp string) error {
	if !util.InList(f.conf.AcceptTypes, tp) {
		return v1.UploadFileErrorFormat("不支持的文件后缀")
	}
	return nil
}

// checkSize 检查大小是否合法
func (f *File) checkSize(size int64) error {
	if size > f.maxChunkSize()*f.conf.MaxChunkCount {
		return v1.UploadFileErrorFormat("超过传输文件大小")
	}
	return nil
}

// maxSingularSize 获取单个文件的最大大小,单位KB
func (f *File) maxSingularSize() int64 {
	return f.conf.MaxSingularSize * 1024
}

// maxChunkSize 获取分片的大小 单位KB
func (f *File) maxChunkSize() int64 {
	return f.conf.MaxChunkSize * 1024
}

func (f *File) fileSrcFormat() string {
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

func (f *File) fileSrc(src string) string {
	return strings.Replace(f.fileSrcFormat(), "{src}", src, 1)
}

// fileMime 获取文件的Mime
func (f *File) fileMime(body []byte) string {
	return mimetype.Detect(body).String()
}
