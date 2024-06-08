package service

import (
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	thttp "github.com/go-kratos/kratos/v2/transport/http"

	"github.com/limes-cloud/resource/api/resource/errors"
	pb "github.com/limes-cloud/resource/api/resource/file/v1"
	"github.com/limes-cloud/resource/internal/pkg/image"
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

func (s *FileService) LocalPath(next http.Handler, src string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = src
		next.ServeHTTP(w, r)
	})
}

func (s *FileService) SrcBlob() thttp.HandlerFunc {
	return func(ctx thttp.Context) error {
		var req pb.StaticFileRequest
		if err := ctx.BindQuery(&req); err != nil {
			return err
		}
		if err := ctx.BindVars(&req); err != nil {
			return err
		}

		if err := s.uc.VerifyURL(req.Src, req.Expire, req.Sign); err != nil {
			return err
		}

		blw := NewWriter()
		fs := http.FileServer(http.Dir(s.conf.Storage.LocalDir))
		fs = s.LocalPath(fs, req.Src)
		fs.ServeHTTP(blw, ctx.Request())

		// http.Redirect(w, r, "https://taadis.com", http.StatusMovedPermanently)

		// 处理图片裁剪
		cType := blw.header.Get("Content-Type")
		if strings.Contains(cType, "image/") && req.Width > 0 && req.Height > 0 {
			blw.header.Del("Content-Length")
			tp := strings.Split(cType, "/")[1]
			rb := blw.body.Bytes()
			if img, err := image.New(tp, rb); err == nil {
				if req.Mode == "" {
					req.Mode = image.AspectFill
				}
				if nrb, err := img.Resize(int(req.Width), int(req.Height), req.Mode); err == nil {
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

		if req.Download {
			fn := req.Src
			if req.SaveName != "" {
				fn = req.SaveName + filepath.Ext(req.Src)
			}
			header.Set("Content-Type", "application/octet-stream")
			header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fn))
		}

		ctx.Response().WriteHeader(blw.code)
		if _, err := ctx.Response().Write(blw.body.Bytes()); err != nil {
			return errors.SystemError()
		}

		return nil
	}
}
