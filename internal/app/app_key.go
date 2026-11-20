package app

import (
	"context"
	"strings"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	reserrors "github.com/limes-cloud/resource/api/errors"
	"github.com/limes-cloud/resource/internal/core"
)

// currentApp 获取本次请求的目标应用标识。
// X-App-Key 始终表示登录应用；manager 可通过 ?app=xxx 显式指定目标应用，其他应用传入的 app 会被忽略。
func currentApp(ctx context.Context) (string, error) {
	loginApp := strings.TrimSpace(core.MustContext(ctx).AppKey())
	if loginApp == "" {
		return "", reserrors.ParamsError("缺少应用标识（X-App-Key）")
	}
	if loginApp != "manager" {
		return loginApp, nil
	}
	if r, ok := khttp.RequestFromServerContext(ctx); ok {
		if app := strings.TrimSpace(r.URL.Query().Get("app")); app != "" {
			return app, nil
		}
	}
	return loginApp, nil
}
