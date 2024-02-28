package initiator

import (
	"context"

	"github.com/limes-cloud/kratosx"

	"github.com/limes-cloud/resource/internal/config"
	"github.com/limes-cloud/resource/internal/initiator/migrate"
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

	// 自动迁移
	migrate.Run(ctx)

	return nil
}
