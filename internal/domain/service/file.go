package service

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"

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
	return &File{repo: repo, userRepo: userRepo, directory: directory, newStore: newStore}
}

func (u *File) getStoreByKey(key string) (repository.Store, error) {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) == 2 {
		return u.newStore(parts[0])
	}
	return u.newStore()
}

func (u *File) getDirectoryLimit(ctx core.Context, dirId *uint32, dirPath *string) (*entity.DirectoryLimit, error) {
	if dirId != nil {
		return u.directory.GetDirectoryLimitById(ctx, *dirId)
	}
	return u.directory.GetDirectoryLimitByPath(ctx, strings.Split(*dirPath, "/"))
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

// GetFileBytes 获取文件二进制文件
func (u *File) GetFileBytes(ctx core.Context, key string, reply types.GetFileBytesFunc) error {
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
	limit, err := u.getDirectoryLimit(ctx, req.DirectoryId, req.DirectoryPath)
	if err != nil {
		return nil, errors.DatabaseError(err.Error())
	}

	st, err := u.newStore(req.Store)
	if err != nil {
		return nil, errors.SystemError(err.Error())
	}

	chunkSize := pkg.GetKBSize(ctx.Config().ChunkSize)

	oldFile, err := u.repo.GetFileBySha(ctx, st.Config().Keyword, req.Sha)
	if err != nil && !gormtranserror.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.UpdateError(err.Error())
	}

	if err == nil {
		// 秒传
		if oldFile.Status == STATUS_COMPLETED {
			has, err := u.userRepo.IsExistUserFile(ctx, ctx.Auth().UserId, oldFile.Id)
			if err != nil {
				return nil, errors.UploadFileError(err.Error())
			}
			if !has {
				if _, err := u.userRepo.CreateUserFile(ctx, &entity.UserFile{
					DirectoryId: limit.DirectoryId,
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

		// 断点续传
		chunkFactory, err := st.NewPutChunkByUploadID(oldFile.Key, oldFile.UploadId)
		if err != nil {
			ctx.Logger().Warnw("msg", "get upload chunks error", "err", err.Error())
			return nil, errors.SystemError()
		}
		uploadedChunks := chunkFactory.UploadedChunkIndex()
		if len(uploadedChunks) == int(oldFile.ChunkCount) {
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
			UploadChunks: uploadedChunks,
			Sha:          proto.String(oldFile.Sha),
			Key:          proto.String(oldFile.Key),
		}, nil
	}

	// 校验文件大小和类型
	if size := pkg.GetKBSize(limit.MaxSize); size < req.Size {
		return nil, errors.ExceedMaxSizeError()
	}
	tp := pkg.GetFileType(req.Name)
	if !value.InList(limit.Accepts, tp) {
		return nil, errors.NoSupportFileTypeError()
	}

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
		_, err = u.userRepo.CreateUserFile(ctx, &entity.UserFile{
			DirectoryId: limit.DirectoryId,
			Name:        req.Name,
			FileId:      id,
		})
		return err
	})
	if err != nil {
		return nil, errors.UpdateError(err.Error())
	}

	return &types.PrepareUploadFileReply{
		Uploaded:   false,
		UploadId:   proto.String(fe.UploadId),
		ChunkSize:  proto.Uint32(chunkSize),
		ChunkCount: proto.Uint32(fe.ChunkCount),
		Key:        proto.String(fe.Key),
	}, nil
}

// UploadFileByURL 上传文件信息
func (u *File) UploadFileByURL(ctx core.Context, req *types.UploadFileByURLRequest) (*types.UploadFileByURLReply, error) {
	resp, err := http.Get(req.URL)
	if err != nil {
		return nil, errors.SystemError(err.Error())
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.SystemError(err.Error())
	}

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
	return &types.UploadFileByURLReply{Sha: reply.Sha, Key: reply.Key}, nil
}

// UploadFile 上传文件信息
func (u *File) UploadFile(ctx core.Context, req *types.UploadFileRequest) (*types.UploadFileReply, error) {
	limit, err := u.getDirectoryLimit(ctx, req.DirectoryId, req.DirectoryPath)
	if err != nil {
		return nil, errors.DatabaseError(err.Error())
	}

	st, err := u.newStore(req.Store)
	if err != nil {
		return nil, errors.SystemError(err.Error())
	}

	fileSize := uint32(len(req.Data)) / 1024
	if size := pkg.GetKBSize(limit.MaxSize); size < fileSize {
		return nil, errors.ExceedMaxSizeError()
	}

	tp := pkg.GetFileType(req.Name)
	if !value.InList(limit.Accepts, tp) {
		return nil, errors.NoSupportFileTypeError()
	}

	fileKey := fmt.Sprintf("%s/%s.%s", st.Config().Keyword, req.Sha, tp)

	oldFile, err := u.repo.GetFileBySha(ctx, st.Config().Keyword, req.Sha)
	if err != nil && !gormtranserror.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.UpdateError(err.Error())
	}
	if err == nil && oldFile.Status == STATUS_COMPLETED {
		has, err := u.userRepo.IsExistUserFile(ctx, ctx.Auth().UserId, oldFile.Id)
		if err != nil {
			return nil, errors.UploadFileError(err.Error())
		}
		if !has {
			if _, err := u.userRepo.CreateUserFile(ctx, &entity.UserFile{
				DirectoryId: limit.DirectoryId,
				Name:        req.Name,
				FileId:      oldFile.Id,
			}); err != nil {
				return nil, errors.UploadFileError(err.Error())
			}
		}
		return &types.UploadFileReply{Key: oldFile.Key, Sha: oldFile.Sha}, nil
	}

	fe := &entity.File{
		Store:      st.Config().Keyword,
		Size:       fileSize,
		Sha:        req.Sha,
		Key:        fileKey,
		Status:     STATUS_COMPLETED,
		Type:       tp,
		UploadId:   uuid.NewString(),
		ChunkCount: 1,
	}

	err = ctx.Transaction(func(ctx core.Context) error {
		var fileID uint32
		if oldFile != nil && oldFile.Id != 0 {
			fileID = oldFile.Id
			if err := u.repo.UpdateFile(ctx, &entity.File{
				BaseModel: model.BaseModel{Id: oldFile.Id},
				Status:    STATUS_COMPLETED,
				Size:      fileSize,
			}); err != nil {
				return err
			}
		} else {
			var err error
			fileID, err = u.repo.CreateFile(ctx, fe)
			if err != nil {
				return err
			}
		}
		if _, err := u.userRepo.CreateUserFile(ctx, &entity.UserFile{
			DirectoryId: limit.DirectoryId,
			Name:        req.Name,
			FileId:      fileID,
		}); err != nil {
			return err
		}
		return st.PutBytes(fe.Key, req.Data)
	})
	if err != nil {
		return nil, errors.UpdateError(err.Error())
	}
	return &types.UploadFileReply{Sha: req.Sha, Key: fe.Key}, nil
}

// UploadChunkFile 上传分片文件
func (u *File) UploadChunkFile(ctx core.Context, req *types.UploadChunkFileRequest) (*types.UploadFileReply, error) {
	fe, err := u.repo.GetFileByUploadId(ctx, req.UploadId)
	if err != nil {
		return nil, errors.UpdateError("不存在上传任务")
	}
	if fe.Status == STATUS_COMPLETED {
		return &types.UploadFileReply{Sha: fe.Sha, Key: fe.Key}, nil
	}

	st, err := u.getStoreByKey(fe.Key)
	if err != nil {
		return nil, errors.SystemError(err.Error())
	}

	chunkFactory, err := st.NewPutChunkByUploadID(fe.Key, req.UploadId)
	if err != nil {
		return nil, errors.UpdateError(err.Error())
	}
	if err = chunkFactory.AppendBytes(req.Data, int(req.Index)); err != nil {
		return nil, err
	}

	// 最后一片到达时合并
	if int(req.Index) == int(fe.ChunkCount) {
		if err := chunkFactory.Complete(); err != nil {
			return nil, errors.UpdateError(err.Error())
		}
		if err := u.repo.UpdateFile(ctx, &entity.File{
			BaseModel: model.BaseModel{Id: fe.Id},
			Status:    STATUS_COMPLETED,
		}); err != nil {
			return nil, errors.UploadFileError(err.Error())
		}
	}

	return &types.UploadFileReply{Sha: fe.Sha, Key: fe.Key}, nil
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
			if chunk, err := st.NewPutChunkByUploadID(file.Key, file.UploadId); err == nil {
				_ = chunk.Abort()
			}
		}
	})
	if err != nil {
		return 0, errors.DeleteError(err.Error())
	}
	return total, nil
}
