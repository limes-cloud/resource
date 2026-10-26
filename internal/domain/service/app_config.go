package service

import (
	"github.com/limes-cloud/kratosx/pkg/value"

	"github.com/limes-cloud/resource/api/errors"
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/repository"
	"github.com/limes-cloud/resource/internal/types"
)

// AppConfig 应用配置领域服务。
type AppConfig struct {
	repo      repository.AppConfig // 应用配置仓储（Redis 缓存 + 延迟双删）
	storeRepo repository.Store     // 存储后端仓储（校验 store_keyword 指向存在的桶）
}

// NewAppConfig 构造应用配置服务
func NewAppConfig(repo repository.AppConfig, storeRepo repository.Store) *AppConfig {
	return &AppConfig{repo: repo, storeRepo: storeRepo}
}

// GetAppConfig 按 app 读取应用配置（缺行即应用未配置）。
func (s *AppConfig) GetAppConfig(ctx core.Context, app string) (*entity.AppConfig, error) {
	ent, err := s.repo.GetAppConfig(ctx, app)
	if err != nil {
		ctx.Logger().Warnw("msg", "get app config error", "key", app, "err", err.Error())
		if isNotFound(err) {
			return nil, errors.AppNotConfiguredError()
		}
		return nil, errors.GetError()
	}
	return ent, nil
}

// ListAppConfig 分页查询应用配置
func (s *AppConfig) ListAppConfig(ctx core.Context, req *types.ListAppConfigRequest) ([]*entity.AppConfig, uint32, error) {
	list, total, err := s.repo.ListAppConfig(ctx, req)
	if err != nil {
		ctx.Logger().Warnw("msg", "list app config error", "err", err.Error())
		return nil, 0, errors.ListError()
	}
	return list, total, nil
}

// CreateAppConfig 创建应用配置（参数校验在 app 层 / proto 完成，此处只校验 store 引用存在性）。
func (s *AppConfig) CreateAppConfig(ctx core.Context, ent *entity.AppConfig) (uint32, error) {
	if err := s.checkStoreRefs(ctx, ent); err != nil {
		return 0, err
	}

	id, err := s.repo.CreateAppConfig(ctx, ent)
	if err != nil {
		ctx.Logger().Warnw("msg", "create app config error", "key", ent.App, "err", err.Error())
		if isDuplicate(err) {
			return 0, errors.DuplicateError("应用配置已存在")
		}
		return 0, errors.CreateError()
	}
	return id, nil
}

// UpdateAppConfig 局部更新：传了才改（零值可写列在 entity 用指针），直接 Updates；只校验本次要改的 store 引用。
func (s *AppConfig) UpdateAppConfig(ctx core.Context, req *types.UpdateAppConfigRequest) error {
	ent := &entity.AppConfig{
		App:                req.App,
		StoreKeyword:       value.Value(req.StoreKeyword),
		ExportStoreKeyword: value.Value(req.ExportStoreKeyword),
		DefaultAccept:      req.DefaultAccept,
		DefaultMaxSize:     value.Value(req.DefaultMaxSize),
		TempUrlTtl:         value.Value(req.TempUrlTtl),
		ExportTtl:          value.Value(req.ExportTtl),
		Quota:              req.Quota,
		UpdatedBy:          req.UpdatedBy,
	}
	if req.Status != nil {
		ent.Status = value.Pointer(uint8(*req.Status))
	}

	if req.StoreKeyword != nil {
		if err := ensureStoreKeyword(ctx, s.storeRepo, ent.StoreKeyword, "store_keyword"); err != nil {
			ctx.Logger().Warnw("msg", "app config store keyword not found", "key", ent.App, "err", err.Error())
			return errors.ParamsError()
		}
	}
	if req.ExportStoreKeyword != nil {
		if err := ensureStoreKeyword(ctx, s.storeRepo, ent.ExportStoreKeyword, "export_store_keyword"); err != nil {
			ctx.Logger().Warnw("msg", "app config export store keyword not found", "key", ent.App, "err", err.Error())
			return errors.ParamsError()
		}
	}

	if err := s.repo.UpdateAppConfig(ctx, ent); err != nil {
		ctx.Logger().Warnw("msg", "update app config error", "app", ent.App, "err", err.Error())
		return errors.UpdateError()
	}
	return nil
}

// RecalcUsage 按明细重算该 app 的 used_bytes（对账纠漂移），返回重算后用量。
func (s *AppConfig) RecalcUsage(ctx core.Context, app string) (uint64, error) {
	used, err := s.repo.RecalcUsage(ctx, app)
	if err != nil {
		ctx.Logger().Warnw("msg", "recalc app usage error", "app", app, "err", err.Error())
		return 0, errors.UpdateError()
	}
	return used, nil
}

// checkStoreRefs 校验 store_keyword / export_store_keyword 指向已加载的存储后端（引用存在性，需要仓储，非参数校验）。
func (s *AppConfig) checkStoreRefs(ctx core.Context, ent *entity.AppConfig) error {
	if err := ensureStoreKeyword(ctx, s.storeRepo, ent.StoreKeyword, "store_keyword"); err != nil {
		ctx.Logger().Warnw("msg", "app config store keyword not found", "key", ent.App, "err", err.Error())
		return errors.ParamsError()
	}
	if err := ensureStoreKeyword(ctx, s.storeRepo, ent.ExportStoreKeyword, "export_store_keyword"); err != nil {
		ctx.Logger().Warnw("msg", "app config export store keyword not found", "key", ent.App, "err", err.Error())
		return errors.ParamsError()
	}
	return nil
}
