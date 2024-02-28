package file

import (
	"io"
	"mime"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/limes-cloud/kratosx"

	"github.com/limes-cloud/resource/api/errors"
	"github.com/limes-cloud/resource/internal/config"
	"github.com/limes-cloud/resource/internal/consts"
	"github.com/limes-cloud/resource/internal/factory"
	"github.com/limes-cloud/resource/internal/pkg/image"
)

type UseCase struct {
	config  *config.Config
	repo    Repo
	factory *factory.Factory
	muiOnce map[string]*sync.Once
	rw      sync.RWMutex
}

func NewUseCase(config *config.Config, repo Repo) *UseCase {
	return &UseCase{config: config, repo: repo, factory: factory.New(config)}
}

func (u *UseCase) AllDirectoryByParentID(ctx kratosx.Context, pid uint32, app string) ([]*Directory, error) {
	list, err := u.repo.AllDirectoryByParentID(ctx, pid, app)
	if err != nil {
		return nil, errors.Database()
	}
	return list, nil
}

func (u *UseCase) AddDirectory(ctx kratosx.Context, in *Directory) (uint32, error) {
	if in.ParentID != 0 {
		directory, err := u.repo.GetDirectoryByID(ctx, in.ParentID)
		if err != nil {
			return 0, errors.NotExistDirectory()
		}
		if directory.App != in.App {
			return 0, errors.System()
		}
	}
	id, err := u.repo.AddDirectory(ctx, in)
	if err != nil {
		return 0, errors.DatabaseFormat(err.Error())
	}
	return id, nil
}

func (u *UseCase) UpdateDirectory(ctx kratosx.Context, in *Directory) error {
	directory, err := u.repo.GetDirectoryByID(ctx, in.ID)
	if err != nil {
		return errors.NotExistDirectory()
	}
	if directory.App != in.App {
		return errors.System()
	}

	if err := u.repo.UpdateDirectory(ctx, in); err != nil {
		return errors.DatabaseFormat(err.Error())
	}
	return nil
}

func (u *UseCase) DeleteDirectory(ctx kratosx.Context, id uint32, app string) error {
	directory, err := u.repo.GetDirectoryByID(ctx, id)
	if err != nil {
		return errors.NotExistDirectory()
	}
	if directory.App != app {
		return errors.System()
	}

	// 是否存在文件
	if count, _ := u.repo.FileCountByDirectoryID(ctx, id); count != 0 {
		return errors.DeleteDirectoryFormat("当前目录下存在文件或目录")
	}

	// 判断是否存在目录
	if count, _ := u.repo.DirectoryCountByParentID(ctx, id); count != 0 {
		return errors.DeleteDirectoryFormat("当前目录下存在文件或目录")
	}

	if err := u.repo.DeleteDirectory(ctx, id); err != nil {
		return errors.Database()
	}
	return nil
}

