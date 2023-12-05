package router

import (
	"context"
	"resource/internal/handler"
	"resource/internal/types"

	"github.com/go-kratos/kratos/v2/transport/http"
)

func Register(hs *http.Server, srv *handler.Handler) {
	cr := hs.Route("/")
	cr.GET("/resource/v1/static/{src}", func(ctx http.Context) error {
		var in types.GetFileRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		if err := ctx.BindVars(&in); err != nil {
			return err
		}
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.GetFile(ctx, req.(*types.GetFileRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*types.GetFileResponse)
		return ctx.Blob(200, reply.Mime, reply.Data)
	})
}
