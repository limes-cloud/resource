package store

import (
	"context"
	"errors"

	"github.com/limes-cloud/kratosx"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/limes-cloud/resource/internal/conf"
	"github.com/limes-cloud/resource/internal/infra/store/channel/aliyun"
	"github.com/limes-cloud/resource/internal/infra/store/channel/baidu"
	"github.com/limes-cloud/resource/internal/infra/store/channel/local"
	"github.com/limes-cloud/resource/internal/infra/store/channel/tencent"
	"github.com/limes-cloud/resource/internal/infra/store/config"
	"github.com/limes-cloud/resource/internal/infra/store/types"
)

func initStore(st *conf.Storage) (types.Store, error) {
	ctx := kratosx.MustContext(context.Background())
	cfg := &config.Config{
		AntiTheft: st.AntiTheft,
		Keyword:   st.Keyword,
		Endpoint:  st.Endpoint,
		Id:        st.Id,
		Secret:    st.Secret,
		Bucket:    st.Bucket,
		LocalDir:  st.LocalDir,
		DB: ctx.DB().Session(&gorm.Session{
			Logger: logger.Default.LogMode(logger.Silent),
		}),
		Cache:           ctx.Redis(),
		TemporaryExpire: st.TemporaryExpire,
		ServerURL:       st.ServerURL,
	}

	var (
		err   error
		store types.Store
	)
	switch st.Type {
	case conf.STORE_ALIYUN:
		store, err = aliyun.New(cfg)
	case conf.STORE_TENCENT:
		store, err = tencent.New(cfg)
	case conf.STORE_BAIDU:
		store, err = baidu.New(cfg)
	case conf.STORE_LOCAL:
		store, err = local.New(cfg)
	default:
		err = errors.New("not support storage:" + st.Type)
	}

	return store, err
}

type st struct {
	e types.Store            // 导出文件存储地址
	d types.Store            // 默认存储地址
	b map[string]types.Store // 存储桶
}

func (s st) GetDefaultStore() types.Store {
	return s.d
}

func (s st) GetStore(key string) (types.Store, error) {
	si, ok := s.b[key]
	if !ok {
		return nil, errors.New("not exist store " + key)
	}
	return si, nil
}

func (s st) GetExportStore() types.Store {
	return s.e
}

func NewStore(cfs *conf.Config) types.Stores {
	if len(cfs.Storages) == 0 {
		panic("must set storage")
	}
	var ins = st{b: map[string]types.Store{}}
	for i, storage := range cfs.Storages {
		si, err := initStore(storage)
		if err != nil {
			panic("存储器初始化失败" + err.Error())
		}
		if i == 0 {
			ins.d = si
		}
		if storage.IsExporter {
			ts := *storage
			ts.ServerURL = cfs.Export.ServerURL
			si, err := initStore(&ts)
			if err != nil {
				panic("存储器初始化失败" + err.Error())
			}
			ins.e = si
		}
		ins.b[storage.Keyword] = si
	}
	return ins
}

func NewExportStore(cfs *conf.Config) types.Store {
	for _, storage := range cfs.Storages {
		if !storage.IsExporter {
			continue
		}

		ins, err := initStore(storage)
		if err != nil {
			panic("存储器初始化失败" + err.Error())
		}

		return ins
	}
	panic("must set export storage")
}
