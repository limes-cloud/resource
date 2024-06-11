package service

import (
	"fmt"
	"net/http"
	"path/filepath"

	thttp "github.com/go-kratos/kratos/v2/transport/http"

	"github.com/limes-cloud/resource/api/resource/errors"
	pb "github.com/limes-cloud/resource/api/resource/file/v1"
)

func (s *ExportService) LocalPath(next http.Handler, src string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = src
		next.ServeHTTP(w, r)
	})
}

func (s *ExportService) Download() thttp.HandlerFunc {
	return func(ctx thttp.Context) error {
		var req pb.DownloadFileRequest
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
		fs := http.FileServer(http.Dir(s.conf.Export.LocalDir))
		fs = s.LocalPath(fs, req.Src)
		fs.ServeHTTP(blw, ctx.Request())

		header := ctx.Response().Header()
		fn := req.Src
		if req.SaveName != "" {
			fn = req.SaveName + filepath.Ext(req.Src)
		}
		header.Set("Content-Type", "application/octet-stream")
		header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fn))

		ctx.Response().WriteHeader(blw.code)
		if _, err := ctx.Response().Write(blw.body.Bytes()); err != nil {
			return errors.SystemError()
		}

		return nil
	}
}
