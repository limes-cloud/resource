package service

import (
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	filepb "github.com/limes-cloud/resource/api/file/v1"
	"github.com/limes-cloud/resource/internal/config"
)

func New(c *config.Config, hs *http.Server, gs *grpc.Server) *FileService {
	fileSrv := NewFile(c)
	filepb.RegisterServiceHTTPServer(hs, fileSrv)
	filepb.RegisterServiceServer(gs, fileSrv)

	// 自定义路由
	return fileSrv
}
