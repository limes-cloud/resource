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
	"github.com/limes-cloud/kratosx/pkg/util"
	_ "go.uber.org/automaxprocs"

	resourceconf "github.com/limes-cloud/resource/internal/config"
	"github.com/limes-cloud/resource/internal/initiator"
	"github.com/limes-cloud/resource/internal/router"
	"github.com/limes-cloud/resource/internal/service"
)

func main() {
	app := kratosx.New(
		kratosx.Config(configure.NewFromEnv()),
		kratosx.RegistrarServer(RegisterServer),
		kratosx.Options(kratos.AfterStart(func(ctx context.Context) error {
			kt := kratosx.MustContext(ctx)
			util.PrintArtFont(fmt.Sprintf("Hello %s !", kt.Name()))
			return nil
		})),
	)

	if err := app.Run(); err != nil {
		log.Fatal("run service fail", err)
	}
}

func RegisterServer(c config.Config, hs *http.Server, gs *grpc.Server) {
	// 初始化并监听配置变更
	conf := &resourceconf.Config{}
	c.ScanWatch("file", func(value config.Value) {
		if err := value.Scan(conf); err != nil {
			panic("file config format error:" + err.Error())
		}
	})

	// 初始化逻辑
	ior := initiator.New(conf)
	if err := ior.Run(); err != nil {
		panic("initiator error:" + err.Error())
	}

	// 注册服务
	fileSrv := service.New(conf, hs, gs)
	router.Register(hs, fileSrv)
}
