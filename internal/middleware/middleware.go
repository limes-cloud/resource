package middleware

import (
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/limes-cloud/resource/internal/middleware/auth"
)

func Middleware() []middleware.Middleware {
	return []middleware.Middleware{
		auth.Parse(),
	}
}
