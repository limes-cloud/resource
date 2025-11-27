package app

import (
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
)

type registerFunc func(hs *http.Server, gs *grpc.Server)

var registerList []registerFunc

func register(fn registerFunc) {
	registerList = append(registerList, fn)
}

func Register(hs *http.Server, gs *grpc.Server) {
	for _, registry := range registerList {
		registry(hs, gs)
	}
}
