package service

import (
	"bytes"
	"fmt"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/limes-cloud/resource/internal/pkg/image"

	thttp "github.com/go-kratos/kratos/v2/transport/http"
	pb "github.com/limes-cloud/resource/api/resource/file/v1"

	"github.com/google/uuid"
	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/library/db/gormtranserror"
	"github.com/limes-cloud/kratosx/pkg/valx"
	ktypes "github.com/limes-cloud/kratosx/types"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"

	"github.com/limes-cloud/resource/api/resource/errors"
	"github.com/limes-cloud/resource/internal/conf"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/repository"
	"github.com/limes-cloud/resource/internal/pkg"
	"github.com/limes-cloud/resource/internal/types"
)

const (
	STATUS_PROGRESS  = "PROGRESS"
	STATUS_COMPLETED = "COMPLETED"
)

type File struct {
	conf      *conf.Config
	rw        sync.RWMutex
	mui       map[string]*sync.Once
	repo      repository.File
	store     repository.Store
	directory repository.Directory
}

func NewFile(
	conf *conf.Config,
	repo repository.File,
	directory repository.Directory,
	store repository.Store,
) *File {
	return &File{
		conf:      conf,
		mui:       make(map[string]*sync.Once),
		rw:        sync.RWMutex{},
		repo:      repo,
		store:     store,
		directory: directory,
	}
}

// GetFile 获取指定的文件信息
func (u *File) GetFile(ctx kratosx.Context, req *types.GetFileRequest) (*entity.File, error) {
	var (
		res *entity.File
		err error
	)

	if req.Id != nil {
		res, err = u.repo.GetFile(ctx, *req.Id)
	} else if req.Sha != nil {
		res, err = u.repo.GetFileBySha(ctx, *req.Sha)
	} else if req.Src != nil {
		res, err = u.repo.GetFileBySha(ctx, *req.Src)
	} else {
		return nil, errors.ParamsError()
	}
	if err != nil {
		return nil, errors.GetError(err.Error())
	}
	if res.Status != STATUS_COMPLETED {
		return nil, errors.NotExistFileError()
	}
	if err != nil {
		return nil, errors.NotExistFileError(err.Error())
	}
	res.Url, _ = u.store.GenTemporaryURL(res.Key)
	return res, nil
}

// ListFile 获取文件信息列表
func (u *File) ListFile(ctx kratosx.Context, req *types.ListFileRequest) ([]*entity.File, uint32, error) {
	list, total, err := u.repo.ListFile(ctx, req)
	if err != nil {
		return nil, 0, errors.ListError(err.Error())
	}
	for ind, item := range list {
		url, err := u.store.GenTemporaryURL(item.Key)
		if err != nil {
			continue
		}
		list[ind].Url = url
	}
	return list, total, nil
}

