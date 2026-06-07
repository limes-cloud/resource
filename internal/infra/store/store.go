package store

import (
	"context"
	"errors"
	"strings"

	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/repository"
)

func createStore(ctx core.Context, conf *core.Storage) (repository.Store, error) {
	return newS3(ctx, conf)
}

func NewStore(keyword ...string) (repository.Store, error) {
	ctx := core.MustContext(context.Background())
	cs := ctx.Config().Storage
	if len(cs) == 0 {
		return nil, errors.New("not found store")
	}

	conf := cs[0]
	if len(keyword) != 0 {
		for _, item := range cs {
			if item.Keyword == keyword[0] {
				conf = item
			}
		}
	}
	return createStore(ctx, conf)
}

func NewExportStore() (repository.Store, error) {
	ctx := core.MustContext(context.Background())
	cs := ctx.Config().Storage
	if len(cs) == 0 {
		return nil, errors.New("not found store")
	}

	conf := cs[0]
	for _, item := range cs {
		if item.IsExporter {
			conf = item
		}
	}
	return createStore(ctx, conf)
}

func NewStoreByKey(key string) (repository.Store, error) {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) == 2 {
		return NewStore(parts[0])
	}
	return NewStore()
}
