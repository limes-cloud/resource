package file

import (
	"github.com/limes-cloud/kratosx"

	"github.com/limes-cloud/resource/internal/pkg/store"
)

type Factory interface {
	Storage() string
	ChunkCount(size int64) int
	GetType(name string) string
	StoreKey(sha string, tp string) string
	CheckType(tp string) error
	CheckSize(size int64) error
	MaxSingularSize() int64
	MaxChunkSize() int64
	FileSrcFormat() string
	FileSrc(src string) string
	FileMime(body []byte) string
	Store(ctx kratosx.Context) (store.Store, error)
}
