package dbs

import (
	"context"
	"sync"

	"github.com/limes-cloud/kratosx"
	"github.com/limes-cloud/kratosx/pkg/cache"

	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/repository"
	"github.com/limes-cloud/resource/internal/infra/store"
	"github.com/limes-cloud/resource/internal/types"
)

const storeCacheKey = "store"

// Store 存储后端仓储实现（store 行走 pkg/cache：内存读 + Redis 跨节点同步）。
type Store struct {
	cache   *cache.Cache[string, *entity.Store]
	clients *sync.Map // keyword -> *clientMemo，按行指针记忆化复用已构造的客户端
}

// clientMemo 缓存某行构造出的客户端；row 变（本地 Put 或跨节点 Repair 换新指针）即重建。
type clientMemo struct {
	row    *entity.Store
	client repository.Client
}

var (
	storeIns  *Store
	storeOnce sync.Once
)

// NewStore 取存储后端仓储单例（首次调用注册 BeforeStart 钩子，启动前全量灌入 store 缓存）。
func NewStore() *Store {
	storeOnce.Do(func() {
		ctx := core.MustContext(context.Background(), kratosx.WithTrace("cache", storeCacheKey))
		storeIns = &Store{clients: &sync.Map{}}
		ctx.BeforeStart("load cache store", func() {
			storeIns.cache = cache.NewCacheAndInit(
				ctx,
				ctx.Redis(),
				storeCacheKey,
				cache.HookLoad(func() (map[string]*entity.Store, error) {
					return storeIns.load(ctx)
				}),
			)
		})
	})
	return storeIns
}

// load 全量加载 store 行（含 status=0 停用行：停用后存量文件 URL 仍需可解析）。
func (r Store) load(ctx core.Context) (map[string]*entity.Store, error) {
	var rows []*entity.Store
	if err := ctx.DB().Order("id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	bucket := make(map[string]*entity.Store, len(rows))
	for _, row := range rows {
		bucket[row.Keyword] = row
	}
	return bucket, nil
}

// ListStore 分页查询存储后端
func (r Store) ListStore(ctx core.Context, req *types.ListStoreRequest) ([]*entity.Store, uint32, error) {
	var (
		list  []*entity.Store
		total int64
	)

	db := ctx.DB().Model(&entity.Store{})
	if req.Keyword != nil {
		db = db.Where("keyword = ?", *req.Keyword)
	}
	if req.Provider != nil {
		db = db.Where("provider = ?", *req.Provider)
	}
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset, limit := req.Normalize()
	if err := db.Order("id asc").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, uint32(total), nil
}

// GetStore 按主键查询存储后端
func (r Store) GetStore(ctx core.Context, id uint32) (*entity.Store, error) {
	var ent entity.Store
	if err := ctx.DB().First(&ent, id).Error; err != nil {
		return nil, err
	}
	return &ent, nil
}

// GetStoreByKeyword 按 keyword 取存储后端（含 status，用于启用态判定）
func (r Store) GetStoreByKeyword(ctx core.Context, keyword string) (*entity.Store, error) {
	var ent entity.Store
	if err := ctx.DB().Where("keyword = ?", keyword).First(&ent).Error; err != nil {
		return nil, err
	}
	return &ent, nil
}

// CreateStore 新建存储后端（DB 写入与缓存 Put 同事务）。
func (r Store) CreateStore(ctx core.Context, ent *entity.Store) (uint32, error) {
	err := ctx.Transaction(func(ctx core.Context) error {
		if err := ctx.DB().Create(ent).Error; err != nil {
			return err
		}
		return r.cache.OP(ctx).Put(ent.Keyword, ent).Do()
	})
	if err != nil {
		return 0, err
	}
	return ent.Id, nil
}

// UpdateStore 局部更新（service 已读当前行并叠加传入字段；零值可写列如 status 在 entity 用指针，Updates 不跳过）。
// 写回后重取权威行（含最新时间戳）Put 进缓存，与 DB 写同事务。
func (r Store) UpdateStore(ctx core.Context, ent *entity.Store) error {
	return ctx.Transaction(func(ctx core.Context) error {
		if err := ctx.DB().Model(&entity.Store{}).Where("id = ?", ent.Id).Updates(ent).Error; err != nil {
			return err
		}
		var na entity.Store
		if err := ctx.DB().Where("id = ?", ent.Id).First(&na).Error; err != nil {
			return err
		}
		return r.cache.OP(ctx).Put(na.Keyword, &na).Do()
	})
}

// GetClient 取厂商客户端（缓存读行→按行指针记忆化构造，读路径零网络跳）。
func (r Store) GetClient(keyword string) (repository.Client, bool) {
	row, ok := r.cache.Get(keyword)
	if !ok {
		return nil, false
	}
	if m, ok := r.clients.Load(keyword); ok {
		if cm := m.(*clientMemo); cm.row == row {
			return cm.client, true
		}
	}
	client, err := store.NewClient(row)
	if err != nil {
		return nil, false
	}
	r.clients.Store(keyword, &clientMemo{row: row, client: client})
	return client, true
}

// ListClients 取全部客户端（GC / 离线核对遍历用）。
func (r Store) ListClients() []repository.Client {
	keys := r.cache.Keys()
	out := make([]repository.Client, 0, len(keys))
	for _, keyword := range keys {
		if client, ok := r.GetClient(keyword); ok {
			out = append(out, client)
		}
	}
	return out
}

// 编译期断言：*Store 实现 repository.Store
var _ repository.Store = (*Store)(nil)
