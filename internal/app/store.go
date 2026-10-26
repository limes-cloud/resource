package app

import (
	"context"
	"errors"
	"strings"

	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/limes-cloud/kratosx/pkg/value"

	reserrors "github.com/limes-cloud/resource/api/errors"
	storeapi "github.com/limes-cloud/resource/api/store"
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/service"
	"github.com/limes-cloud/resource/internal/infra/dbs"
	"github.com/limes-cloud/resource/internal/infra/store"
	"github.com/limes-cloud/resource/internal/pkg/provider"
	"github.com/limes-cloud/resource/internal/types"
)

// storeHandler 实现 store.StoreServiceHTTPServer / StoreServiceServer。
type storeHandler struct {
	storeapi.UnimplementedStoreServiceServer
	s *service.Store
}

// init 自注册 StoreService。
func init() {
	register(func(hs *http.Server, gs *grpc.Server) {
		h := &storeHandler{s: service.NewStore(dbs.NewStore())}
		storeapi.RegisterStoreServiceHTTPServer(hs, h)
		storeapi.RegisterStoreServiceServer(gs, h)
	})
}

// ListStore 分页查询存储后端（不含 sk）。
func (h *storeHandler) ListStore(ctx context.Context, req *storeapi.ListStoreRequest) (*storeapi.ListStoreResponse, error) {
	list, total, err := h.s.ListStore(core.MustContext(ctx), &types.ListStoreRequest{
		PageRequest: types.PageRequest{Page: req.Page, PageSize: req.PageSize},
		Keyword:     req.Keyword,
		Provider:    req.Provider,
		Status:      req.Status,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*storeapi.Store, 0, len(list))
	for _, e := range list {
		out = append(out, toStore(e))
	}
	return &storeapi.ListStoreResponse{List: out, Total: total}, nil
}

// GetStore 按主键查询存储后端。
func (h *storeHandler) GetStore(ctx context.Context, req *storeapi.GetStoreRequest) (*storeapi.GetStoreResponse, error) {
	ent, err := h.s.GetStore(core.MustContext(ctx), uint32(req.Id))
	if err != nil {
		return nil, err
	}
	return &storeapi.GetStoreResponse{Store: toStore(ent)}, nil
}

// CreateStore 新建存储后端。
func (h *storeHandler) CreateStore(ctx context.Context, req *storeapi.CreateStoreRequest) (*storeapi.CreateStoreResponse, error) {
	ent := &entity.Store{
		Keyword:       req.Keyword,
		Provider:      req.Provider,
		Name:          req.Name,
		Endpoint:      req.Endpoint,
		Region:        req.Region,
		Bucket:        req.Bucket,
		AK:            req.Ak,
		SK:            req.Sk,
		PublicBaseUrl: req.PublicBaseUrl,
		CdnAuthType:   req.CdnAuthType,
		CdnAuthKey:    req.CdnAuthKey,
		Status:        value.Pointer(uint8(req.Status)),
		Description:   req.Description,
		UpdatedBy:     uint32(core.MustContext(ctx).Auth().UserId),
	}
	if err := prepareStore(ent); err != nil {
		return nil, err
	}
	id, err := h.s.CreateStore(core.MustContext(ctx), ent)
	if err != nil {
		return nil, err
	}
	return &storeapi.CreateStoreResponse{Id: uint32(id)}, nil
}

// UpdateStore 局部更新存储后端（仅 id 必传，其余传了才改）。
func (h *storeHandler) UpdateStore(ctx context.Context, req *storeapi.UpdateStoreRequest) (*storeapi.UpdateStoreResponse, error) {
	c := core.MustContext(ctx)
	err := h.s.UpdateStore(c, &types.UpdateStoreRequest{
		Id:            uint32(req.Id),
		Provider:      req.Provider,
		Name:          req.Name,
		Endpoint:      req.Endpoint,
		Region:        req.Region,
		Bucket:        req.Bucket,
		AK:            req.Ak,
		SK:            req.Sk,
		PublicBaseUrl: req.PublicBaseUrl,
		CdnAuthType:   req.CdnAuthType,
		CdnAuthKey:    req.CdnAuthKey,
		Status:        req.Status,
		Description:   req.Description,
		UpdatedBy:     uint32(c.Auth().UserId),
	})
	if err != nil {
		return nil, err
	}
	return &storeapi.UpdateStoreResponse{Id: req.Id}, nil
}

// prepareStore 归一化 endpoint / public_base_url，并校验 proto 表达不了的跨字段自洽性（provider 白名单 + CDN 配置）。
func prepareStore(ent *entity.Store) error {
	endpoint, err := provider.CanonicalEndpoint(ent.Provider, ent.Endpoint)
	if err != nil {
		return reserrors.ParamsError(err.Error())
	}
	ent.Endpoint = endpoint
	ent.PublicBaseUrl = provider.CanonicalBaseURL(ent.PublicBaseUrl)
	if err := ent.CheckCDN(); err != nil {
		return reserrors.ParamsError(err.Error())
	}
	return nil
}

// CheckStore 测试存储配置连通性（不落库）：归一化后构造客户端并 HeadBucket 探测。
func (h *storeHandler) CheckStore(ctx context.Context, req *storeapi.CheckStoreRequest) (*storeapi.CheckStoreResponse, error) {
	// 已保存的 store 编辑测试：sk 为掩码/未改时回退用 DB 存的真实 sk（明文仅在服务端取），其余字段以入参为准
	if req.Id != nil && (req.Sk == "" || strings.Contains(req.Sk, "*")) {
		row, err := h.s.GetStore(core.MustContext(ctx), value.Value(req.Id))
		if err != nil {
			return &storeapi.CheckStoreResponse{Ok: false, Message: "查询存储配置失败：请确认该存储后端已保存"}, nil
		}
		req.Sk = row.SK
	}

	ent := &entity.Store{
		Provider:    req.Provider,
		Endpoint:    req.Endpoint,
		Region:      req.Region,
		Bucket:      req.Bucket,
		AK:          req.Ak,
		SK:          req.Sk,
		CdnAuthType: "none",
	}
	// 只做与探测相关的形态校验：endpoint 归一化。
	// 不套用 prepareStore（其要求 public_base_url 非空、CDN 自洽），那些与 HeadBucket 连通性无关。
	if endpoint, err := provider.CanonicalEndpoint(ent.Provider, ent.Endpoint); err != nil {
		return &storeapi.CheckStoreResponse{Ok: false, Message: err.Error()}, nil
	} else {
		ent.Endpoint = endpoint
	}

	client, err := store.NewClient(ent)
	if err != nil {
		return &storeapi.CheckStoreResponse{Ok: false, Message: "配置无法构造客户端: " + err.Error()}, nil
	}
	if err := client.Check(core.MustContext(ctx)); err != nil {
		switch {
		case errors.Is(err, store.ErrCheckAuthFailed):
			return &storeapi.CheckStoreResponse{Ok: false, Message: "认证失败：AccessKey / SecretKey 错误或缺少该桶权限"}, nil
		case errors.Is(err, store.ErrCheckBucketNotFound):
			return &storeapi.CheckStoreResponse{Ok: false, Message: "存储桶不存在或可见性受限"}, nil
		default:
			return &storeapi.CheckStoreResponse{Ok: false, Message: "无法连接：请检查域名 / 网络 / 区域配置（" + err.Error() + "）"}, nil
		}
	}
	return &storeapi.CheckStoreResponse{Ok: true, Message: "连接正常"}, nil
}

// ListProvider 返回代码里已注册的厂商清单。
func (h *storeHandler) ListProvider(ctx context.Context, _ *storeapi.ListProviderRequest) (*storeapi.ListProviderResponse, error) {
	list := h.s.ListProvider(core.MustContext(ctx))
	out := make([]*storeapi.Provider, 0, len(list))
	for _, p := range list {
		out = append(out, &storeapi.Provider{Value: p.Value, Label: p.Label, Implemented: p.Implemented})
	}
	return &storeapi.ListProviderResponse{List: out}, nil
}

// toStore 领域 Store → proto Store（ak 完整返回，sk 固定占位符回显，cdn_auth_key 不出）。
func toStore(e *entity.Store) *storeapi.Store {
	return &storeapi.Store{
		Id:            uint32(e.Id),
		Keyword:       e.Keyword,
		Provider:      e.Provider,
		Name:          e.Name,
		Endpoint:      e.Endpoint,
		Region:        e.Region,
		Bucket:        e.Bucket,
		Ak:            e.AK,
		Sk:            maskSecret(e.SK),
		PublicBaseUrl: e.PublicBaseUrl,
		CdnAuthType:   e.CdnAuthType,
		Status:        uint32(value.Value(e.Status)),
		Description:   value.Value(e.Description),
		UpdatedBy:     uint32(e.UpdatedBy),
		CreatedAt:     uint32(e.CreatedAt),
		UpdatedAt:     uint32(e.UpdatedAt),
	}
}

// maskSecret 固定占位符回显敏感串：有值即 "******"（不泄漏长度/字符），空值回空。
// 占位符含 * ，写入侧据此识别回填并忽略（见 service.UpdateStore）。
func maskSecret(s string) string {
	if s == "" {
		return ""
	}
	return "******"
}
