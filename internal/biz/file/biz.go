package file

import (
	oe "errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/pkg/valx"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"

	"github.com/limes-cloud/resource/api/resource/errors"
	"github.com/limes-cloud/resource/internal/conf"
	"github.com/limes-cloud/resource/internal/pkg/util"
)

type UseCase struct {
	conf *conf.Config
	repo Repo
	rw   sync.RWMutex
	mui  map[string]*sync.Once
}

func NewUseCase(config *conf.Config, repo Repo) *UseCase {
	return &UseCase{conf: config, repo: repo, mui: make(map[string]*sync.Once), rw: sync.RWMutex{}}
}

// GetFile 获取指定的文件信息
func (u *UseCase) GetFile(ctx kratosx.Context, req *GetFileRequest) (*File, error) {
	var (
		res *File
		err error
	)

	if req.Id != nil {
		res, err = u.repo.GetFile(ctx, *req.Id)
	} else if req.Sha != nil {
		res, err = u.repo.GetFileBySha(ctx, *req.Sha)
	} else {
		return nil, errors.ParamsError()
	}

	if res.Status != STATUS_COMPLETED {
		return nil, errors.NotExistFileError()
	}

	if err != nil {
		return nil, errors.NotExistFileError(err.Error())
	}
	return res, nil
}

// ListFile 获取文件信息列表
func (u *UseCase) ListFile(ctx kratosx.Context, req *ListFileRequest) ([]*File, uint32, error) {
	list, total, err := u.repo.ListFile(ctx, req)
	if err != nil {
		return nil, 0, errors.ListError(err.Error())
	}
	return list, total, nil
}

// PrepareUploadFile 预上传文件信息
func (u *UseCase) PrepareUploadFile(ctx kratosx.Context, req *PrepareUploadFileRequest) (*PrepareUploadFileReply, error) {
	var (
		err         error
		limit       *DirectoryLimit
		directoryId uint32
	)
	if req.DirectoryId != nil {
		limit, err = u.repo.GetDirectoryLimitById(ctx, *req.DirectoryId)
	} else {
		paths := strings.Split(*req.DirectoryPath, "/")
		limit, err = u.repo.GetDirectoryLimitByPath(ctx, paths)
	}
	if err != nil {
		return nil, errors.DatabaseError(err.Error())
	}
	directoryId = limit.DirectoryId
	chunkSize := util.GetKBSize(u.conf.ChunkSize)

	// 校验是否存在上传记录
	oldFile, err := u.repo.GetFileBySha(ctx, req.Sha)
	if err != nil && !oe.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.UpdateError(err.Error())
	}
	if err == nil {
		// 触发秒传
		if oldFile.Status == STATUS_COMPLETED {
			if err := u.repo.CopyFile(ctx, oldFile, directoryId, req.Name); err != nil {
				return nil, errors.UploadFileError(err.Error())
			}
			return &PrepareUploadFileReply{
				Uploaded: true,
				Src:      proto.String(oldFile.Src),
				Sha:      proto.String(oldFile.Sha),
				URL:      proto.String(oldFile.URL),
			}, nil
		}
		// 触发断点续传
		chunkFactory, err := u.repo.GetStore().NewPutChunkByUploadID(oldFile.Sha, oldFile.UploadId)
		if err != nil {
			ctx.Logger().Warnf("get upload chunks error:%s", err.Error())
		}
		return &PrepareUploadFileReply{
			Uploaded:     false,
			UploadId:     proto.String(oldFile.UploadId),
			ChunkSize:    proto.Uint32(chunkSize),
			ChunkCount:   proto.Uint32(oldFile.ChunkCount),
			UploadChunks: chunkFactory.UploadedChunkIndex(),
			Sha:          proto.String(oldFile.Sha),
		}, nil
	}

	// 校验文件大小
	if size := util.GetKBSize(limit.MaxSize); size < req.Size {
		return nil, errors.ExceedMaxSizeError()
	}

	// 校验文件类型
	tp := util.GetFileType(req.Name)
	if !valx.InList(limit.Accepts, tp) {
		return nil, errors.NoSupportFileTypeError()
	}

	// 构建文件对象
	file := &File{
		DirectoryId: directoryId,
		Size:        req.Size,
		Sha:         req.Sha,
		Name:        req.Name,
		Src:         fmt.Sprintf("%s.%s", req.Sha, tp),
		Status:      STATUS_PROGRESS,
		Type:        tp,
		UploadId:    uuid.NewString(),
		ChunkCount:  1,
	}

	// 判断是否需要切片
	if chunkSize < req.Size {
		file.ChunkCount = uint32(math.Ceil(float64(req.Size) / float64(chunkSize)))
		chunkFactory, err := u.repo.GetStore().NewPutChunk(file.Src)
		if err != nil {
			return nil, errors.UpdateError(err.Error())
		}
		file.UploadId = chunkFactory.UploadID()
	}

	if _, err = u.repo.CreateFile(ctx, file); err != nil {
		return nil, errors.UpdateError(err.Error())
	}

	return &PrepareUploadFileReply{
		Uploaded:     false,
		UploadId:     proto.String(file.UploadId),
		ChunkSize:    proto.Uint32(chunkSize),
		ChunkCount:   proto.Uint32(file.ChunkCount),
		UploadChunks: nil,
	}, nil
}

// UploadFile 上传文件信息
func (u *UseCase) UploadFile(ctx kratosx.Context, req *UploadFileRequest) (*UploadFileReply, error) {
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
		if err = u.repo.GetStore().PutBytes(file.Src, req.Data); err != nil {
			return nil, err
		}
		if err = u.repo.UpdateFileStatus(ctx, file.Id, STATUS_COMPLETED); err != nil {
			return nil, errors.UploadFileError(err.Error())
		}
	} else {
		chunkFactory, err := u.repo.GetStore().NewPutChunkByUploadID(file.Src, req.UploadId)
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
					if err := u.repo.UpdateFileStatus(ctx, file.Id, STATUS_COMPLETED); err != nil {
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

	return &UploadFileReply{
		Src: file.Src,
		Sha: file.Sha,
		URL: file.URL,
	}, nil
}

// UpdateFile 更新文件信息
func (u *UseCase) UpdateFile(ctx kratosx.Context, req *File) error {
	if err := u.repo.UpdateFile(ctx, req); err != nil {
		return errors.UpdateError(err.Error())
	}
	return nil
}

// DeleteFile 删除文件信息
func (u *UseCase) DeleteFile(ctx kratosx.Context, ids []uint32) (uint32, error) {
	total, err := u.repo.DeleteFile(ctx, ids)
	if err != nil {
		return 0, errors.DeleteError(err.Error())
	}
	return total, nil
}

// VerifyURL 验证访问url
func (u *UseCase) VerifyURL(key string, expire string, sign string) error {
	if err := u.repo.GetStore().VerifyTemporaryURL(key, expire, sign); err != nil {
		return errors.VerifySignError(err.Error())
	}
	return nil
}
