package service

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/limes-cloud/kratosx/pkg/crypto"

	"github.com/google/uuid"
	"github.com/limes-cloud/kratosx/library/db/gormtranserror"
	"github.com/limes-cloud/kratosx/model"
	"github.com/limes-cloud/kratosx/pkg/value"
	"github.com/limes-cloud/resource/api/errors"
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/repository"
	"github.com/limes-cloud/resource/internal/pkg"
	"github.com/limes-cloud/resource/internal/types"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

const (
	STATUS_PROGRESS  = "PROGRESS"
	STATUS_COMPLETED = "COMPLETED"
)

type File struct {
	rw        sync.RWMutex
	mui       map[string]*sync.Once
	repo      repository.File
	userRepo  repository.UserFile
	directory repository.Directory
	newStore  func(keyword ...string) (repository.Store, error)
}

func NewFile(
	repo repository.File,
	userRepo repository.UserFile,
	directory repository.Directory,
	newStore func(keyword ...string) (repository.Store, error),
) *File {
	return &File{
		mui:       make(map[string]*sync.Once),
		rw:        sync.RWMutex{},
		repo:      repo,
		userRepo:  userRepo,
		directory: directory,
		newStore:  newStore,
	}
}

// GetUserFile 获取用户指定的文件信息
func (u *File) GetUserFile(ctx core.Context, req *types.GetUserFileRequest) (*entity.UserFile, error) {
	var (
		err error
		res *entity.File
	)
	if req.Id != nil {
		res, err = u.repo.GetFile(ctx, *req.Id)
	} else if req.Key != nil {
		res, err = u.repo.GetFileByKey(ctx, *req.Key)
	} else {
		return nil, errors.ParamsError()
	}
	if err != nil {
		return nil, errors.NotExistFileError(err.Error())
	}
	if res.Status != STATUS_COMPLETED {
		return nil, errors.NotExistFileError()
	}

	if req.Directory != nil {
		dir, err := u.directory.GetDirectoryLimitByPath(ctx, strings.Split(*req.Directory, "/"))
		if err != nil {
			return nil, errors.GetError()
		}
		req.DirectoryId = dir.DirectoryId
	}

	req.FileId = res.Id
	req.UserId = ctx.Auth().UserId
	uf, err := u.userRepo.GetUserFile(ctx, req)
	if err != nil {
		return nil, err
	}

	uf.File = res
	return uf, nil
}

func (u *File) getStoreByKey(key string) (repository.Store, error) {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) == 2 {
		return u.newStore(parts[0])
	}
	return u.newStore()
}

// GetFileBytes 获取文件二进制文件
func (u *File) GetFileBytes(ctx core.Context, key string, reply types.GetFileBytesFunc) error {
	// 获取key
	fe, err := u.repo.GetFileByKey(ctx, key)
	if err != nil {
		return errors.NotExistFileError()
	}

	st, err := u.getStoreByKey(key)
	if err != nil {
		return err
	}

	reader, err := st.Get(fe.Key)
	if err != nil {
		return errors.GetError(err.Error())
	}
	buf := make([]byte, 32*1024)
	for {
		nr, er := reader.Read(buf)
		if nr > 0 {
			if err := reply(buf[:nr]); err != nil {
				return err
			}
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			return er
		}
	}
	return nil
}

// ListUserFile 获取文件信息列表
func (u *File) ListUserFile(ctx core.Context, req *types.ListFileRequest) ([]*entity.UserFile, uint32, error) {
	list, total, err := u.userRepo.ListUserFile(ctx, req)
	if err != nil {
		return nil, 0, errors.ListError(err.Error())
	}
	return list, total, nil
}

