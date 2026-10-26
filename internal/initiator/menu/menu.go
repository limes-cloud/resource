package menu

import (
	"context"

	managerMenu "github.com/limes-cloud/manager/api/menu"
	"github.com/limes-cloud/resource/internal/core"
)

const registerModeAppend = "append"

func Register(ctx core.Context) {
	ctx.BeforeStart("register resource menu", func() {
		conn, err := ctx.GrpcConn("Manager")
		if err != nil {
			ctx.Logger().Errorw("msg", "get manager grpc conn error", "err", err.Error())
			return
		}

		reply, err := managerMenu.NewMenuClient(conn).RegisterMenu(context.Background(), &managerMenu.RegisterMenuRequest{
			AppKeyword: appKeyword,
			Mode:       registerModeAppend,
			Menus:      menus,
		})
		if err != nil {
			ctx.Logger().Errorw("msg", "register resource menu error", "err", err.Error())
			return
		}

		ctx.Logger().Infow(
			"msg", "register resource menu success",
			"created", reply.Created,
			"updated", reply.Updated,
			"deleted", reply.Deleted,
		)
	})
}
