package router

import (
	"context"
	"io"

	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/limes-cloud/kratosx/pkg/util"

	"github.com/limes-cloud/resource/api/errors"
	pb "github.com/limes-cloud/resource/api/file/v1"
	"github.com/limes-cloud/resource/internal/service"
)

func Upload(srv *service.FileService) http.HandlerFunc {
	return func(ctx http.Context) error {
		var in pb.UploadFileRequest
		// if err := ctx.Request().PostForm.Get()(&in); err != nil {
		//	return err
		// }
		in.UploadId = ctx.Request().FormValue("upload_id")
		in.Index = util.ToUint32(ctx.Request().FormValue("index"))
		file, _, err := ctx.Request().FormFile("data")
		if err != nil {
			return errors.UploadFileFormat(err.Error())
		}

		in.Data, err = io.ReadAll(file)
		if err != nil {
			return errors.UploadFileFormat(err.Error())
		}

		if in.UploadId == "" || int(in.Index) <= 0 || len(in.Data) == 0 {
			return errors.UploadFileFormat("参数缺失")
		}

		h := ctx.Middleware(func(ctx context.Context, req any) (any, error) {
			return srv.UploadFile(ctx, req.(*pb.UploadFileRequest))
		})

		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*pb.UploadFileReply)
		return ctx.Result(200, reply)
	}
}
