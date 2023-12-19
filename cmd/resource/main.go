package main

import (
	"log"

	configure "github.com/limes-cloud/configure/client"

	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/config"
	v1 "github.com/limes-cloud/resource/api/v1"
	resourceconf "github.com/limes-cloud/resource/config"
	"github.com/limes-cloud/resource/internal/handler"
	"github.com/limes-cloud/resource/router"
	_ "go.uber.org/automaxprocs"
)

func main() {
	app := kratosx.New(
		kratosx.Config(configure.NewFromEnv()),
		kratosx.RegistrarServer(RegisterServer),
	)

	if err := app.Run(); err != nil {
		log.Fatal("run service fail", err)
	}
}

func RegisterServer(c config.Config, hs *http.Server, gs *grpc.Server) {
	conf := &resourceconf.Config{}

	// 配置初始化
	if err := c.Value("file").Scan(conf); err != nil {
		panic("author config format error:" + err.Error())
	}

	// 监听服务
	c.Watch("file", func(value config.Value) {
		if err := value.Scan(conf); err != nil {
			log.Printf("business配置变更失败：%s", err.Error())
		}
	})

	srv := handler.New(conf)

	// 自定义路由
	router.Register(hs, srv)

	v1.RegisterServiceHTTPServer(hs, srv)
	v1.RegisterServiceServer(gs, srv)
}