func (u *UseCase) GetFile(ctx kratosx.Context, in *GetFileRequest) (*GetFileResponse, error) {
	store, err := u.factory.Store(ctx)
	if err != nil {
		return nil, errors.System()
	}
	reader, err := store.Get(in.Src)
	if err != nil {
		return nil, errors.NotExistResource()
	}
	rb, _ := io.ReadAll(reader)

	fileMime := mime.TypeByExtension("." + u.factory.GetType(in.Src))
	if fileMime == "" {
		fileMime = u.factory.FileMime(rb)
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

	return &GetFileResponse{
		Data: rb,
		Mime: fileMime,
	}, nil
}

func (u *UseCase) GetFileBySha(ctx kratosx.Context, sha string) (*File, error) {
	file, err := u.repo.GetFileBySha(ctx, sha)
	if err != nil {
		return nil, err
	}
	file.Src = u.factory.FileSrc(file.Src)
	return file, nil
}

func (u *UseCase) PageFile(ctx kratosx.Context, in *PageFileRequest) ([]*File, uint32, error) {
	list, total, err := u.repo.PageFile(ctx, in)
	if err != nil {
		return nil, 0, errors.DatabaseFormat(err.Error())
	}
	for ind, item := range list {
		list[ind].Src = u.factory.FileSrc(item.Src)
	}
	return list, total, nil
}

// UpdateFile 修改文件名称
func (u *UseCase) UpdateFile(ctx kratosx.Context, file *File) error {
	if err := u.repo.UpdateFile(ctx, file); err != nil {
		return errors.DatabaseFormat(err.Error())
	}
	return nil
}

// DeleteFiles 删除文件
func (u *UseCase) DeleteFiles(ctx kratosx.Context, pid uint32, ids []uint32) error {
	if err := u.repo.DeleteFiles(ctx, pid, ids); err != nil {
		return errors.DatabaseFormat(err.Error())
	}
	return nil
}

// PrepareUploadFile 预上传文件
func (u *UseCase) PrepareUploadFile(ctx kratosx.Context, in *PrepareUploadFileRequest) (*PrepareUploadFileReply, error) {
	if in.DirectoryPath == "" && in.DirectoryId == 0 {
		return nil, errors.Params()
	}

	var err error
	var directory *Directory
	if in.DirectoryPath != "" {
		paths := strings.Split(in.DirectoryPath, "/")
		directory, err = u.repo.GetDirectoryByPaths(ctx, in.App, paths)
	} else {
		directory, err = u.repo.GetDirectoryByID(ctx, in.DirectoryId)
	}
	if err != nil {
		return nil, errors.DatabaseFormat(err.Error())
	}

	file, err := u.repo.GetFileBySha(ctx, in.Sha)
	if err == nil {
		// 触发秒传
		if file.Status == consts.STATUS_COMPLETED {
			if err := u.repo.CopyFile(ctx, file, directory.ID, in.Name); err != nil {
				return nil, errors.UploadFileFormat(err.Error())
			}
			return &PrepareUploadFileReply{
				Uploaded: proto.Bool(true),
				Src:      proto.String(u.factory.FileSrc(file.Src)),
				Sha:      proto.String(file.Sha),
			}, nil
		}

		var chunks []int
		if file.ChunkCount > 1 && file.UploadID != nil {
			store, err := u.factory.Store(ctx)
			if err != nil {
				return nil, errors.InitStoreFormat(err.Error())
			}
			chunk, err := store.NewPutChunkByUploadID(file.Src, *file.UploadID)
			if err != nil {
				return nil, errors.ChunkUpload()
			}
			chunks = chunk.UploadedChunkIndex()
		}
		return &PrepareUploadFileReply{
			Uploaded:     proto.Bool(false),
			UploadId:     file.UploadID,
			ChunkSize:    proto.Uint32(uint32(u.factory.MaxChunkSize())),
			ChunkCount:   proto.Uint32(file.ChunkCount),
			UploadChunks: chunks,
			Sha:          proto.String(file.Sha),
		}, nil
	}

	// 检查文件大小
	if err := u.factory.CheckSize(int64(in.Size)); err != nil {
		return nil, err
	}

	// 获取文件类型
	fileType := u.factory.GetType(in.Name)

	// 检查文件后缀
	if err := u.factory.CheckType(fileType); err != nil {
		return nil, err
	}

	// 构建文件对象
	file = &File{
		DirectoryID: directory.ID,
		Size:        in.Size,
		Sha:         in.Sha,
		Name:        in.Name,
		Src:         u.factory.StoreKey(in.Sha, fileType),
		Status:      consts.STATUS_PROGRESS,
		Storage:     u.factory.Storage(),
		Type:        fileType,
		UploadID:    proto.String(uuid.NewString()),
		ChunkCount:  1,
	}

	// 判断是否需要切片
	if u.factory.MaxSingularSize() < int64(in.Size) {
		store, err := u.factory.Store(ctx)
		if err != nil {
			return nil, err
		}

		pc, err := store.NewPutChunk(file.Src)
		if err != nil {
			return nil, errors.ChunkUpload()
		}
		file.UploadID = proto.String(pc.UploadID())
		file.ChunkCount = uint32(u.factory.ChunkCount(int64(in.Size)))
	}

	if err := u.repo.AddFile(ctx, file); err != nil {
		return nil, errors.Database()
	}

	return &PrepareUploadFileReply{
		Uploaded:     proto.Bool(false),
		UploadId:     file.UploadID,
		ChunkSize:    proto.Uint32(uint32(u.factory.MaxChunkSize())),
		ChunkCount:   proto.Uint32(file.ChunkCount),
		UploadChunks: []int{},
	}, nil
}

func (u *UseCase) UploadFile(ctx kratosx.Context, in *UploadFileRequest) (*UploadFileReply, error) {
	file, err := u.repo.GetFileByUploadID(ctx, in.UploadId)
	if err != nil {
		return nil, errors.UploadFileFormat("上传id不存在")
	}
	if file.Status == consts.STATUS_COMPLETED {
		return nil, errors.UploadFileFormat("请勿重复上传")
	}

	store, err := u.factory.Store(ctx)
	if err != nil {
		return nil, err
	}

	// 直接上传
	if file.ChunkCount == 1 {
		if err := store.PutBytes(file.Src, in.Data); err != nil {
			return nil, errors.UploadFileFormat(err.Error())
		}
		if err := u.repo.UpdateFileSuccess(ctx, file.ID); err != nil {
			return nil, errors.UploadFileFormat(err.Error())
		}
		return &UploadFileReply{
			Src: u.factory.FileSrc(file.Src),
			Sha: file.Sha,
		}, nil
	}

	// 切片上传
	chunk, err := store.NewPutChunkByUploadID(file.Src, in.UploadId)
	if err != nil {
		return nil, errors.ChunkUploadFormat(err.Error())
	}

	if err := chunk.AppendBytes(in.Data, int(in.Index)); err != nil {
		return nil, errors.ChunkUploadFormat(err.Error())
	}

	u.rw.Lock()
	if u.muiOnce[in.UploadId] == nil {
		u.muiOnce[in.UploadId] = &sync.Once{}
	}
	u.rw.Unlock()

	if chunk.ChunkCount() == int(file.ChunkCount) {
		u.rw.RLock()
		if u.muiOnce[in.UploadId] != nil {
			u.muiOnce[in.UploadId].Do(func() {
				_ = chunk.Complete()
				_ = u.repo.UpdateFileSuccess(ctx, file.ID)
			})
		}
		delete(u.muiOnce, in.UploadId)
		u.rw.RUnlock()
	}

	return &UploadFileReply{
		Src: u.factory.FileSrc(file.Src),
		Sha: file.Sha,
	}, nil
}
