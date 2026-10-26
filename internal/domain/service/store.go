package service

import (
	"fmt"
	"strings"

	"github.com/limes-cloud/kratosx/pkg/value"

	"github.com/limes-cloud/resource/api/errors"
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/repository"
	"github.com/limes-cloud/resource/internal/pkg/provider"
	"github.com/limes-cloud/resource/internal/types"
)

// Store 存储后端领域服务。
type Store struct {
	repo repository.Store // 存储后端仓储（含进程内客户端 registry）
}

// NewStore 构造存储后端服务
func NewStore(repo repository.Store) *Store {
	return &Store{repo: repo}
}

// ListStore 分页查询存储后端（不含 sk 明文）
func (s *Store) ListStore(ctx core.Context, req *types.ListStoreRequest) ([]*entity.Store, uint32, error) {
	list, total, err := s.repo.ListStore(ctx, req)
	if err != nil {
		ctx.Logger().Warnw("msg", "list store error", "err", err.Error())
		return nil, 0, errors.ListError()
	}
	return list, total, nil
}

// GetStore 按主键查询存储后端（不含 sk 明文）
func (s *Store) GetStore(ctx core.Context, id uint32) (*entity.Store, error) {
	ent, err := s.repo.GetStore(ctx, id)
	if err != nil {
		ctx.Logger().Warnw("msg", "get store error", "id", id, "err", err.Error())
		return nil, errors.GetError()
	}
	return ent, nil
}

// CreateStore 新建存储后端（endpoint 归一化与跨字段校验在 app 层完成）。
func (s *Store) CreateStore(ctx core.Context, ent *entity.Store) (uint32, error) {
	id, err := s.repo.CreateStore(ctx, ent)
	if err != nil {
		ctx.Logger().Warnw("msg", "create store error", "keyword", ent.Keyword, "err", err.Error())
		if isDuplicate(err) {
			return 0, errors.DuplicateError("存储后端已存在（keyword 或 endpoint+bucket 重复）")
		}
		return 0, errors.CreateError()
	}

	return id, nil
}

// UpdateStore 局部更新存储后端：读当前行→叠加传入字段→归一化+跨字段校验→保存。
func (s *Store) UpdateStore(ctx core.Context, req *types.UpdateStoreRequest) error {
	ent, err := s.repo.GetStore(ctx, req.Id)
	if err != nil {
		ctx.Logger().Warnw("msg", "get store error", "id", req.Id, "err", err.Error())
		return errors.GetError()
	}

	if req.Provider != nil {
		ent.Provider = *req.Provider
	}
	if req.Name != nil {
		ent.Name = *req.Name
	}
	if req.Endpoint != nil {
		ent.Endpoint = *req.Endpoint
	}
	if req.Region != nil {
		ent.Region = *req.Region
	}
	if req.Bucket != nil {
		ent.Bucket = *req.Bucket
	}
	if req.AK != nil {
		ent.AK = *req.AK
	}
	// sk 含 * = 掩码回显（真实 sk 不含 *）：当成不修改，避免表单带掩码回填把密钥写坏
	if req.SK != nil && *req.SK != "" && !strings.Contains(*req.SK, "*") {
		ent.SK = *req.SK
	}
	if req.PublicBaseUrl != nil {
		ent.PublicBaseUrl = *req.PublicBaseUrl
	}
	if req.CdnAuthType != nil {
		ent.CdnAuthType = *req.CdnAuthType
	}
	if req.CdnAuthKey != nil {
		ent.CdnAuthKey = req.CdnAuthKey
	}
	if req.Status != nil {
		ent.Status = value.Pointer(uint8(*req.Status))
	}
	if req.Description != nil {
		ent.Description = req.Description
	}
	ent.UpdatedBy = req.UpdatedBy

	endpoint, err := provider.CanonicalEndpoint(ent.Provider, ent.Endpoint)
	if err != nil {
		return errors.ParamsError(err.Error())
	}
	ent.Endpoint = endpoint
	ent.PublicBaseUrl = provider.CanonicalBaseURL(ent.PublicBaseUrl)
	if err := ent.CheckCDN(); err != nil {
		return errors.ParamsError(err.Error())
	}

	if err := s.repo.UpdateStore(ctx, ent); err != nil {
		ctx.Logger().Warnw("msg", "update store error", "id", ent.Id, "err", err.Error())
		if isDuplicate(err) {
			return errors.DuplicateError("存储后端已存在（keyword 或 endpoint+bucket 重复）")
		}
		return errors.UpdateError()
	}

	return nil
}

// ListProvider 返回代码里已注册的厂商清单。
func (s *Store) ListProvider(_ core.Context) []provider.Provider {
	return provider.List()
}

// ensureStoreKeyword 校验 keyword 指向一个已加载的存储后端
func ensureStoreKeyword(ctx core.Context, repo repository.Store, keyword, field string) error {
	if _, ok := repo.GetClient(keyword); !ok {
		return fmt.Errorf("%s 指向不存在的 store.keyword: %s", field, keyword)
	}
	return nil
}
