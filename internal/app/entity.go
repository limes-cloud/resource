package app

import (
	"context"
	"github.com/limes-cloud/resource/api/entity"
	"github.com/limes-cloud/resource/api/errors"
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/service"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/limes-cloud/kratosx/pkg/value"

	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
)

type Entity struct {
	entity.UnimplementedEntityServer
	srv *service.Entity
}

// NewEntity 初始化租户应用
func NewEntity() *Entity {
	return &Entity{
		srv: service.NewEntity(),
	}
}

// init 应用注册
func init() {
	register(func(hs *http.Server, gs *grpc.Server) {
		srv := NewEntity()
		entity.RegisterEntityHTTPServer(hs, srv)
		entity.RegisterEntityServer(gs, srv)
	})
}

// LoadEntity 载入全部实体信息
func (app *Entity) LoadEntity(c context.Context, _ *emptypb.Empty) (*entity.LoadEntityReply, error) {
	ctx := core.MustContext(c)

	// 调用服务
	list, err := app.srv.LoadEntity(ctx)
	if err != nil {
		return nil, err
	}

	// 处理返回数据
	reply := entity.LoadEntityReply{}
	if err := value.Transform(list, &reply.List); err != nil {
		ctx.Logger().Errorw("msg", "get entity resp transform error", "err", err)
		return nil, errors.TransformError()
	}
	return &reply, nil
}
