package store

import (
	"context"
	"errors"
	"github.com/limes-cloud/resource/internal/core"
	"sync"

	"github.com/limes-cloud/resource/internal/infra/store/channel/aliyun"
	"github.com/limes-cloud/resource/internal/infra/store/channel/baidu"
	"github.com/limes-cloud/resource/internal/infra/store/channel/local"
	"github.com/limes-cloud/resource/internal/infra/store/channel/tencent"
	"github.com/limes-cloud/resource/internal/infra/store/types"
)

var (
	storeIns  types.Store
	storeOnce sync.Once
)

const (
	STORE_ALIYUN  = "aliyun"
	STORE_TENCENT = "tencent"
	STORE_BAIDU   = "baidu"
	STORE_LOCAL   = "local"
)

func NewStore() types.Store {
	storeOnce.Do(func() {
		var (
			err   error
			store types.Store
			ctx   = core.MustContext(context.Background())
			cs    = ctx.Config().Storage
		)
		switch cs.Type {
		case STORE_ALIYUN:
			store, err = aliyun.New(ctx)
		case STORE_TENCENT:
			store, err = tencent.New(ctx)
		case STORE_BAIDU:
			store, err = baidu.New(ctx)
		case STORE_LOCAL:
			store, err = local.New(ctx)
		default:
			err = errors.New("not support storage:" + cs.Type)
		}
		if err != nil {
			panic(err)
		}
		storeIns = store
	})

	return storeIns
}
