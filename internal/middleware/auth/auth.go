package auth

import (
	"context"
	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/limes-cloud/kratosx"
	km "github.com/limes-cloud/kratosx/middleware"
	"github.com/limes-cloud/manager/api/auth"
	"github.com/limes-cloud/manager/api/errors"
)

type infoKey struct {
}

type Info struct {
	UserId   uint32
	DeptId   uint32
	TenantId uint32
	Username string
}

// Parse 鉴权
func Parse() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(c context.Context, req any) (any, error) {

			md, ok := metadata.FromServerContext(c)
			if !ok {
				return handler(c, req)
			}
			token := md.Get(km.TokenKey)
			if token == "" {
				return handler(c, req)
			}

			ctx := kratosx.MustContext(c)
			conn, err := ctx.GrpcConn("Manager")
			if err != nil {
				return nil, err
			}

			client := auth.NewAuthClient(conn)
			reply, err := client.ParseToken(ctx, &auth.ParseTokenRequest{
				Token: token,
			})
			if err != nil {
				return nil, errors.ForbiddenError()
			}

			cctx := context.WithValue(ctx.Ctx(), infoKey{}, &Info{
				UserId:   reply.UserId,
				DeptId:   reply.DeptId,
				TenantId: reply.TenantId,
				Username: reply.Username,
			})
			return handler(cctx, req)
		}
	}
}

func Get(ctx context.Context) *Info {
	v, _ := ctx.Value(infoKey{}).(*Info)
	return v
}
