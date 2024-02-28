package router

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport/http"

	pb "github.com/limes-cloud/resource/api/file/v1"
	"github.com/limes-cloud/resource/internal/service"
)

func SrcBlob(srv *service.FileService) http.HandlerFunc {
	return func(ctx http.Context) error {
		var in pb.GetFileRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		if err := ctx.BindVars(&in); err != nil {
			return err
		}
		h := ctx.Middleware(func(ctx context.Context, req any) (any, error) {
			return srv.GetFile(ctx, req.(*pb.GetFileRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*pb.GetFileReply)
		return ctx.Blob(200, reply.Mime, reply.Data)
	}
}
