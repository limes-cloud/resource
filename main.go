package main

import (
	"context"

	"github.com/limes-cloud/kratosx/library"
	"github.com/limes-cloud/kratosx/library/db"
	"github.com/limes-cloud/manager/api/scope"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/resource/api/errors"
	"github.com/limes-cloud/resource/internal/app"
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/middleware"
	_ "go.uber.org/automaxprocs"
)

// bu
func main() {
	srv := core.InitApp(
		kratosx.WithRegistrarServer(app.Register),
		kratosx.WithValidateErrHook(func(ctx context.Context, err error) error {
			c := core.MustContext(ctx)
			c.Logger().Warnw("msg", "params validate error", "err", err)
			return errors.ParamsError()
		}),
		kratosx.WithLibraryOptions(
			library.WithDBOptions(db.WithHookScope(scope.Hook)),
		),
		kratosx.WithMiddleware(middleware.Middleware()...),
	)

	if err := srv.App().Run(); err != nil {
		log.Fatal(err)
	}
}
