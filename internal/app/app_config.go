package app

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/limes-cloud/kratosx/pkg/value"

	appcfgapi "github.com/limes-cloud/resource/api/app_config"
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/service"
	"github.com/limes-cloud/resource/internal/infra/dbs"
	"github.com/limes-cloud/resource/internal/types"
)

// appConfigHandler 实现 app_config.AppConfigServiceHTTPServer / AppConfigServiceServer。
type appConfigHandler struct {
	appcfgapi.UnimplementedAppConfigServiceServer
	s *service.AppConfig
}

// init 自注册 AppConfigService。
func init() {
	register(func(hs *http.Server, gs *grpc.Server) {
		h := &appConfigHandler{s: service.NewAppConfig(dbs.NewAppConfig(), dbs.NewStore())}
		appcfgapi.RegisterAppConfigServiceHTTPServer(hs, h)
		appcfgapi.RegisterAppConfigServiceServer(gs, h)
	})
}

// ListAppConfig 分页查询应用配置（app 取自请求上下文）。
func (h *appConfigHandler) ListAppConfig(ctx context.Context, req *appcfgapi.ListAppConfigRequest) (*appcfgapi.ListAppConfigResponse, error) {
	app, err := currentApp(ctx)
	if err != nil {
		return nil, err
	}
	list, total, err := h.s.ListAppConfig(core.MustContext(ctx), &types.ListAppConfigRequest{
		PageRequest: types.PageRequest{Page: req.Page, PageSize: req.PageSize},
		App:         value.Pointer(app),
		Status:      req.Status,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*appcfgapi.AppConfigView, 0, len(list))
	for _, e := range list {
		out = append(out, toAppConfigView(e))
	}
	return &appcfgapi.ListAppConfigResponse{List: out, Total: total}, nil
}

// GetAppConfig 按 app 读取应用配置（app 取自请求上下文）。
func (h *appConfigHandler) GetAppConfig(ctx context.Context, req *appcfgapi.GetAppConfigRequest) (*appcfgapi.GetAppConfigResponse, error) {
	app, err := currentApp(ctx)
	if err != nil {
		return nil, err
	}
	ent, err := h.s.GetAppConfig(core.MustContext(ctx), app)
	if err != nil {
		return nil, err
	}
	return &appcfgapi.GetAppConfigResponse{AppConfig: toAppConfigView(ent)}, nil
}

// CreateAppConfig 创建应用配置（app 取自请求上下文）。
func (h *appConfigHandler) CreateAppConfig(ctx context.Context, req *appcfgapi.CreateAppConfigRequest) (*appcfgapi.CreateAppConfigResponse, error) {
	app, err := currentApp(ctx)
	if err != nil {
		return nil, err
	}
	ent := &entity.AppConfig{
		App:                app,
		StoreKeyword:       req.StoreKeyword,
		ExportStoreKeyword: req.ExportStoreKeyword,
		DefaultAccept:      value.Pointer(req.DefaultAccept),
		DefaultMaxSize:     req.DefaultMaxSize,
		TempUrlTtl:         req.TempUrlTtl,
		ExportTtl:          req.ExportTtl,
		Quota:              value.Pointer(req.Quota),
		Status:             value.Pointer(uint8(req.Status)),
		UpdatedBy:          uint32(core.MustContext(ctx).Auth().UserId),
	}
	id, err := h.s.CreateAppConfig(core.MustContext(ctx), ent)
	if err != nil {
		return nil, err
	}
	return &appcfgapi.CreateAppConfigResponse{Id: uint32(id)}, nil
}

// UpdateAppConfig 局部更新应用配置（app 为 WHERE 条件，取自请求上下文；传了才改）。
func (h *appConfigHandler) UpdateAppConfig(ctx context.Context, req *appcfgapi.UpdateAppConfigRequest) (*appcfgapi.UpdateAppConfigResponse, error) {
	app, err := currentApp(ctx)
	if err != nil {
		return nil, err
	}
	c := core.MustContext(ctx)
	if err := h.s.UpdateAppConfig(c, &types.UpdateAppConfigRequest{
		App:                app,
		StoreKeyword:       req.StoreKeyword,
		ExportStoreKeyword: req.ExportStoreKeyword,
		DefaultAccept:      req.DefaultAccept,
		DefaultMaxSize:     req.DefaultMaxSize,
		TempUrlTtl:         req.TempUrlTtl,
		ExportTtl:          req.ExportTtl,
		Quota:              req.Quota,
		Status:             req.Status,
		UpdatedBy:          uint32(c.Auth().UserId),
	}); err != nil {
		return nil, err
	}
	return &appcfgapi.UpdateAppConfigResponse{App: app}, nil
}

// RecalcAppConfigUsage 按明细重算已用量（对账，app 取自请求上下文）。
func (h *appConfigHandler) RecalcAppConfigUsage(ctx context.Context, _ *appcfgapi.RecalcAppConfigUsageRequest) (*appcfgapi.RecalcAppConfigUsageResponse, error) {
	app, err := currentApp(ctx)
	if err != nil {
		return nil, err
	}
	used, err := h.s.RecalcUsage(core.MustContext(ctx), app)
	if err != nil {
		return nil, err
	}
	return &appcfgapi.RecalcAppConfigUsageResponse{App: app, UsedBytes: used}, nil
}

// toAppConfigView 领域 AppConfig → proto AppConfigView。
func toAppConfigView(e *entity.AppConfig) *appcfgapi.AppConfigView {
	return &appcfgapi.AppConfigView{
		Id:                 uint32(e.Id),
		App:                e.App,
		StoreKeyword:       e.StoreKeyword,
		ExportStoreKeyword: e.ExportStoreKeyword,
		DefaultAccept:      value.Value(e.DefaultAccept),
		DefaultMaxSize:     e.DefaultMaxSize,
		TempUrlTtl:         e.TempUrlTtl,
		ExportTtl:          e.ExportTtl,
		Quota:              value.Value(e.Quota),
		UsedBytes:          e.UsedBytes,
		Status:             uint32(value.Value(e.Status)),
		UpdatedBy:          uint32(e.UpdatedBy),
		CreatedAt:          uint32(e.CreatedAt),
		UpdatedAt:          uint32(e.UpdatedAt),
	}
}
