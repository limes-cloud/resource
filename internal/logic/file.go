package logic

import (
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"mime"
	v1 "resource/api/v1"
	"resource/config"
	"resource/consts"
	"resource/internal/model"
	"resource/internal/types"
	"resource/pkg/image"
	"resource/pkg/store"
	"resource/pkg/store/aliyun"
	"resource/pkg/store/local"
	"resource/pkg/store/tencent"
	"resource/pkg/util"
	"strings"
	"sync"

	"github.com/gabriel-vasile/mimetype"

	"github.com/golang/protobuf/ptypes/empty"

	"gorm.io/gorm"

	"github.com/google/uuid"

	"google.golang.org/protobuf/proto"

	"github.com/limes-cloud/kratos"
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

// PageFile 获取分页文件
func (f *File) PageFile(ctx kratos.Context, in *v1.PageFileRequest) (*v1.PageFileReply, error) {
	file := model.File{}
	list, total, err := file.Page(ctx, &model.PageOptions{
		Page:     in.Page,
		PageSize: in.PageSize,
		Scopes: func(db *gorm.DB) *gorm.DB {
			db = db.Where("directory_id=?", in.DirectoryId)
			if in.Name != nil {
				db = db.Where("name like ?", *in.Name+"%")
			}
			return db
		},
	})

	if err != nil {
		return nil, v1.ErrorDatabaseFormat(err.Error())
	}

	for ind, item := range list {
		list[ind].Src = f.fileSrc(item.Src)
	}

	reply := v1.PageFileReply{Total: &total}
	// 进行数据转换
	if err = util.Transform(list, &reply.List); err != nil {
		return nil, v1.ErrorTransformFormat(err.Error())
	}

	return &reply, nil
}

// UpdateFile 修改文件
func (f *File) UpdateFile(ctx kratos.Context, in *v1.UpdateFileRequest) (*empty.Empty, error) {
	dir := model.Directory{}
	if err := dir.OneByID(ctx, in.DirectoryId); err != nil {
		return nil, v1.ErrorNotExistDirectory()
	}

	// 需要通过app鉴权，所以这里检测一遍
	if dir.App != in.App {
		return nil, v1.ErrorSystem()
	}

	oldFile := model.File{}
	if err := oldFile.OneByID(ctx, in.Id); err != nil {
		return nil, v1.ErrorNotExistFile()
	}
	// 查询文件名称是否已经存在
	if err := oldFile.OneByDirAndName(ctx, oldFile.DirectoryID, in.Name); err == nil {
		return nil, v1.ErrorAlreadyExistFileName()
	}

	file := model.File{
		BaseModel: model.BaseModel{ID: in.Id},
		Name:      in.Name,
	}

	if err := file.Update(ctx); err != nil {
		return nil, v1.ErrorDatabaseFormat(err.Error())
	}
	return nil, nil
}

// DeleteFile 删除文件
func (f *File) DeleteFile(ctx kratos.Context, in *v1.DeleteFileRequest) (*empty.Empty, error) {
	dir := model.Directory{}
	if err := dir.OneByID(ctx, in.DirectoryId); err != nil {
		return nil, v1.ErrorNotExistDirectory()
	}

	// 需要通过app鉴权，所以这里检测一遍
	if dir.App != in.App {
		return nil, v1.ErrorSystem()
	}

	file := model.File{}
	if err := file.DeleteByDirAndIds(ctx, in.DirectoryId, in.Ids); err != nil {
		return nil, v1.ErrorDatabaseFormat(err.Error())
	}
	return nil, nil
}

// PrepareUploadFile 预上传文件
func (f *File) PrepareUploadFile(ctx kratos.Context, in *v1.PrepareUploadFileRequest) (*v1.PrepareUploadFileReply, error) {
	dir := model.Directory{}
	if err := dir.OneByID(ctx, in.DirectoryId); err != nil {
		return nil, v1.ErrorUploadFileFormat("不存在文件夹")
	}

	// 需要通过app鉴权，所以这里检测一遍
	if dir.App != in.App {
		return nil, v1.ErrorSystem()
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
			_ = file.Copy(ctx, in.DirectoryId, in.Name)
			return &v1.PrepareUploadFileReply{
				Uploaded: proto.Bool(true),
				Src:      proto.String(f.fileSrc(file.Src)),
				Sha:      proto.String(file.Sha),
			}, nil
		}

		// 存在但是处于上传中
		uploadChunks := []uint32{}
		if file.ChunkCount > 1 {
			store, err := f.NewStore(ctx)
			if err != nil {
				return nil, v1.ErrorInitStore()
			}
			chunk, err := store.NewPutChunkByUploadID(file.Src, file.UploadID)
			if err != nil {
				return nil, v1.ErrorChunkUpload()
			}
			_ = util.Transform(chunk.UploadedChunkIndex(), &uploadChunks)
		}
		return &v1.PrepareUploadFileReply{
			Uploaded:     proto.Bool(false),
			UploadId:     proto.String(file.UploadID),
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
		DirectoryID: in.DirectoryId,
		Size:        in.Size,
		Sha:         in.Sha,
		Name:        in.Name,
		Src:         f.storeKey(in.Sha, fileType),
		Status:      consts.STATUS_PROGRESS,
		Storage:     f.conf.Storage,
		Type:        fileType,
		UploadID:    uuid.NewString(),
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
			return nil, v1.ErrorChunkUpload()
		}
		file.UploadID = pc.UploadID()
		file.ChunkCount = uint32(f.chunkCount(int64(in.Size)))
	}

	if err := file.Create(ctx); err != nil {
		return nil, v1.ErrorDatabase()
	}

	return &v1.PrepareUploadFileReply{
		Uploaded:     proto.Bool(false),
		UploadId:     proto.String(file.UploadID),
		ChunkSize:    proto.Uint32(uint32(f.maxChunkSize())),
		ChunkCount:   proto.Uint32(file.ChunkCount),
		UploadChunks: []uint32{},
	}, nil
}

