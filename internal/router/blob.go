package router

import (
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	thttp "github.com/go-kratos/kratos/v2/transport/http"

	"github.com/limes-cloud/resource/api/errors"
	pb "github.com/limes-cloud/resource/api/file/v1"
	"github.com/limes-cloud/resource/internal/pkg/image"
	"github.com/limes-cloud/resource/internal/service"
)

type ResponseWriterWrapper struct {
	body   *bytes.Buffer
	header http.Header
	code   int
}

func NewWriter() *ResponseWriterWrapper {
	return &ResponseWriterWrapper{body: bytes.NewBufferString(""), header: make(http.Header)}
}

func (w *ResponseWriterWrapper) Header() http.Header {
	return w.header
}

func (w *ResponseWriterWrapper) WriteHeader(statusCode int) {
	w.code = statusCode
}

func (w *ResponseWriterWrapper) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

func (w *ResponseWriterWrapper) WriteString(s string) (int, error) {
	return w.body.WriteString(s)
}

func SrcBlob(srv *service.FileService) thttp.HandlerFunc {
	return func(ctx thttp.Context) error {
		var in pb.GetFileRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		if err := ctx.BindVars(&in); err != nil {
			return err
		}

		blw := NewWriter()

		fs := http.FileServer(http.Dir(srv.Config().Storage.LocalDir))
		fs = http.StripPrefix(srv.Config().Storage.ServerPath, fs)
		fs.ServeHTTP(blw, ctx.Request())

		// 处理图片裁剪
		cType := blw.header.Get("Content-Type")
		if strings.Contains(cType, "image/") && in.Width > 0 && in.Height > 0 {
			blw.header.Del("Content-Length")
			tp := strings.Split(cType, "/")[1]
			rb := blw.body.Bytes()
			if img, err := image.New(tp, rb); err == nil {
				if in.Mode == "" {
					in.Mode = image.AspectFill
				}
				if nrb, err := img.Resize(int(in.Width), int(in.Height), in.Mode); err == nil {
					blw.body = bytes.NewBuffer(nrb)
					blw.header.Set("Content-Length", strconv.Itoa(len(nrb)))
				}
			}
		}

		// 处理返回
		header := ctx.Response().Header()
		for key := range blw.header {
			header.Set(key, blw.header.Get(key))
		}

		if in.Download {
			fn := in.Src
			if in.SaveName != "" {
				fn = in.SaveName + filepath.Ext(in.Src)
			}
			header.Set("Content-Type", "application/octet-stream")
			header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fn))
		}

		ctx.Response().WriteHeader(blw.code)
		if _, err := ctx.Response().Write(blw.body.Bytes()); err != nil {
			return errors.System()
		}

		return nil
	}
}

// func SrcBlob(srv *service.FileService) http.HandlerFunc {
//	return func(ctx http.Context) error {
//		var in pb.GetFileRequest
//		if err := ctx.BindQuery(&in); err != nil {
//			return err
//		}
//		if err := ctx.BindVars(&in); err != nil {
//			return err
//		}
//
//		h := ctx.Middleware(func(ctx context.Context, req any) (any, error) {
//			return srv.GetFile(ctx, req.(*pb.GetFileRequest))
//		})
//		out, err := h(ctx, &in)
//		if err != nil {
//			return err
//		}
//		reply := out.(*pb.GetFileReply)
//		header := ctx.Response().Header()
//		header.Set("Content-Length", fmt.Sprint(len(reply.Data)))
//		if in.IsRange {
//			header.Set("Content-Range", fmt.Sprintf("bytes %d-%d", in.Start, in.End))
//		}
//
//		return ctx.Blob(200, reply.Mime, reply.Data)
//	}
// }
