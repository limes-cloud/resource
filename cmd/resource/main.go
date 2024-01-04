package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	configure "github.com/limes-cloud/configure/client"
	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/config"
	_ "go.uber.org/automaxprocs"

	v1 "github.com/limes-cloud/resource/api/v1"
	resourceconf "github.com/limes-cloud/resource/config"
	"github.com/limes-cloud/resource/internal/handler"
	"github.com/limes-cloud/resource/internal/initiator"
	"github.com/limes-cloud/resource/pkg/pt"
	"github.com/limes-cloud/resource/router"
)

func main() {
	app := kratosx.New(
		kratosx.Config(configure.NewFromEnv()),
		kratosx.RegistrarServer(RegisterServer),
		kratosx.Options(kratos.AfterStart(func(ctx context.Context) error {
			kt := kratosx.MustContext(ctx)
			pt.ArtFont(fmt.Sprintf("Hello %s !", kt.Name()))
			return nil
		})),
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

	// 初始化逻辑
	ior := initiator.New(conf)
	if err := ior.Run(); err != nil {
		panic("initiator error:" + err.Error())
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