// PrepareUploadFile 预上传文件信息
func (u *File) PrepareUploadFile(ctx core.Context, req *types.PrepareUploadFileRequest) (*types.PrepareUploadFileReply, error) {
	var (
		err         error
		limit       *entity.DirectoryLimit
		directoryId uint32
		conf        = ctx.Config()
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
	chunkSize := pkg.GetKBSize(conf.ChunkSize)

	st, err := u.newStore(req.Store)
	if err != nil {
		return nil, errors.SystemError(err.Error())
	}

	// 校验是否存在上传记录
	oldFile, err := u.repo.GetFileBySha(ctx, st.Config().Keyword, req.Sha)
	if err != nil && !gormtranserror.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.UpdateError(err.Error())
	}

	if err == nil {
		// 触发秒传
		if oldFile.Status == STATUS_COMPLETED {
			// 判断当前用户是否已经拥有了图片
			has, err := u.userRepo.IsExistUserFile(ctx, ctx.Auth().UserId, oldFile.Id)
			if err != nil {
				return nil, errors.UploadFileError(err.Error())
			}
			if !has {
				if _, err := u.userRepo.CreateUserFile(ctx, &entity.UserFile{
					DirectoryId: directoryId,
					Name:        req.Name,
					FileId:      oldFile.Id,
				}); err != nil {
					return nil, errors.UploadFileError(err.Error())
				}
			}

			return &types.PrepareUploadFileReply{
				Uploaded: true,
				Key:      proto.String(oldFile.Key),
				Sha:      proto.String(oldFile.Sha),
			}, nil
		}

		// 触发断点续传
		chunkFactory, err := st.NewPutChunkByUploadID(oldFile.Sha, oldFile.UploadId)
		if err != nil {
			ctx.Logger().Warnw("msg", "get upload chunks error", "err", err.Error())
			return nil, errors.SystemError()
		}
		// 判断是否完成，完成则合并
		if len(chunkFactory.UploadedChunkIndex()) == int(oldFile.ChunkCount) {
			if err := chunkFactory.Complete(); err != nil {
				return nil, errors.SystemError(err.Error())
			}
			if err := u.repo.UpdateFile(ctx, &entity.File{
				BaseModel: model.BaseModel{Id: oldFile.Id},
				Status:    STATUS_COMPLETED,
			}); err != nil {
				return nil, errors.SystemError(err.Error())
			}
			return &types.PrepareUploadFileReply{
				Uploaded: true,
				Sha:      proto.String(oldFile.Sha),
				Key:      proto.String(oldFile.Key),
			}, nil
		}

		return &types.PrepareUploadFileReply{
			Uploaded:     false,
			UploadId:     proto.String(oldFile.UploadId),
			ChunkSize:    proto.Uint32(chunkSize),
			ChunkCount:   proto.Uint32(oldFile.ChunkCount),
			UploadChunks: chunkFactory.UploadedChunkIndex(),
			Sha:          proto.String(oldFile.Sha),
			Key:          proto.String(oldFile.Key),
		}, nil
	}

	// 校验文件大小
	if size := pkg.GetKBSize(limit.MaxSize); size < req.Size {
		return nil, errors.ExceedMaxSizeError()
	}

	// 校验文件类型
	tp := pkg.GetFileType(req.Name)
	if !value.InList(limit.Accepts, tp) {
		return nil, errors.NoSupportFileTypeError()
	}

	// 构建文件对象
	fe := &entity.File{
		Store:      st.Config().Keyword,
		Size:       req.Size,
		Sha:        req.Sha,
		Key:        fmt.Sprintf("%s/%s.%s", st.Config().Keyword, req.Sha, tp),
		Status:     STATUS_PROGRESS,
		Type:       tp,
		UploadId:   uuid.NewString(),
		ChunkCount: 1,
	}

	// 判断是否需要切片
	if chunkSize < req.Size {
		fe.ChunkCount = uint32(math.Ceil(float64(req.Size) / float64(chunkSize)))
		chunkFactory, err := st.NewPutChunk(fe.Key)
		if err != nil {
			return nil, errors.UpdateError(err.Error())
		}
		fe.UploadId = chunkFactory.UploadID()
	}

	err = ctx.Transaction(func(ctx core.Context) error {
		id, err := u.repo.CreateFile(ctx, fe)
		if err != nil {
			return err
		}
		if _, err = u.userRepo.CreateUserFile(ctx, &entity.UserFile{
			DirectoryId: directoryId,
			Name:        req.Name,
			FileId:      id,
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, errors.UpdateError(err.Error())
	}

	return &types.PrepareUploadFileReply{
		Uploaded:     false,
		UploadId:     proto.String(fe.UploadId),
		ChunkSize:    proto.Uint32(chunkSize),
		ChunkCount:   proto.Uint32(fe.ChunkCount),
		UploadChunks: nil,
		Key:          proto.String(fmt.Sprintf("%s.%s", req.Sha, tp)),
	}, nil
}

// UploadFileByURL 上传文件信息
func (u *File) UploadFileByURL(ctx core.Context, req *types.UploadFileByURLRequest) (*types.UploadFileByURLReply, error) {
	// 下载URL
	resp, err := http.Get(req.URL)
	if err != nil {
		return nil, errors.SystemError(err.Error())
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.SystemError(err.Error())
	}

	// 上传文件
	reply, err := u.UploadFile(ctx, &types.UploadFileRequest{
		DirectoryPath: req.DirectoryPath,
		Store:         req.Store,
		Name:          req.Name,
		Data:          b,
		Sha:           crypto.MD5(b),
	})
	if err != nil {
		return nil, err
	}

	return &types.UploadFileByURLReply{
		Sha: reply.Sha,
		Key: reply.Key,
	}, nil
}

// UploadFile 上传文件信息
func (u *File) UploadFile(ctx core.Context, req *types.UploadFileRequest) (*types.UploadFileReply, error) {
	var (
		err         error
		limit       *entity.DirectoryLimit
		hasUserFile bool
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

	st, err := u.newStore(req.Store)
	if err != nil {
		return nil, errors.SystemError(err.Error())
	}

	// 校验文件大小
	fileSize := pkg.GetKBSize(uint32(len(req.Data)))
	if size := pkg.GetKBSize(limit.MaxSize * 1024 * 1024); size < fileSize {
		return nil, errors.ExceedMaxSizeError()
	}

	// 校验文件类型
	tp := pkg.GetFileType(req.Name)
	if !value.InList(limit.Accepts, tp) {
		return nil, errors.NoSupportFileTypeError()
	}

	// 校验是否存在上传记录
	oldFile, err := u.repo.GetFileBySha(ctx, st.Config().Keyword, req.Sha)
	if err != nil && !gormtranserror.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.UpdateError(err.Error())
	}
	if err == nil {
		// 判断当前用户是否已经拥有了图片
		has, err := u.userRepo.IsExistUserFile(ctx, ctx.Auth().UserId, oldFile.Id)
		if err != nil && !gormtranserror.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.UploadFileError(err.Error())
		}
		hasUserFile = has
		if oldFile.Status == STATUS_COMPLETED {
			if !has {
				if _, err := u.userRepo.CreateUserFile(ctx, &entity.UserFile{
					DirectoryId: limit.DirectoryId,
					Name:        req.Name,
					FileId:      oldFile.Id,
				}); err != nil {
					return nil, errors.UploadFileError(err.Error())
				}
			}

			return &types.UploadFileReply{
				Key: oldFile.Key,
				Sha: oldFile.Sha,
			}, nil
		}

	}

	// 构建文件对象
	fe := &entity.File{
		Store:      st.Config().Keyword,
		Size:       fileSize,
		Sha:        req.Sha,
		Key:        fmt.Sprintf("%s/%s.%s", st.Config().Keyword, req.Sha, tp),
		Status:     STATUS_COMPLETED,
		Type:       tp,
		UploadId:   uuid.NewString(),
		ChunkCount: 1,
	}

	err = ctx.Transaction(func(ctx core.Context) error {
		var fileID uint32
		if oldFile != nil && oldFile.Id != 0 {
			fileID = oldFile.Id
			if err = u.repo.UpdateFile(ctx, &entity.File{
				BaseModel: model.BaseModel{Id: oldFile.Id},
				Status:    STATUS_COMPLETED,
				Size:      fileSize,
			}); err != nil {
				return err
			}
		} else {
			fileID, err = u.repo.CreateFile(ctx, fe)
			if err != nil {
				return err
			}
		}
		if !hasUserFile {
			if _, err = u.userRepo.CreateUserFile(ctx, &entity.UserFile{
				DirectoryId: limit.DirectoryId,
				Name:        req.Name,
				FileId:      fileID,
			}); err != nil {
				return err
			}
		}

		return st.PutBytes(fe.Key, req.Data)
	})
	if err != nil {
		return nil, errors.UpdateError(err.Error())
	}

	return &types.UploadFileReply{
		Sha: req.Sha,
		Key: fe.Key,
	}, nil
}

// UploadChunkFile 上传文件信息
func (u *File) UploadChunkFile(ctx core.Context, req *types.UploadChunkFileRequest) (*types.UploadFileReply, error) {
	fe, err := u.repo.GetFileByUploadId(ctx, req.UploadId)
	if err != nil {
		return nil, errors.UpdateError("不存在上传任务")
	}

	if fe.Status == STATUS_COMPLETED {
		return nil, errors.UpdateError("请勿重复上传")
	}

	st, err := u.getStoreByKey(fe.Key)
	if err != nil {
		return nil, errors.SystemError(err.Error())
	}

	// 直接上传
	if fe.ChunkCount == 1 {
		if err = st.PutBytes(fe.Key, req.Data); err != nil {
			return nil, err
		}
		if err = u.repo.UpdateFile(ctx, &entity.File{
			BaseModel: model.BaseModel{Id: fe.Id},
			Status:    STATUS_COMPLETED,
		}); err != nil {
			return nil, errors.UploadFileError(err.Error())
		}
	} else {
		chunkFactory, err := st.NewPutChunkByUploadID(fe.Key, req.UploadId)
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
		if chunkFactory.ChunkCount() == int(fe.ChunkCount) {
			u.rw.RLock()
			if u.mui[req.UploadId] != nil {
				var cErr error
				u.mui[req.UploadId].Do(func() {
					defer func() {
						go func() {
							time.Sleep(10 * time.Second)
							delete(u.mui, req.UploadId)
						}()
					}()

					if err := chunkFactory.Complete(); err != nil {
						cErr = err
						return
					}
					if err := u.repo.UpdateFile(ctx, &entity.File{
						BaseModel: model.BaseModel{Id: fe.Id},
						Status:    STATUS_COMPLETED,
					}); err != nil {
						cErr = err
					}
				})
				if cErr != nil {
					return nil, errors.UpdateError(err.Error())
				}
			}
			u.rw.RUnlock()
		}
	}
	return &types.UploadFileReply{
		Sha: fe.Sha,
		Key: fe.Key,
	}, nil
}

// UpdateUserFile 更新文件信息
func (u *File) UpdateUserFile(ctx core.Context, req *entity.UserFile) error {
	if err := u.userRepo.UpdateUserFile(ctx, req); err != nil {
		return errors.UpdateError(err.Error())
	}
	return nil
}

// DeleteUserFile 删除文件信息
func (u *File) DeleteUserFile(ctx core.Context, ids []uint32) (uint32, error) {
	total, err := u.userRepo.DeleteUserFile(ctx, ids, func(file *entity.File) {
		st, err := u.getStoreByKey(file.Key)
		if err != nil {
			return
		}

		if file.Status == STATUS_COMPLETED {
			_ = st.Delete(file.Key)
		} else {
			chunk, err := st.NewPutChunkByUploadID(file.Key, file.UploadId)
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
