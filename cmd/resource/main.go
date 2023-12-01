package main

import (
	"os"
	"resource/internal/handler"
	"resource/router"

	v1 "resource/api/v1"
	srcConf "resource/config"

	"github.com/limes-cloud/kratos"
	"github.com/limes-cloud/kratos/config"
	"github.com/limes-cloud/kratos/config/file"
	"github.com/limes-cloud/kratos/log"
	"github.com/limes-cloud/kratos/middleware/tracing"
	"github.com/limes-cloud/kratos/transport/grpc"
	"github.com/limes-cloud/kratos/transport/http"
	_ "go.uber.org/automaxprocs"
)

var (
	Name  string
	id, _ = os.Hostname()
)

func main() {
	app := kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Metadata(map[string]string{}),
		kratos.Config(file.NewSource("config/config.yaml")),
		kratos.RegistrarServer(RegisterServer),
		kratos.LoggerWith(kratos.LogField{
			"id":    id,
			"name":  Name,
			"trace": tracing.TraceID(),
			"span":  tracing.SpanID(),
		}),
	)

	if err := app.Run(); err != nil {
		log.Errorf("run service fail: %v", err)
	}
}

func RegisterServer(hs *http.Server, gs *grpc.Server, c config.Config) {
	conf := &srcConf.Config{}
	if err := c.ScanKey("file", conf); err != nil {
		panic("business config format error:" + err.Error())
	}

	srv := handler.New(conf)

	// 自定义路由
	router.Register(hs, srv)

	v1.RegisterServiceHTTPServer(hs, srv)
	v1.RegisterServiceServer(gs, srv)
}