// PrepareUploadFile 预上传文件信息
func (u *File) PrepareUploadFile(ctx kratosx.Context, req *types.PrepareUploadFileRequest) (*types.PrepareUploadFileReply, error) {
	var (
		err         error
		limit       *entity.DirectoryLimit
		directoryId uint32
	)
	if req.DirectoryId != nil {
		limit, err = u.directory.GetDirectoryLimitById(ctx, *req.DirectoryId)
	} else {
		paths := strings.Split(*req.DirectoryPath, "/")
		limit, err = u.directory.GetDirectoryLimitByPath(ctx, paths)
	}
	if err != nil {
		return nil, errors.DatabaseError(err.Error())
	}
	directoryId = limit.DirectoryId
	chunkSize := pkg.GetKBSize(u.conf.ChunkSize)

	// 校验是否存在上传记录
	oldFile, err := u.repo.GetFileBySha(ctx, req.Sha)
	if err != nil && !gormtranserror.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.UpdateError(err.Error())
	}
	if err == nil {
		// 触发秒传
		if oldFile.Status == STATUS_COMPLETED {
			if err := u.repo.CopyFile(ctx, oldFile, directoryId, req.Name); err != nil {
				return nil, errors.UploadFileError(err.Error())
			}
			url, _ := u.store.GenTemporaryURL(oldFile.Key)
			return &types.PrepareUploadFileReply{
				Uploaded: true,
				Src:      proto.String(oldFile.Src),
				Sha:      proto.String(oldFile.Sha),
				Url:      proto.String(url),
			}, nil
		}
		// 触发断点续传
		chunkFactory, err := u.store.NewPutChunkByUploadID(oldFile.Sha, oldFile.UploadId)
		if err != nil {
			ctx.Logger().Warnf("get upload chunks error:%s", err.Error())
		}
		return &types.PrepareUploadFileReply{
			Uploaded:     false,
			UploadId:     proto.String(oldFile.UploadId),
			ChunkSize:    proto.Uint32(chunkSize),
			ChunkCount:   proto.Uint32(oldFile.ChunkCount),
			UploadChunks: chunkFactory.UploadedChunkIndex(),
			Sha:          proto.String(oldFile.Sha),
		}, nil
	}

	// 校验文件大小
	if size := pkg.GetKBSize(limit.MaxSize); size < req.Size {
		return nil, errors.ExceedMaxSizeError()
	}

	// 校验文件类型
	tp := pkg.GetFileType(req.Name)
	if !valx.InList(limit.Accepts, tp) {
		return nil, errors.NoSupportFileTypeError()
	}

	// 构建文件对象
	file := &entity.File{
		DirectoryId: directoryId,
		Size:        req.Size,
		Sha:         req.Sha,
		Name:        req.Name,
		Key:         fmt.Sprintf("%s.%s", req.Sha, tp),
		Status:      STATUS_PROGRESS,
		Type:        tp,
		UploadId:    uuid.NewString(),
		ChunkCount:  1,
	}

	// 判断是否需要切片
	if chunkSize < req.Size {
		file.ChunkCount = uint32(math.Ceil(float64(req.Size) / float64(chunkSize)))
		chunkFactory, err := u.store.NewPutChunk(file.Key)
		if err != nil {
			return nil, errors.UpdateError(err.Error())
		}
		file.UploadId = chunkFactory.UploadID()
	}

	if _, err = u.repo.CreateFile(ctx, file); err != nil {
		return nil, errors.UpdateError(err.Error())
	}

	return &types.PrepareUploadFileReply{
		Uploaded:     false,
		UploadId:     proto.String(file.UploadId),
		ChunkSize:    proto.Uint32(chunkSize),
		ChunkCount:   proto.Uint32(file.ChunkCount),
		UploadChunks: nil,
	}, nil
}

// UploadFile 上传文件信息
func (u *File) UploadFile(ctx kratosx.Context, req *types.UploadFileRequest) (*types.UploadFileReply, error) {
	file, err := u.repo.GetFileByUploadId(ctx, req.UploadId)
	if err != nil {
		return nil, errors.UpdateError("不存在上传任务")
	}
	if file.Status == STATUS_COMPLETED {
		return nil, errors.UpdateError("请勿重复上传")
	}

	if err != nil {
		return nil, errors.UpdateError(err.Error())
	}

	// 直接上传
	if file.ChunkCount == 1 {
		if err = u.store.PutBytes(file.Key, req.Data); err != nil {
			return nil, err
		}
		if err = u.repo.UpdateFile(ctx, &entity.File{
			BaseModel: ktypes.BaseModel{Id: file.Id},
			Status:    STATUS_COMPLETED,
		}); err != nil {
			return nil, errors.UploadFileError(err.Error())
		}
	} else {
		chunkFactory, err := u.store.NewPutChunkByUploadID(file.Key, req.UploadId)
		if err != nil {
			return nil, errors.UpdateError(err.Error())
		}
		if err = chunkFactory.AppendBytes(req.Data, int(req.Index)); err != nil {
			return nil, err
		}
		u.rw.Lock()
		if u.mui[req.UploadId] == nil {
			u.mui[req.UploadId] = &sync.Once{}
		}
		u.rw.Unlock()

		// 当前已经上传完成
		if chunkFactory.ChunkCount() == int(file.ChunkCount) {
			u.rw.RLock()
			if u.mui[req.UploadId] != nil {
				var cErr error
				u.mui[req.UploadId].Do(func() {
					if err := chunkFactory.Complete(); err != nil {
						cErr = err
						return
					}
					if err := u.repo.UpdateFile(ctx, &entity.File{
						BaseModel: ktypes.BaseModel{Id: file.Id},
						Status:    STATUS_COMPLETED,
					}); err != nil {
						cErr = err
					}
					go func() {
						time.Sleep(10 * time.Second)
						delete(u.mui, req.UploadId)
					}()
				})
				if cErr != nil {
					return nil, errors.UpdateError(err.Error())
				}
			}
			u.rw.RUnlock()
		}
	}
	url, _ := u.store.GenTemporaryURL(file.Key)
	return &types.UploadFileReply{
		Src: file.Src,
		Sha: file.Sha,
		Url: url,
	}, nil
}

