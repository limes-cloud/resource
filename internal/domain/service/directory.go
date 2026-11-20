package service

import (
	"strings"

	"github.com/limes-cloud/kratosx/model"
	"github.com/limes-cloud/resource/api/errors"
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/repository"
	"github.com/limes-cloud/resource/internal/types"
)

// Directory 目录领域服务（物化路径，无 parent_id）
type Directory struct {
	repo      repository.Directory // 目录仓储
	appConfig repository.AppConfig // 应用配置仓储（accept / max_size 快照的来源）
}

// NewDirectory 构造目录服务
func NewDirectory(repo repository.Directory, appConfig repository.AppConfig) *Directory {
	return &Directory{repo: repo, appConfig: appConfig}
}

// GetDirectory 按主键查询目录（必须带 app 限定，防跨应用读取）
func (s *Directory) GetDirectory(ctx core.Context, app string, id uint32) (*entity.Directory, error) {
	ent, err := s.repo.GetDirectory(ctx, app, id)
	if err != nil {
		ctx.Logger().Warnw("msg", "get directory error", "id", id, "err", err.Error())
		if isNotFound(err) {
			return nil, errors.GetError()
		}
		return nil, errors.GetError()
	}
	return ent, nil
}

// GetDirectoryById 按主键查询目录（不限定 app，用于从目录派生 app 等读场景）
func (s *Directory) GetDirectoryById(ctx core.Context, id uint32) (*entity.Directory, error) {
	ent, err := s.repo.GetDirectoryById(ctx, id)
	if err != nil {
		ctx.Logger().Warnw("msg", "get directory by id error", "id", id, "err", err.Error())
		if isNotFound(err) {
			return nil, errors.GetError()
		}
		return nil, errors.GetError()
	}
	return ent, nil
}

// ListDirectory 全量取回某 app 的目录（按 path 升序，内存建树在应用层做）
func (s *Directory) ListDirectory(ctx core.Context, app string) ([]*entity.Directory, error) {
	list, err := s.repo.ListDirectory(ctx, app)
	if err != nil {
		ctx.Logger().Warnw("msg", "list directory error", "key", app, "err", err.Error())
		return nil, errors.ListError()
	}
	return list, nil
}

// CreateDirectory 按完整路径创建目录（策略从 app_config 抄快照）。
func (s *Directory) CreateDirectory(ctx core.Context, req *types.CreateDirectoryRequest) (*entity.Directory, error) {
	segs, err := s.parsePath(req.Path)
	if err != nil {
		ctx.Logger().Warnw("msg", "parse directory path error", "key", req.App, "err", err.Error())
		return nil, err
	}

	cfg, err := s.appConfig.GetAppConfig(ctx, req.App)
	if err != nil {
		ctx.Logger().Warnw("msg", "get app config error", "key", req.App, "err", err.Error())
		if isNotFound(err) {
			return nil, errors.AppNotConfiguredError()
		}
		return nil, errors.GetError()
	}

	return s.ensurePath(ctx, req.App, segs, cfg)
}

// UpdateDirectory 更新目录：改名 / 移动 / 改策略。
func (s *Directory) UpdateDirectory(ctx core.Context, req *types.UpdateDirectoryRequest) error {
	cur, err := s.repo.GetDirectory(ctx, req.App, req.Id)
	if err != nil {
		ctx.Logger().Warnw("msg", "get directory error", "id", req.Id, "err", err.Error())
		return errors.UpdateError()
	}

	node := &entity.Directory{BaseModel: model.BaseModel{Id: cur.Id}, App: req.App}
	var (
		newPath string
		moving  bool
	)

	if req.Path != nil {
		next := normalizePath(*req.Path)
		if next == "" {
			return errors.ParamsError()
		}
		if next != cur.Path {
			if err := s.checkMove(ctx, req.App, cur, next); err != nil {
				return err
			}
			newPath, moving = next, true
		}
	}

	if req.Accept != nil {
		node.Accept = req.Accept
	}
	if req.MaxSize != nil {
		node.MaxSize = *req.MaxSize
	}

	if moving {
		node.Name = lastSegment(newPath) // 节点自己的 name 跟新路径走；子树各行 name 不变
	}

	// accept 用指针区分"未传"与"设为空"；三者都未动即无变更
	if !moving && node.Name == "" && node.Accept == nil && node.MaxSize == 0 {
		return nil
	}
	node.UpdatedBy = req.UpdatedBy

	// 移子树(重写 path) 与 节点本身(name/accept/max_size) 必须同一事务，否则半成功会留下 path 与 name 错位
	return ctx.Transaction(func(ctx core.Context) error {
		if moving {
			if err := s.repo.MoveDirectory(ctx, &types.MoveDirectoryParams{App: req.App, Id: cur.Id, NewPath: newPath, UpdatedBy: req.UpdatedBy}); err != nil {
				ctx.Logger().Warnw("msg", "move directory error", "id", cur.Id, "err", err.Error())
				if isDuplicate(err) {
					return errors.PathConflictError()
				}
				return errors.UpdateError()
			}
		}
		if err := s.repo.UpdateDirectory(ctx, node); err != nil {
			ctx.Logger().Warnw("msg", "update directory error", "id", cur.Id, "err", err.Error())
			return errors.UpdateError()
		}
		return nil
	})
}

