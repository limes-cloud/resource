package main

import (
	"context"

	"github.com/limes-cloud/manager/api/authorize"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/limes-cloud/kratosx"
	_ "go.uber.org/automaxprocs"

	"github.com/limes-cloud/resource/api/errors"
	"github.com/limes-cloud/resource/internal/app"
	"github.com/limes-cloud/resource/internal/core"
	initmenu "github.com/limes-cloud/resource/internal/initiator/menu"
	initreclaim "github.com/limes-cloud/resource/internal/initiator/reclaim"
)

func main() {
	srv := core.InitApp(
		kratosx.WithRegistrarServer(app.Register),
		kratosx.WithValidateErrHook(func(ctx context.Context, err error) error {
			c := core.MustContext(ctx)
			c.Logger().Warnw("msg", "params validate error", "err", err)
			return errors.ParamsError()
		}),
		kratosx.WithMiddleware(authorize.AuthMiddleware()),
	)

	ctx := core.MustContext(context.Background())
	initmenu.Register(ctx)
	initreclaim.Register(ctx)

	if err := srv.App().Run(); err != nil {
		log.Fatal(err)
	}
}