// UpdateFile 更新文件信息
func (u *File) UpdateFile(ctx kratosx.Context, req *entity.File) error {
	if err := u.repo.UpdateFile(ctx, req); err != nil {
		return errors.UpdateError(err.Error())
	}
	return nil
}

// DeleteFile 删除文件信息
func (u *File) DeleteFile(ctx kratosx.Context, ids []uint32) (uint32, error) {
	total, err := u.repo.DeleteFile(ctx, ids, func(file *entity.File) {
		if file.Status == STATUS_COMPLETED {
			_ = u.store.Delete(file.Key)
		} else {
			chunk, err := u.store.NewPutChunkByUploadID(file.Key, file.UploadId)
			if err == nil {
				_ = chunk.Abort()
			}
		}
	})
	if err != nil {
		return 0, errors.DeleteError(err.Error())
	}
	return total, nil
}

func (s *File) LocalPath(next http.Handler, src string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = src
		next.ServeHTTP(w, r)
	})
}

func (s *File) SrcBlob() thttp.HandlerFunc {
	return func(ctx thttp.Context) error {
		var req pb.StaticFileRequest
		if err := ctx.BindQuery(&req); err != nil {
			return err
		}
		if err := ctx.BindVars(&req); err != nil {
			return err
		}

		if err := s.store.VerifyTemporaryURL(req.Src, req.Expire, req.Sign); err != nil {
			return err
		}

		blw := pkg.NewWriter()
		fs := http.FileServer(http.Dir(s.conf.Storage.LocalDir))
		fs = s.LocalPath(fs, req.Src)
		fs.ServeHTTP(blw, ctx.Request())

		// 处理图片裁剪
		cType := blw.Header().Get("Content-Type")
		if strings.Contains(cType, "image/") && req.Width > 0 && req.Height > 0 {
			blw.Header().Del("Content-Length")
			tp := strings.Split(cType, "/")[1]
			rb := blw.Body()
			if img, err := image.New(tp, rb); err == nil {
				if req.Mode == "" {
					req.Mode = image.AspectFill
				}
				if nrb, err := img.Resize(int(req.Width), int(req.Height), req.Mode); err == nil {
					blw.SetBody(bytes.NewBuffer(nrb))
					blw.Header().Set("Content-Length", strconv.Itoa(len(nrb)))
				}
			}
		}

		// 处理返回
		header := ctx.Response().Header()
		blwHeader := blw.Header()
		for key := range blwHeader {
			header.Set(key, blwHeader.Get(key))
		}

		if req.Download {
			fn := req.Src
			if req.SaveName != "" {
				fn = req.SaveName + filepath.Ext(req.Src)
			}
			header.Set("Content-Type", "application/octet-stream")
			header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fn))
		}

		ctx.Response().WriteHeader(blw.Code())
		if _, err := ctx.Response().Write(blw.Body()); err != nil {
			return errors.SystemError()
		}

		return nil
	}
}
