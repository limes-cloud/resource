package store

import (
	"context"
	"errors"

	"github.com/limes-cloud/resource/internal/infra/store/types"

	"github.com/limes-cloud/kratosx"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/limes-cloud/resource/internal/conf"
	"github.com/limes-cloud/resource/internal/infra/store/channel/aliyun"
	"github.com/limes-cloud/resource/internal/infra/store/channel/local"
	"github.com/limes-cloud/resource/internal/infra/store/channel/tencent"
	"github.com/limes-cloud/resource/internal/infra/store/config"
)

const (
	STORE_ALIYUN  = "aliyun"
	STORE_TENCENT = "tencent"
	STORE_LOCAL   = "local"
)

func NewStore(conf *conf.Config) types.Store {
	ctx := kratosx.MustContext(context.Background())
	cfg := &config.Config{
		Endpoint: conf.Storage.Endpoint,
		Id:       conf.Storage.Id,
		Secret:   conf.Storage.Secret,
		Bucket:   conf.Storage.Bucket,
		LocalDir: conf.Storage.LocalDir,
		DB: ctx.DB().Session(&gorm.Session{
			Logger: logger.Default.LogMode(logger.Silent),
		}),
		Cache:           ctx.Redis(),
		TemporaryExpire: conf.Storage.TemporaryExpire,
		ServerURL:       conf.Storage.ServerURL,
	}

	var (
		err   error
		store types.Store
	)
	switch conf.Storage.Type {
	case STORE_ALIYUN:
		store, err = aliyun.New(cfg)
	case STORE_TENCENT:
		store, err = tencent.New(cfg)
	case STORE_LOCAL:
		store, err = local.New(cfg)
	default:
		err = errors.New("not support storage:" + conf.Storage.Type)
	}
	if err != nil {
		panic(err)
	}
	return store
}

func NewExportStore(conf *conf.Config) types.Store {
	ctx := kratosx.MustContext(context.Background())
	cfg := &config.Config{
		Secret:   conf.Storage.Secret,
		LocalDir: conf.Storage.LocalDir,
		DB: ctx.DB().Session(&gorm.Session{
			Logger: logger.Default.LogMode(logger.Silent),
		}),
		Cache:           ctx.Redis(),
		TemporaryExpire: conf.Storage.TemporaryExpire,
		ServerURL:       conf.Export.ServerURL,
	}

	store, err := local.New(cfg)
	if err != nil {
		panic(err)
	}
	return store
}
