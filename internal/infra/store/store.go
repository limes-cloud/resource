package store

import (
	"context"
	"errors"

	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/infra/store/channel/aliyun"
	"github.com/limes-cloud/resource/internal/infra/store/channel/baidu"
	"github.com/limes-cloud/resource/internal/infra/store/channel/local"
	"github.com/limes-cloud/resource/internal/infra/store/channel/tencent"
	"github.com/limes-cloud/resource/internal/infra/store/types"
)

const (
	STORE_ALIYUN  = "aliyun"
	STORE_TENCENT = "tencent"
	STORE_BAIDU   = "baidu"
	STORE_LOCAL   = "local"
)

func NewStore(keyword ...string) (types.Store, error) {
	var (
		err   error
		store types.Store
		ctx   = core.MustContext(context.Background())
		cs    = ctx.Config().Storage
	)
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

	switch conf.Type {
	case STORE_ALIYUN:
		store, err = aliyun.New(ctx, conf)
	case STORE_TENCENT:
		store, err = tencent.New(ctx, conf)
	case STORE_BAIDU:
		store, err = baidu.New(ctx, conf)
	case STORE_LOCAL:
		store, err = local.New(ctx, conf)
	default:
		err = errors.New("not support storage:" + conf.Type)
	}
	if err != nil {
		return nil, err
	}

	return store, nil
}

func NewExportStore() (types.Store, error) {
	var (
		err   error
		store types.Store
		ctx   = core.MustContext(context.Background())
		cs    = ctx.Config().Storage
	)
	if len(cs) == 0 {
		return nil, errors.New("not found store")
	}

	conf := cs[0]
	for _, item := range cs {
		if item.IsExporter {
			conf = item
		}
	}

	switch conf.Type {
	case STORE_ALIYUN:
		store, err = aliyun.New(ctx, conf)
	case STORE_TENCENT:
		store, err = tencent.New(ctx, conf)
	case STORE_BAIDU:
		store, err = baidu.New(ctx, conf)
	case STORE_LOCAL:
		store, err = local.New(ctx, conf)
	default:
		err = errors.New("not support storage:" + conf.Type)
	}
	if err != nil {
		return nil, err
	}

	return store, nil
}
