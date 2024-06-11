package service

import (
	"context"
	"errors"

	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/limes-cloud/kratosx"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/limes-cloud/resource/internal/conf"
	"github.com/limes-cloud/resource/internal/pkg/store"
	"github.com/limes-cloud/resource/internal/pkg/store/aliyun"
	"github.com/limes-cloud/resource/internal/pkg/store/local"
	"github.com/limes-cloud/resource/internal/pkg/store/tencent"
)

type registryFunc func(c *conf.Config, hs *http.Server, gs *grpc.Server)

var registries []registryFunc

var (
	globalStore       store.Store
	globalExportStore store.Store
)

func register(fn registryFunc) {
	registries = append(registries, fn)
}

func New(c *conf.Config, hs *http.Server, gs *grpc.Server) {
	st, err := NewStore(c)
	if err != nil {
		panic(err)
	}

	lst, err := NewExportStore(c)
	if err != nil {
		panic(err)
	}

	globalStore = st
	globalExportStore = lst

	for _, registry := range registries {
		registry(c, hs, gs)
	}
}

func NewStore(conf *conf.Config) (store.Store, error) {
	ctx := kratosx.MustContext(context.Background())
	cfg := &store.Config{
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

	switch conf.Storage.Type {
	case store.STORE_ALIYUN:
		return aliyun.New(cfg)
	case store.STORE_TENCENT:
		return tencent.New(cfg)
	case store.STORE_LOCAL:
		return local.New(cfg)
	default:
		return nil, errors.New("not support storage:" + conf.Storage.Type)
	}
}

func NewExportStore(conf *conf.Config) (store.Store, error) {
	ctx := kratosx.MustContext(context.Background())
	cfg := &store.Config{
		Secret:   conf.Storage.Secret,
		LocalDir: conf.Storage.LocalDir,
		DB: ctx.DB().Session(&gorm.Session{
			Logger: logger.Default.LogMode(logger.Silent),
		}),
		Cache:           ctx.Redis(),
		TemporaryExpire: conf.Storage.TemporaryExpire,
		ServerURL:       conf.Export.ServerURL,
	}

	return local.New(cfg)
}