// EnsurePath 按路径逐级确保目录存在，返回最深一级的目录（上传入口用）。
func (s *Directory) EnsurePath(ctx core.Context, app, path string, cfg *entity.AppConfig) (*entity.Directory, error) {
	segs, err := s.parsePath(path)
	if err != nil {
		return nil, err
	}
	return s.ensurePath(ctx, app, segs, cfg)
}

// parsePath 解析并校验路径（限深、逐段 name 值域）
func (s *Directory) parsePath(path string) ([]string, error) {
	segs := splitPath(path)
	if len(segs) == 0 {
		// 根目录没有行，空路径直接拒绝
		return nil, errors.ParamsError()
	}
	if len(segs) > entity.MaxDepth {
		return nil, errors.PathTooDeepError()
	}
	for _, seg := range segs {
		if err := entity.ValidateName(seg); err != nil {
			return nil, errors.NameInvalidError()
		}
	}
	return segs, nil
}

// ensurePath 逐级确保目录存在
func (s *Directory) ensurePath(ctx core.Context, app string, segs []string, cfg *entity.AppConfig) (*entity.Directory, error) {
	// 新行的策略从 app_config 抄快照；app_config 的默认值在其写入边界已校验，这里直接抄
	// 前缀集合在内存按 "/" 切出，一次查询定位最深已存在层
	prefixes := make([]string, 0, len(segs))
	for i := range segs {
		prefixes = append(prefixes, "/"+strings.Join(segs[:i+1], "/"))
	}
	existing, err := s.repo.ListDirectoryInPaths(ctx, app, prefixes)
	if err != nil {
		ctx.Logger().Warnw("msg", "list directory in paths error", "key", app, "err", err.Error())
		return nil, errors.ListError()
	}
	found := make(map[string]*entity.Directory, len(existing))
	for _, ent := range existing {
		found[ent.Path] = ent
	}

	// 从那一层往下逐层 INSERT IGNORE，撞唯一键后重查
	var deepest *entity.Directory
	for i, path := range prefixes {
		if ent, ok := found[path]; ok {
			deepest = ent
			continue
		}
		ent, err := s.repo.CreateDirectoryIgnore(ctx, &entity.Directory{
			App:     app,
			Name:    segs[i],
			Path:    path,
			Accept:  cfg.DefaultAccept,
			MaxSize: cfg.DefaultMaxSize,
		})
		if err != nil {
			ctx.Logger().Warnw("msg", "create directory ignore error", "key", app, "path", path, "err", err.Error())
			return nil, errors.CreateError()
		}
		deepest = ent
	}
	return deepest, nil
}

// checkMove 校验移动的三条硬约束：路径值域、目标已存在则拒绝、路径前缀防环
func (s *Directory) checkMove(ctx core.Context, app string, cur *entity.Directory, newPath string) error {
	segs := splitPath(newPath)
	if len(segs) > entity.MaxDepth {
		return errors.PathTooDeepError()
	}
	for _, seg := range segs {
		if err := entity.ValidateName(seg); err != nil {
			return errors.NameInvalidError()
		}
	}

	// 防环：新父级的 path 是自己的 path 的前缀 → 拒绝。
	if strings.HasPrefix(newPath, cur.Path+"/") {
		return errors.PathConflictError()
	}

	// 目标已存在则拒绝移动，不重试、不合并、不自动改名
	if exist, err := s.repo.GetDirectoryByPath(ctx, app, newPath); err == nil && exist.Id != cur.Id {
		return errors.PathConflictError()
	}

	// 父目录必须存在（顶层目录的父是根，没有行）
	if parent := parentPath(newPath); parent != "" {
		if _, err := s.repo.GetDirectoryByPath(ctx, app, parent); err != nil {
			if isNotFound(err) {
				return errors.ParamsError()
			}
			return errors.GetError()
		}
	}
	return nil
}

// splitPath 按 "/" 切分路径并丢弃空段（容忍首尾与重复斜杠）
func splitPath(p string) []string {
	parts := strings.Split(strings.TrimSpace(p), "/")
	segs := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			segs = append(segs, part)
		}
	}
	return segs
}

// normalizePath 归一化路径为 /a/b 形态；空路径返回 ""
func normalizePath(p string) string {
	segs := splitPath(p)
	if len(segs) == 0 {
		return ""
	}
	return "/" + strings.Join(segs, "/")
}

// parentPath 取父路径；顶层目录返回 ""（根的父级是空串，不存在行）
func parentPath(p string) string {
	idx := strings.LastIndex(p, "/")
	if idx <= 0 {
		return ""
	}
	return p[:idx]
}

// lastSegment 取路径最后一段
func lastSegment(p string) string {
	idx := strings.LastIndex(p, "/")
	if idx < 0 {
		return p
	}
	return p[idx+1:]
}
