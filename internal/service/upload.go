package service

import (
	"context"
	"io"

	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/limes-cloud/kratosx/pkg/valx"

	"github.com/limes-cloud/resource/api/resource/errors"
	pb "github.com/limes-cloud/resource/api/resource/file/v1"
)

func (s *FileService) Upload() http.HandlerFunc {
	return func(ctx http.Context) error {
		var in pb.UploadFileRequest

		in.UploadId = ctx.Request().FormValue("uploadId")
		in.Index = valx.ToUint32(ctx.Request().FormValue("index"))
		file, _, err := ctx.Request().FormFile("data")
		if err != nil {
			return errors.UploadFileError(err.Error())
		}

		in.Data, err = io.ReadAll(file)
		if err != nil {
			return errors.UploadFileError(err.Error())
		}
		if in.UploadId == "" || int(in.Index) <= 0 || len(in.Data) == 0 {
			return errors.ParamsError()
		}

		h := ctx.Middleware(func(ctx context.Context, req any) (any, error) {
			return s.UploadFile(ctx, req.(*pb.UploadFileRequest))
		})

		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*pb.UploadFileReply)
		return ctx.Result(200, reply)
	}
}