// UploadFile 上传文件
func (f *File) UploadFile(ctx kratos.Context, in *v1.UploadFileRequest) (*v1.UploadFileReply, error) {
	f.rw.Lock()
	if f.muiOnce[in.UploadId] == nil {
		f.muiOnce[in.UploadId] = &sync.Once{}
	}
	f.rw.Unlock()

	file := model.File{}
	if err := file.OneByUploadID(ctx, in.UploadId); err != nil {
		return nil, v1.ErrorUploadFileFormat("上传id不存在")
	}

	fileByte, err := base64.StdEncoding.DecodeString(in.Data)
	if err != nil {
		return nil, v1.ErrorFileFormat()
	}

	if file.Status == consts.STATUS_COMPLETED {
		return nil, v1.ErrorUploadFileFormat("请勿重复上传")
	}

	file.Status = consts.STATUS_COMPLETED

	store, err := f.NewStore(ctx)
	if err != nil {
		return nil, err
	}

	// 直接上传
	if file.ChunkCount == 1 {
		if err := store.PutBytes(file.Src, fileByte); err != nil {
			return nil, v1.ErrorUploadFile()
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
		return nil, v1.ErrorChunkUpload()
	}

	if err := chunk.AppendBytes(fileByte, int(in.Index)); err != nil {
		return nil, v1.ErrorChunkUpload()
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
func (f *File) GetFile(ctx kratos.Context, in *types.GetFileRequest) (*types.GetFileResponse, error) {
	store, err := f.NewStore(ctx)
	if err != nil {
		return nil, v1.ErrorSystem()
	}
	reader, err := store.Get(in.Src)
	if err != nil {
		return nil, v1.ErrorNotExistResource()
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
func (f *File) NewStore(ctx kratos.Context) (store.Store, error) {
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
		return nil, v1.ErrorNoSupportStore()
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
		return v1.ErrorUploadFileFormat("不支持的文件后缀")
	}
	return nil
}

// checkSize 检查大小是否合法
func (f *File) checkSize(size int64) error {
	if size > f.maxChunkSize()*f.conf.MaxChunkCount {
		return v1.ErrorUploadFileFormat("超过传输文件大小")
	}
	return nil
}

// maxSingularSize 获取单个文件的最大大小
func (f *File) maxSingularSize() int64 {
	return f.conf.MaxSingularSize * 1024 * 1024
}

// maxChunkSize 获取分片的大小
func (f *File) maxChunkSize() int64 {
	return f.conf.MaxChunkSize * 1024 * 1024
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
