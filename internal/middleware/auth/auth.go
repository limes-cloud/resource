package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-kratos/kratos/v2/middleware"
	khttp "github.com/go-kratos/kratos/v2/transport/http"

	"github.com/limes-cloud/manager/api/authorize"
	managererrors "github.com/limes-cloud/manager/api/errors"
)

// Strict rejects HTTP requests that have neither gateway-injected identity nor a token.
// Internal gRPC calls stay trusted and are only reachable from the private network.
func Strict() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			r, ok := khttp.RequestFromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}
			if r.Method == http.MethodOptions || isPublic(r.Method, r.URL.Path) {
				return handler(ctx, req)
			}

			if strings.TrimSpace(r.Header.Get(authorize.InfoKey)) != "" {
				return handler(ctx, req)
			}
			if token, ok := credential(r.Header.Get(authorize.TokenKey), r.Header.Get("Authorization")); ok {
				r.Header.Set(authorize.TokenKey, token)
				return handler(ctx, req)
			}
			return nil, managererrors.NotLoginError()
		}
	}
}

func isPublic(method, path string) bool {
	if method != http.MethodGet {
		return false
	}
	return path == "/resource/api/file/object-by-key" || path == "/resource/api/file/read-by-key"
}

func credential(token, authorization string) (string, bool) {
	if token = strings.TrimSpace(token); token != "" {
		return token, true
	}

	authorization = strings.TrimSpace(authorization)
	if authorization == "" {
		return "", false
	}
	parts := strings.Fields(authorization)
	if len(parts) == 1 && !strings.EqualFold(parts[0], "Bearer") {
		return parts[0], true
	}
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1], true
	}
	return "", false
}
