package service

import (
	"io"

	"github.com/limes-cloud/resource/api/errors"
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/repository"
	"github.com/limes-cloud/resource/internal/pkg/objectkey"
	"github.com/limes-cloud/resource/internal/types"
)

// File 文件读路径服务（列表 / 取 URL / 改名 / 移动）。
type File struct {
	repo       repository.File
	objectRepo repository.Object
	storeRepo  repository.Store
	appConfig  repository.AppConfig
	directory  *Directory
}

// NewFile 构造文件服务
func NewFile(repo repository.File, objectRepo repository.Object, storeRepo repository.Store, appConfig repository.AppConfig, directory *Directory) *File {
	return &File{repo: repo, objectRepo: objectRepo, storeRepo: storeRepo, appConfig: appConfig, directory: directory}
}

// GetFile 取单个文件 + 派生 URL（仅本人文件）。
func (f *File) GetFile(ctx core.Context, id uint32) (*types.FileItem, error) {
	file, err := f.repo.GetFile(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return nil, errors.ObjectNotFoundError()
		}
		ctx.Logger().Warnw("msg", "get file error", "id", id, "err", err.Error())
		return nil, errors.GetError()
	}
	if file.UserId != uint32(ctx.Auth().UserId) {
		return nil, errors.ObjectNotFoundError()
	}
	return f.toItem(ctx, file)
}

// ListFileByDirectory 目录内文件分页 + 批量派生 URL。
func (f *File) ListFileByDirectory(ctx core.Context, app string, directoryID uint32, req *types.PageRequest) ([]*types.FileItem, uint32, error) {
	files, total, err := f.repo.ListFileByDirectory(ctx, app, directoryID, req)
	if err != nil {
		ctx.Logger().Warnw("msg", "list file by directory error", "app", app, "directoryId", directoryID, "err", err.Error())
		return nil, 0, errors.GetError()
	}
	items, err := f.toItems(ctx, app, files)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ListFileByUser 我的文件分页 + 批量派生 URL。
func (f *File) ListFileByUser(ctx core.Context, app string, userID uint32, req *types.PageRequest) ([]*types.FileItem, uint32, error) {
	files, total, err := f.repo.ListFileByUser(ctx, app, userID, req)
	if err != nil {
		ctx.Logger().Warnw("msg", "list file by user error", "app", app, "userId", userID, "err", err.Error())
		return nil, 0, errors.GetError()
	}
	items, err := f.toItems(ctx, app, files)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// UpdateFile 改名 / 移动（file.app ≡ directory.app 不变量；仅本人文件；传了才改）。
func (f *File) UpdateFile(ctx core.Context, req *types.UpdateFileRequest) (*types.FileItem, error) {
	file, err := f.repo.GetFile(ctx, req.Id)
	if err != nil {
		if isNotFound(err) {
			return nil, errors.ObjectNotFoundError()
		}
		ctx.Logger().Warnw("msg", "get file error", "id", req.Id, "err", err.Error())
		return nil, errors.GetError()
	}
	// 归属：仅本人文件；app 一致性：文件必须属于请求方 app
	if file.UserId != uint32(ctx.Auth().UserId) || file.App != req.App {
		return nil, errors.ObjectNotFoundError()
	}

	ent := &entity.File{}
	ent.Id = req.Id
	if req.Name != nil {
		ent.Name = *req.Name
	}
	if req.DirectoryId != nil {
		dir, derr := f.directory.GetDirectoryById(ctx, uint32(*req.DirectoryId))
		if derr != nil {
			return nil, derr
		}
		if dir.App != file.App {
			return nil, errors.ParamsError("目标目录不属于当前应用")
		}
		ent.DirectoryId = uint32(*req.DirectoryId)
	}
	// name 不能空、directory_id 不能为 0，两者都未落值即无变更
	if ent.Name == "" && ent.DirectoryId == 0 {
		return f.toItem(ctx, file)
	}
	if uerr := f.repo.UpdateFile(ctx, ent); uerr != nil {
		ctx.Logger().Warnw("msg", "update file error", "id", req.Id, "err", uerr.Error())
		return nil, errors.UpdateError()
	}
	return f.GetFile(ctx, req.Id)
}

// DeleteFile 单文件删除：删 file 行 → 零引用则删 object 行 → 提交后删桶对象。
func (f *File) DeleteFile(ctx core.Context, app string, id uint32) error {
	file, err := f.repo.GetFile(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return errors.ObjectNotFoundError()
		}
		ctx.Logger().Warnw("msg", "get file error", "id", id, "err", err.Error())
		return errors.GetError()
	}
	if file.UserId != uint32(ctx.Auth().UserId) || file.App != app {
		return errors.ObjectNotFoundError()
	}

	// 删行前读出对象信息（桶删对象时用；可能已被前一轮回收掉）
	obj, err := f.objectRepo.GetObject(ctx, file.ObjectId)
	if err != nil && !isNotFound(err) {
		ctx.Logger().Warnw("msg", "get object error", "objectId", file.ObjectId, "err", err.Error())
		return errors.GetError()
	}

	// 同一事务读己之写：删 file 行 + 零引用才 CAS 删 object 行
	delBucket := false
	if terr := ctx.Transaction(func(tx core.Context) error {
		if e := f.repo.DeleteFile(tx, id); e != nil {
			return e
		}
		if obj == nil {
			return nil
		}
		// 归还配额：file 行创建时按引用 object 的 size 计入，删除按同口径归还（逻辑计量）
		if e := f.appConfig.ReleaseQuota(tx, app, obj.Size); e != nil {
			return e
		}
		deleted, e := f.objectRepo.DeleteObjectIfUnreferenced(tx, obj.Id)
		if e != nil {
			return e
		}
		delBucket = deleted
		return nil
	}); terr != nil {
		ctx.Logger().Warnw("msg", "delete file tx error", "id", id, "err", terr.Error())
		return errors.DeleteError()
	}
	if !delBucket {
		return nil
	}

	// 提交后删桶对象：先查 (store_keyword, md5) 是否因重新上传出现新行，命中即放弃
	if exists, e := f.objectRepo.ExistsObject(ctx, obj.StoreKeyword, obj.Md5); e == nil && exists {
		return nil
	}
	if client, ok := f.storeRepo.GetClient(obj.StoreKeyword); ok {
		_ = client.Delete(ctx, objectkey.ObjectName(obj.Md5, obj.Ext))
	}
	return nil
}

// toItems 批量派生 URL（按 app 取一次 TTL，按 object_id 批量取对象，避免逐行点查）。
func (f *File) toItems(ctx core.Context, app string, files []*entity.File) ([]*types.FileItem, error) {
	if len(files) == 0 {
		return nil, nil
	}
	ids := make([]uint32, 0, len(files))
	for _, fl := range files {
		ids = append(ids, fl.ObjectId)
	}
	objs, err := f.objectRepo.ListObjectByIDs(ctx, ids)
	if err != nil {
		ctx.Logger().Warnw("msg", "list object by ids error", "err", err.Error())
		return nil, errors.GetError()
	}
	objMap := make(map[uint32]*entity.Object, len(objs))
	for _, o := range objs {
		objMap[o.Id] = o
	}
	ttl := resolveURLTtl(ctx, f.appConfig, app)
	items := make([]*types.FileItem, 0, len(files))
	for _, fl := range files {
		item := fileItemFrom(fl)
		if o, ok := objMap[fl.ObjectId]; ok {
			item.Size = o.Size
			item.Ext = o.Ext
			item.Url = genFileURL(ctx, f.storeRepo, o, ttl)
			item.Key = objectkey.ObjectName(o.Md5, o.Ext)
		}
		items = append(items, item)
	}
	return items, nil
}

// toItem 单个文件派生 URL。
func (f *File) toItem(ctx core.Context, file *entity.File) (*types.FileItem, error) {
	items, err := f.toItems(ctx, file.App, []*entity.File{file})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.ObjectNotFoundError()
	}
	return items[0], nil
}

