package app

import (
	"context"

	reserrors "github.com/limes-cloud/resource/api/errors"
	"github.com/limes-cloud/resource/internal/core"
)

// currentApp 从请求上下文获取调用方应用标识（X-App-Key 头解析，见 kratosx Context.AppKey）。
// 各资源数据按 app 隔离，应用标识不随请求体传输，统一由上下文提供。
func currentApp(ctx context.Context) (string, error) {
	app := core.MustContext(ctx).AppKey()
	if app == "" {
		return "", reserrors.ParamsError("缺少应用标识（X-App-Key）")
	}
	return app, nil
}