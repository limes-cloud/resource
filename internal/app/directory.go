package app

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/limes-cloud/kratosx/pkg/value"

	dirapi "github.com/limes-cloud/resource/api/directory"
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/service"
	"github.com/limes-cloud/resource/internal/infra/dbs"
	initreclaim "github.com/limes-cloud/resource/internal/initiator/reclaim"
	"github.com/limes-cloud/resource/internal/types"
)

// directoryHandler 实现 directory.DirectoryServiceHTTPServer / DirectoryServiceServer。
type directoryHandler struct {
	dirapi.UnimplementedDirectoryServiceServer
	dir     *service.Directory
	recycle *service.Recycle
}

// init 自注册 DirectoryService。
func init() {
	register(func(hs *http.Server, gs *grpc.Server) {
		dir := service.NewDirectory(dbs.NewDirectory(), dbs.NewAppConfig())
		recycle := service.NewRecycle(dbs.NewFile(), dbs.NewObject(), dbs.NewStore(), dbs.NewDirectory(), dbs.NewAppConfig())
		h := &directoryHandler{dir: dir, recycle: recycle}
		dirapi.RegisterDirectoryServiceHTTPServer(hs, h)
		dirapi.RegisterDirectoryServiceServer(gs, h)
	})
}

// ListDirectory 全量取回某 app 的目录。
func (h *directoryHandler) ListDirectory(ctx context.Context, req *dirapi.ListDirectoryRequest) (*dirapi.ListDirectoryResponse, error) {
	app, err := currentApp(ctx)
	if err != nil {
		return nil, err
	}
	list, err := h.dir.ListDirectory(core.MustContext(ctx), app)
	if err != nil {
		return nil, err
	}
	out := make([]*dirapi.DirectoryView, 0, len(list))
	for _, e := range list {
		out = append(out, toDirectoryView(e))
	}
	return &dirapi.ListDirectoryResponse{List: out}, nil
}

// GetDirectory 按主键查询目录。
func (h *directoryHandler) GetDirectory(ctx context.Context, req *dirapi.GetDirectoryRequest) (*dirapi.GetDirectoryResponse, error) {
	app, err := currentApp(ctx)
	if err != nil {
		return nil, err
	}
	ent, err := h.dir.GetDirectory(core.MustContext(ctx), app, uint32(req.Id))
	if err != nil {
		return nil, err
	}
	return &dirapi.GetDirectoryResponse{Directory: toDirectoryView(ent)}, nil
}

// CreateDirectory 按完整路径创建目录。
func (h *directoryHandler) CreateDirectory(ctx context.Context, req *dirapi.CreateDirectoryRequest) (*dirapi.CreateDirectoryResponse, error) {
	app, err := currentApp(ctx)
	if err != nil {
		return nil, err
	}
	ent, err := h.dir.CreateDirectory(core.MustContext(ctx), &types.CreateDirectoryRequest{App: app, Path: req.Path})
	if err != nil {
		return nil, err
	}
	return &dirapi.CreateDirectoryResponse{Directory: toDirectoryView(ent)}, nil
}

// UpdateDirectory 改名 / 移动 / 改策略。
func (h *directoryHandler) UpdateDirectory(ctx context.Context, req *dirapi.UpdateDirectoryRequest) (*dirapi.UpdateDirectoryResponse, error) {
	// proto 字段为 optional（指针），直接透传：nil = 未传，非 nil = 要改（accept 可为 ""）
	app, err := currentApp(ctx)
	if err != nil {
		return nil, err
	}
	c := core.MustContext(ctx)
	d := &types.UpdateDirectoryRequest{
		App:       app,
		Id:        uint32(req.Id),
		Path:      req.Path,
		Accept:    req.Accept,
		MaxSize:   req.MaxSize,
		UpdatedBy: uint32(c.Auth().UserId),
	}
	if err := h.dir.UpdateDirectory(c, d); err != nil {
		return nil, err
	}
	return &dirapi.UpdateDirectoryResponse{Id: req.Id}, nil
}

// DeleteDirectory 删目录（软删：标记子树待删除 + 删目录树 + 归还配额；随后 Kick 后台立即物理回收，reclaim ticker 兜底补偿）。
func (h *directoryHandler) DeleteDirectory(ctx context.Context, req *dirapi.DeleteDirectoryRequest) (*dirapi.DeleteDirectoryResponse, error) {
	app, err := currentApp(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.recycle.DeleteDirectory(core.MustContext(ctx), app, uint32(req.Id)); err != nil {
		return nil, err
	}
	// 第一层：立即触发后台回收，不等定时 ticker（第二层补偿见 initiator/reclaim）
	initreclaim.Kick()
	return &dirapi.DeleteDirectoryResponse{Id: req.Id}, nil
}

// toDirectoryView 领域 Directory → proto DirectoryView。
func toDirectoryView(e *entity.Directory) *dirapi.DirectoryView {
	return &dirapi.DirectoryView{
		Id:        uint32(e.Id),
		App:       e.App,
		Name:      e.Name,
		Path:      e.Path,
		Accept:    value.Value(e.Accept),
		MaxSize:   e.MaxSize,
		UpdatedBy: uint32(e.UpdatedBy),
		CreatedAt: uint32(e.CreatedAt),
		UpdatedAt: uint32(e.UpdatedAt),
	}
}
