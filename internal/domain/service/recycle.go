package service

import (
	"github.com/limes-cloud/resource/api/errors"
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/repository"
	"github.com/limes-cloud/resource/internal/pkg/objectkey"
)

// file.status 取值
const (
	statusFileNormal   uint8 = 1 // 正常
	statusFileDeleting uint8 = 2 // 待删除（软删，后台回收）
)

// Recycle 回收服务：删目录软删 + 后台分批物理回收（文件行 / 对象桶）。
type Recycle struct {
	fileRepo      repository.File
	objectRepo    repository.Object
	storeRepo     repository.Store
	directoryRepo repository.Directory
	appConfig     repository.AppConfig
}

// NewRecycle 构造回收服务
func NewRecycle(fileRepo repository.File, objectRepo repository.Object, storeRepo repository.Store, directoryRepo repository.Directory, appConfig repository.AppConfig) *Recycle {
	return &Recycle{fileRepo: fileRepo, objectRepo: objectRepo, storeRepo: storeRepo, directoryRepo: directoryRepo, appConfig: appConfig}
}

// DeleteDirectory 删目录（软删）：标记子树文件待删除 + 删 directory 树，物理回收交后台。
func (r *Recycle) DeleteDirectory(ctx core.Context, app string, id uint32) error {
	cur, err := r.directoryRepo.GetDirectory(ctx, app, id)
	if err != nil {
		if isNotFound(err) {
			return errors.ObjectNotFoundError()
		}
		return errors.DeleteError()
	}

	// 同一事务：先标记子树 file 行待删除（须靠 directory 行定位子树，必须在删目录前），再删 directory 树。
	if terr := ctx.Transaction(func(tx core.Context) error {
		if _, e := r.fileRepo.MarkDeleteInSubtree(tx, app, cur.Id, cur.Path); e != nil {
			return e
		}
		return r.directoryRepo.DeleteDirectoryTree(tx, app, cur.Id, cur.Path)
	}); terr != nil {
		ctx.Logger().Warnw("msg", "delete directory error", "id", id, "err", terr.Error())
		return errors.DeleteError()
	}

	// 待删文件已不计入配额口径（status=2），重算归还；物理删文件行与桶对象由后台 sweep 处理。
	if _, e := r.appConfig.RecalcUsage(ctx, app); e != nil {
		ctx.Logger().Warnw("msg", "recalc usage after delete directory error", "app", app, "err", e.Error())
	}
	return nil
}

// SweepDeletedFiles 后台阶段 A：分批硬删 status=待删除 的 file 行，返回本批删除行数。
func (r *Recycle) SweepDeletedFiles(ctx core.Context, limit int) (int64, error) {
	return r.fileRepo.DeleteFilesByStatus(ctx, statusFileDeleting, limit)
}

// SweepUnreferencedObjects 后台阶段 B：扫零引用对象 → 逐个 CAS 删行（防重新上传引用）→ 按 store 批量删桶。
// 返回本批成功回收的对象数。桶删除失败不影响行删除；行删不掉（又被引用）则跳过，下轮重扫。
func (r *Recycle) SweepUnreferencedObjects(ctx core.Context, limit int) (int, error) {
	objs, err := r.objectRepo.ScanUnreferenced(ctx, 0, limit)
	if err != nil {
		return 0, err
	}

	// 按 store 分组收集"确实删了行、且未被重新上传"的桶 key
	byStore := make(map[string][]string)
	done := 0
	for _, obj := range objs {
		// CAS 删行：带 NOT EXISTS 再查一次，扫描之后被新 file 行引用则删不掉，跳过
		deleted, e := r.objectRepo.DeleteObjectIfUnreferenced(ctx, obj.Id)
		if e != nil || !deleted {
			continue
		}
		done++
		// 重新上传防误删：删行后同 (store,md5) 若又出现新行（key 与内容相同），跳过删桶
		if exists, ee := r.objectRepo.ExistsObject(ctx, obj.StoreKeyword, obj.Md5); ee == nil && exists {
			continue
		}
		byStore[obj.StoreKeyword] = append(byStore[obj.StoreKeyword], objectkey.ObjectName(obj.Md5, obj.Ext))
	}

	for store, keys := range byStore {
		client, ok := r.storeRepo.GetClient(store)
		if !ok {
			continue
		}
		if e := client.DeleteBatch(ctx, keys); e != nil {
			ctx.Logger().Warnw("msg", "batch delete bucket error", "store", store, "count", len(keys), "err", e.Error())
		}
	}
	return done, nil
}
