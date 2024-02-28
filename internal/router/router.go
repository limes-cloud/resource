package router

import (
	"github.com/go-kratos/kratos/v2/transport/http"

	"github.com/limes-cloud/resource/internal/service"
)

func Register(hs *http.Server, fileSrv *service.FileService) {
	cr := hs.Route("/")
	cr.GET("/resource/v1/static/{src}", SrcBlob(fileSrv))
	cr.POST("/resource/v1/upload", Upload(fileSrv))
	cr.POST("/resource/client/v1/upload", Upload(fileSrv))
}
