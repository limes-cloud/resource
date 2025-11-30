package auth

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/metadata"

	"github.com/go-kratos/kratos/v2/transport"
	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/manager/api/authorize"

	"github.com/go-kratos/kratos/v2/middleware"
)

const (
	TokenKey = "x-md-global-token"
)

type infoKey struct{}

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
			// 从header获取token
			header, ok := transport.FromServerContext(c)
			if !ok {
				return handler(c, req)
			}

			token := header.RequestHeader().Get("Authorization")
			token = strings.TrimPrefix(token, "Bearer ")
			if token == "" {
				return handler(c, req)
			}

			// 设置到md上，grpc带上去
			md, ok := metadata.FromServerContext(c)
			if !ok {
				return handler(c, req)
			}
			md.Set(TokenKey, token)

			ctx := kratosx.MustContext(c)
			conn, err := ctx.GrpcConn("Manager")
			if err != nil {
				return nil, err
			}

			client := authorize.NewAuthorizeClient(conn)
			reply, err := client.ParseToken(ctx, &authorize.ParseTokenRequest{})
			if err != nil {
				return nil, err
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
