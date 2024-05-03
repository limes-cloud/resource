package file

import (
	"github.com/limes-cloud/kratosx"
)

type Repo interface {
	AddDirectory(ctx kratosx.Context, in *Directory) (uint32, error)
	GetDirectoryByID(ctx kratosx.Context, id uint32) (*Directory, error)
	GetDirectoryByName(ctx kratosx.Context, id uint32, name string) (*Directory, error)
	GetDirectoryByPaths(ctx kratosx.Context, app string, paths []string) (*Directory, error)
	UpdateDirectory(ctx kratosx.Context, in *Directory) error
	DeleteDirectory(ctx kratosx.Context, id uint32) error
	AllDirectoryByParentID(ctx kratosx.Context, pid uint32, app string) ([]*Directory, error)
	DirectoryCountByParentID(ctx kratosx.Context, id uint32) (int64, error)

	CopyFile(ctx kratosx.Context, src *File, did uint32, name string) error

	// FileCountByName(ctx kratosx.Context, did uint32, name string) (int64, error)
	FileCountByDirectoryID(ctx kratosx.Context, id uint32) (int64, error)

	GetFileByID(ctx kratosx.Context, id uint32) (*File, error)
	GetFileBySha(ctx kratosx.Context, keyword string) (*File, error)
	GetFileByUploadID(ctx kratosx.Context, uid string) (*File, error)
	PageFile(ctx kratosx.Context, req *PageFileRequest) ([]*File, uint32, error)
	AddFile(ctx kratosx.Context, c *File) error
	UpdateFile(ctx kratosx.Context, file *File) error
	UpdateFileSuccess(ctx kratosx.Context, id uint32) error
	DeleteFile(ctx kratosx.Context, id uint32) error
	DeleteFiles(ctx kratosx.Context, pid uint32, ids []uint32) error
}