// fileItemFrom 由 file 行构造列表项（size/ext/url 待对象补齐）。
func fileItemFrom(fl *entity.File) *types.FileItem {
	return &types.FileItem{
		Id:          fl.Id,
		App:         fl.App,
		UserId:      fl.UserId,
		DirectoryId: fl.DirectoryId,
		Name:        fl.Name,
		ObjectId:    fl.ObjectId,
		CreatedAt:   fl.CreatedAt,
	}
}

// GetFileURLByKey 按对象 key（内容指纹 md5.ext）取可访问地址；对象不存在返回对象未找到。
// app 不参与定位：对象按 md5 全局内容寻址，URL 由对象自带的 store_keyword 生成。
func (f *File) GetFileURLByKey(ctx core.Context, key string) (*types.GetFileURLByKeyReply, error) {
	md5hex, _, err := objectkey.ParseObjectName(key)
	if err != nil {
		return nil, errors.ParamsError("对象key不合法")
	}
	obj, err := f.objectRepo.GetObjectByMd5Any(ctx, md5hex)
	if err != nil {
		if isNotFound(err) {
			return nil, errors.ObjectNotFoundError()
		}
		ctx.Logger().Warnw("msg", "get object by md5 error", "md5", md5hex, "err", err.Error())
		return nil, errors.GetError()
	}
	return &types.GetFileURLByKeyReply{Url: genFileURL(ctx, f.storeRepo, obj, defaultURLTtl)}, nil
}

// ReadFileByKey 按对象 key 服务端读取文件内容（返回读流与 Content-Type）。
// 用于前端跨域读取受限的场景：对象存储（如 OSS）未开 CORS 时，<img> 可显示但 fetch 无法取字节，
// 由服务端代理读出后原样返回，规避存储侧 CORS 限制。
func (f *File) ReadFileByKey(ctx core.Context, key string) (io.ReadCloser, string, error) {
	md5hex, ext, err := objectkey.ParseObjectName(key)
	if err != nil {
		return nil, "", errors.ParamsError("对象key不合法")
	}
	obj, err := f.objectRepo.GetObjectByMd5Any(ctx, md5hex)
	if err != nil {
		if isNotFound(err) {
			return nil, "", errors.ObjectNotFoundError()
		}
		ctx.Logger().Warnw("msg", "get object by md5 error", "md5", md5hex, "err", err.Error())
		return nil, "", errors.GetError()
	}
	client, ok := f.storeRepo.GetClient(obj.StoreKeyword)
	if !ok {
		return nil, "", errors.StoreDisabledError()
	}
	rc, err := client.Get(ctx, objectkey.ObjectName(obj.Md5, obj.Ext))
	if err != nil {
		ctx.Logger().Warnw("msg", "get object content error", "key", objectkey.ObjectName(obj.Md5, obj.Ext), "err", err.Error())
		return nil, "", errors.GetError()
	}
	return rc, objectkey.ContentType(ext), nil
}
