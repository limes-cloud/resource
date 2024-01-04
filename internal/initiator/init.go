package initiator

import (
	"context"

	"github.com/limes-cloud/kratosx"

	"github.com/limes-cloud/resource/config"
	"github.com/limes-cloud/resource/internal/initiator/migrate"
	"github.com/limes-cloud/resource/pkg/pt"
)

type Initiator struct {
	conf *config.Config
}

func New(conf *config.Config) *Initiator {
	return &Initiator{
		conf: conf,
	}
}

// Run 执行系统初始化
func (a *Initiator) Run() error {
	ctx := kratosx.MustContext(context.Background())

	if migrate.IsInit(ctx) {
		pt.Cyan("already init server")
		return nil
	}

	// 自动迁移
	migrate.Init(ctx, a.conf)

	return nil
}
