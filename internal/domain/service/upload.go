package service

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/limes-cloud/kratosx/pkg/value"

	"github.com/limes-cloud/resource/api/errors"
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/repository"
	"github.com/limes-cloud/resource/internal/pkg/chunk"
	"github.com/limes-cloud/resource/internal/pkg/objectkey"
	"github.com/limes-cloud/resource/internal/pkg/provider"
	"github.com/limes-cloud/resource/internal/types"
)

// defaultURLTtl 文件 URL 兜底有效期（配置缺失时使用）。
const defaultURLTtl = time.Hour

// maxURLDownloadSize 服务端 URL 下载的硬上限（字节）；目录/应用未设限时兜底，防空转内存。
const maxURLDownloadSize = 100 * 1024 * 1024

// urlDownloadClient 服务端 URL 下载客户端（超时内联，紧跟重定向）。
var urlDownloadClient = &http.Client{Timeout: 60 * time.Second}

// Upload 上传领域服务（分片上传主链路）。
type Upload struct {
	repo       repository.Upload
	objectRepo repository.Object
	fileRepo   repository.File
	storeRepo  repository.Store
	appConfig  repository.AppConfig
	directory  *Directory
}

// NewUpload 构造上传服务
func NewUpload(repo repository.Upload, objectRepo repository.Object, fileRepo repository.File, storeRepo repository.Store, appConfig repository.AppConfig, directory *Directory) *Upload {
	return &Upload{repo: repo, objectRepo: objectRepo, fileRepo: fileRepo, storeRepo: storeRepo, appConfig: appConfig, directory: directory}
}

// GetUploadChunkSize 取分片大小与片数（纯函数不查库）。
func (s *Upload) GetUploadChunkSize(_ core.Context, req *types.GetUploadChunkSizeRequest) (*types.GetUploadChunkSizeResponse, error) {
	size, err := chunk.Size(req.TotalSize)
	if err != nil {
		return nil, mapChunkErr(err)
	}
	return &types.GetUploadChunkSizeResponse{ChunkSize: size, ChunkCount: chunk.Count(req.TotalSize, size)}, nil
}

// PrepareUploadFile 预上传：秒传判定 + 落桶解析 + 召回 / 建会话。
func (s *Upload) PrepareUploadFile(ctx core.Context, req *types.PrepareUploadFileRequest) (*types.PrepareUploadFileResponse, error) {
	// user_id 必须指向真实用户，0（系统）不是合法身份，拒绝其全部上传/建行路径
	userID := uint32(ctx.Auth().UserId)
	if userID == 0 {
		return nil, errors.ParamsError("用户身份异常")
	}

	if !objectkey.IsValidExt(req.Ext) || !objectkey.IsLowerHex32(req.Md5) || req.Name == "" {
		return nil, errors.ParamsError()
	}

	// 2. app_config
	cfg, err := s.appConfig.GetAppConfig(ctx, req.App)
	if err != nil {
		if isNotFound(err) {
			return nil, errors.AppNotConfiguredError()
		}
		ctx.Logger().Warnw("msg", "get app config error", "app", req.App, "err", err.Error())
		return nil, errors.GetError()
	}
	if value.Value(cfg.Status) == 0 {
		return nil, errors.AppNotConfiguredError()
	}

	// 3. store 启用态 + 厂商能力位（核验前提）
	store, err := s.storeRepo.GetStoreByKeyword(ctx, cfg.StoreKeyword)
	if err != nil {
		if isNotFound(err) {
			return nil, errors.StoreDisabledError()
		}
		ctx.Logger().Warnw("msg", "get store by keyword error", "keyword", cfg.StoreKeyword, "err", err.Error())
		return nil, errors.GetError()
	}
	if value.Value(store.Status) == 0 {
		return nil, errors.StoreDisabledError()
	}
	if !provider.PartETagIsContentMD5(store.Provider) {
		return nil, errors.StoreDisabledError("该存储后端不支持分片摘要核验，已拒绝上传")
	}

	// 4. 解析目录（不存在则自动建，策略抄 app_config 快照）
	dir, err := s.directory.EnsurePath(ctx, req.App, req.Path, cfg)
	if err != nil {
		return nil, err
	}

	// 5. 校验 accept / max_size（只读 directory 一行的策略）
	if !dir.AllowExt(req.Ext) {
		return nil, errors.FileTypeNotAllowedError()
	}
	if !dir.WithinSize(req.Size) {
		return nil, errors.FileTooLargeError()
	}

	// 5.1 配额软预检（权威在完成事务的 ReserveQuota）：提前拒超限，省无效上传；used_bytes 走缓存可能偏小，仅作快速失败
	if cfg.Quota != nil && *cfg.Quota > 0 && cfg.UsedBytes+req.Size > *cfg.Quota {
		return nil, errors.QuotaExceededError()
	}

	// 6. 秒传判定：按 (store_keyword, md5) 查 object
	obj, err := s.objectRepo.GetObjectByMd5(ctx, cfg.StoreKeyword, req.Md5)
	if err == nil {
		if obj.Size != req.Size {
			return nil, errors.SizeMismatchError()
		}
		if _, ferr := s.finishFile(ctx, &finishFileParams{Obj: obj, UploadId: nil, App: dir.App, UserID: userID, DirID: dir.Id, Name: req.Name}); ferr != nil {
			return nil, ferr
		}
		key := objectkey.ObjectName(obj.Md5, obj.Ext)
		return &types.PrepareUploadFileResponse{Skipped: true, Url: genFileURL(ctx, s.storeRepo, obj, urlTtlFromCfg(cfg)), Key: key, Md5: req.Md5}, nil
	}
	if !isNotFound(err) {
		ctx.Logger().Warnw("msg", "get object by md5 error", "keyword", cfg.StoreKeyword, "md5", req.Md5, "err", err.Error())
		return nil, errors.GetError()
	}

	// 召回：按 (store_keyword, md5, user_id, directory_id) 查在飞会话
	exist, err := s.repo.GetUploadByKey(ctx, &types.UploadKey{StoreKeyword: cfg.StoreKeyword, Md5: req.Md5, UserID: userID, DirectoryID: dir.Id})
	if err == nil {
		key := objectkey.ObjectName(exist.Md5, exist.Ext)
		client, ok := s.storeRepo.GetClient(cfg.StoreKeyword)
		if !ok {
			return nil, errors.StoreDisabledError()
		}
		parts, lerr := client.ListParts(ctx, key, exist.UploadId)
		if lerr == nil {
			if exist.TotalSize != req.Size {
				return nil, errors.SizeMismatchError()
			}
			s.repo.TouchUpload(ctx, exist.Id)
			chunkSize, _ := chunk.Size(exist.TotalSize)
			uploaded := make([]uint32, 0, len(parts))
			for _, p := range parts {
				uploaded = append(uploaded, p.Number)
			}
			return &types.PrepareUploadFileResponse{
				UploadId:      exist.UploadId,
				Md5:           exist.Md5,
				ChunkSize:     chunkSize,
				ChunkCount:    chunk.Count(exist.TotalSize, chunkSize),
				UploadedParts: uploaded,
				Key:           objectkey.ObjectName(exist.Md5, exist.Ext),
			}, nil
		}
		if errors.IsUploadSessionNotFoundError(lerr) {
			// 厂商侧已结束 → 删会话行，落未命中分支
			s.repo.DeleteUpload(ctx, exist.Id)
		} else {
			ctx.Logger().Warnw("msg", "recall list parts error", "uploadId", exist.UploadId, "err", lerr.Error())
			return nil, errors.SystemError()
		}
	} else if !isNotFound(err) {
		ctx.Logger().Warnw("msg", "get upload by key error", "keyword", cfg.StoreKeyword, "md5", req.Md5, "err", err.Error())
		return nil, errors.GetError()
	}

	// 未命中 / 删行后 → 建会话（不论 total_size，含 chunkCount=1 单片）
	return s.createSession(ctx, &createSessionParams{Cfg: cfg, Dir: dir, Req: req, UserID: userID})
}

// createSessionParams 建会话参数
type createSessionParams struct {
	Cfg    *entity.AppConfig
	Dir    *entity.Directory
	Req    *types.PrepareUploadFileRequest
	UserID uint32
}

// createSession 建分片上传会话（直接走厂商 multipart）。
func (s *Upload) createSession(ctx core.Context, p *createSessionParams) (*types.PrepareUploadFileResponse, error) {
	cfg, dir, req, userID := p.Cfg, p.Dir, p.Req, p.UserID
	chunkSize, err := chunk.Size(req.Size)
	if err != nil {
		return nil, mapChunkErr(err)
	}
	chunkCount := chunk.Count(req.Size, chunkSize)

	client, ok := s.storeRepo.GetClient(cfg.StoreKeyword)
	if !ok {
		return nil, errors.StoreDisabledError()
	}
	key := objectkey.ObjectName(req.Md5, req.Ext)
	uploadID, err := client.NewMultipart(ctx, key, objectkey.ContentType(req.Ext))
	if err != nil {
		ctx.Logger().Warnw("msg", "new multipart error", "keyword", cfg.StoreKeyword, "key", key, "err", err.Error())
		return nil, errors.SystemError()
	}

	ent := &entity.Upload{
		StoreKeyword: cfg.StoreKeyword,
		Md5:          req.Md5,
		Ext:          req.Ext,
		UserId:       userID,
		DirectoryId:  dir.Id,
		Name:         req.Name,
		UploadId:     uploadID,
		TotalSize:    req.Size,
	}
	if _, err := s.repo.CreateUpload(ctx, ent); err != nil {
		if isDuplicate(err) {
			// 撞 uidx_upload_session → 重查复用赢家
			if winner, gerr := s.repo.GetUploadByKey(ctx, &types.UploadKey{StoreKeyword: cfg.StoreKeyword, Md5: req.Md5, UserID: userID, DirectoryID: dir.Id}); gerr == nil {
				s.repo.TouchUpload(ctx, winner.Id)
				return &types.PrepareUploadFileResponse{
					UploadId:   winner.UploadId,
					Md5:        winner.Md5,
					ChunkSize:  chunkSize,
					ChunkCount: chunkCount,
					Key:        objectkey.ObjectName(winner.Md5, winner.Ext),
				}, nil
			}
		}
		ctx.Logger().Warnw("msg", "create upload session error", "keyword", cfg.StoreKeyword, "md5", req.Md5, "err", err.Error())
		return nil, errors.CreateError()
	}

	return &types.PrepareUploadFileResponse{
		UploadId:   uploadID,
		Md5:        req.Md5,
		ChunkSize:  chunkSize,
		ChunkCount: chunkCount,
		Key:        key,
	}, nil
}

// GetUploadPartURLs 取分片上传预签名地址。
func (s *Upload) GetUploadPartURLs(ctx core.Context, req *types.GetUploadPartURLsRequest) (*types.GetUploadPartURLsResponse, error) {
	userID := uint32(ctx.Auth().UserId)

	sess, err := s.repo.GetUpload(ctx, req.UploadId, userID)
	if err != nil {
		if isNotFound(err) {
			return nil, errors.UploadSessionNotFoundError()
		}
		ctx.Logger().Warnw("msg", "get upload session error", "uploadId", req.UploadId, "err", err.Error())
		return nil, errors.GetError()
	}

	if len(req.PartNumbers) == 0 || len(req.PartNumbers) > 200 {
		return nil, errors.ParamsError()
	}
	chunkSize, cerr := chunk.Size(sess.TotalSize)
	if cerr != nil {
		return nil, mapChunkErr(cerr)
	}
	chunkCount := chunk.Count(sess.TotalSize, chunkSize)
	for _, n := range req.PartNumbers {
		if n == 0 || n > chunkCount {
			return nil, errors.ParamsError()
		}
	}

	client, ok := s.storeRepo.GetClient(sess.StoreKeyword)
	if !ok {
		return nil, errors.StoreDisabledError()
	}
	key := objectkey.ObjectName(sess.Md5, sess.Ext)
	ttl := s.partURLTtl(ctx)

	urls := make([]types.PartURL, 0, len(req.PartNumbers))
	for _, n := range req.PartNumbers {
		u, perr := client.PresignPart(ctx, &repository.PresignPartInput{Key: key, UploadID: sess.UploadId, PartNumber: n, Ttl: ttl})
		if perr != nil {
			ctx.Logger().Warnw("msg", "presign part error", "uploadId", sess.UploadId, "part", n, "err", perr.Error())
			return nil, errors.SystemError()
		}
		urls = append(urls, types.PartURL{PartNumber: n, Url: u})
	}

	// 取址即续期（滑动过期时钟）
	s.repo.TouchUpload(ctx, sess.Id)
	return &types.GetUploadPartURLsResponse{Urls: urls}, nil
}

// CompleteUpload 完成分片上传（合并判定 + 完成事务 + 幂等收口）。
func (s *Upload) CompleteUpload(ctx core.Context, req *types.CompleteUploadRequest) (*types.CompleteUploadResponse, error) {
	userID := uint32(ctx.Auth().UserId)

	sess, err := s.repo.GetUpload(ctx, req.UploadId, userID)
	if err != nil {
		if isNotFound(err) {
			// 会话行已删，按 upload_id 命中 file 行则重放
			if f, ferr := s.fileRepo.GetFileByUploadId(ctx, req.UploadId); ferr == nil && f != nil {
				if obj, oerr := s.objectRepo.GetObject(ctx, f.ObjectId); oerr == nil && obj != nil {
					return &types.CompleteUploadResponse{
						Url: genFileURL(ctx, s.storeRepo, obj, resolveURLTtl(ctx, s.appConfig, f.App)),
						Key: objectkey.ObjectName(obj.Md5, obj.Ext),
					}, nil
				}
			}
			return nil, errors.UploadSessionNotFoundError("会话已结束，请重新发起上传")
		}
		ctx.Logger().Warnw("msg", "get upload session error", "uploadId", req.UploadId, "err", err.Error())
		return nil, errors.GetError()
	}

	client, ok := s.storeRepo.GetClient(sess.StoreKeyword)
	if !ok {
		return nil, errors.StoreDisabledError()
	}
	key := objectkey.ObjectName(sess.Md5, sess.Ext)

	// 1) ListParts 全量分页
	parts, err := client.ListParts(ctx, key, sess.UploadId)
	if err != nil {
		if errors.IsUploadSessionNotFoundError(err) {
			return s.replayOrAbort(ctx, sess, client, key)
		}
		ctx.Logger().Warnw("msg", "list parts error", "uploadId", sess.UploadId, "err", err.Error())
		return nil, errors.SystemError()
	}

	// 2) 四条判据：缺片 / 形状不符 → 不合并、会话存活；摘要不符 → Abort + 删会话
	if err := s.checkCompleteness(sess, parts); err != nil {
		return nil, err
	}
	if err := s.checkDigest(sess, parts); err != nil {
		client.Abort(ctx, key, sess.UploadId)
		s.repo.DeleteUpload(ctx, sess.Id)
		return nil, err
	}

	// 3) CompleteMultipartUpload（对象此刻才诞生）
	completed := make([]repository.CompletedPart, 0, len(parts))
	for _, p := range parts {
		completed = append(completed, repository.CompletedPart{Number: p.Number, ETag: p.ETag})
	}
	if err := client.Complete(ctx, key, sess.UploadId, completed); err != nil {
		if errors.IsUploadSessionNotFoundError(err) {
			return s.replayOrAbort(ctx, sess, client, key)
		}
		ctx.Logger().Warnw("msg", "complete multipart error", "uploadId", sess.UploadId, "err", err.Error())
		return nil, errors.SystemError()
	}

	// 4) 完成事务
	obj, f, terr := s.completeTransaction(ctx, sess)
	if terr != nil {
		return nil, terr
	}
	return &types.CompleteUploadResponse{
		Url: genFileURL(ctx, s.storeRepo, obj, resolveURLTtl(ctx, s.appConfig, f.App)),
		Key: objectkey.ObjectName(obj.Md5, obj.Ext),
	}, nil
}

// replayOrAbort 厂商侧已结束时，桶里对象还在则重放完成事务，否则视为已中止删会话。
func (s *Upload) replayOrAbort(ctx core.Context, sess *entity.Upload, client repository.Client, key string) (*types.CompleteUploadResponse, error) {
	if _, herr := client.Head(ctx, key); herr == nil {
		if obj, f, terr := s.completeTransaction(ctx, sess); terr == nil && f != nil {
			return &types.CompleteUploadResponse{
				Url: genFileURL(ctx, s.storeRepo, obj, resolveURLTtl(ctx, s.appConfig, f.App)),
				Key: objectkey.ObjectName(obj.Md5, obj.Ext),
			}, nil
		}
	}
	s.repo.DeleteUpload(ctx, sess.Id)
	return nil, errors.UploadSessionNotFoundError("会话已结束，请重新发起上传")
}

// GarbageCollect 上传会话 GC：过期会话先跑合并判据试合并，不齐 → Abort + 删行。
func (s *Upload) GarbageCollect(ctx core.Context, before int64, limit int) (int, error) {
	sessions, err := s.repo.ListExpiredUpload(ctx, before, limit)
	if err != nil {
		ctx.Logger().Warnw("msg", "list expired upload error", "err", err.Error())
		return 0, errors.GetError()
	}
	var processed int
	for _, sess := range sessions {
		client, ok := s.storeRepo.GetClient(sess.StoreKeyword)
		if !ok {
			continue
		}
		key := objectkey.ObjectName(sess.Md5, sess.Ext)

		parts, lerr := client.ListParts(ctx, key, sess.UploadId)
		if lerr != nil {
			if errors.IsUploadSessionNotFoundError(lerr) {
				s.repo.DeleteUpload(ctx, sess.Id)
			}
			continue
		}

		// 验尸：判据齐且摘要核验通过则完成事务，捡回一整份流量
		if s.checkCompleteness(sess, parts) != nil || s.checkDigest(sess, parts) != nil {
			client.Abort(ctx, key, sess.UploadId)
			s.repo.DeleteUpload(ctx, sess.Id)
			processed++
			continue
		}
		completed := make([]repository.CompletedPart, 0, len(parts))
		for _, p := range parts {
			completed = append(completed, repository.CompletedPart{Number: p.Number, ETag: p.ETag})
		}
		if cerr := client.Complete(ctx, key, sess.UploadId, completed); cerr != nil {
			if errors.IsUploadSessionNotFoundError(cerr) {
				s.repo.DeleteUpload(ctx, sess.Id)
			}
			continue
		}
		if _, _, terr := s.completeTransaction(ctx, sess); terr != nil {
			continue
		}
		processed++
	}
	return processed, nil
}

// checkCompleteness 判据 1~3：片号集合 / 形状 / 总大小。不齐 → 不合并，会话保持存活。
func (s *Upload) checkCompleteness(sess *entity.Upload, parts []repository.UploadedPart) error {
	chunkSize, err := chunk.Size(sess.TotalSize)
	if err != nil {
		return mapChunkErr(err)
	}
	chunkCount := chunk.Count(sess.TotalSize, chunkSize)
	if uint32(len(parts)) != chunkCount {
		return errors.ChunkNotCompleteError("分片数量不符")
	}

	seen := make(map[uint32]bool, chunkCount)
	var sum uint64
	for _, p := range parts {
		if p.Number == 0 || p.Number > chunkCount || seen[p.Number] {
			return errors.ChunkNotCompleteError("分片号越界或重复")
		}
		seen[p.Number] = true
		if p.Number != chunkCount {
			if p.Size != uint64(chunkSize) {
				return errors.ChunkNotCompleteError("非末片大小不符")
			}
		} else if p.Size == 0 || p.Size > uint64(chunkSize) {
			return errors.ChunkNotCompleteError("末片大小不符")
		}
		sum += p.Size
	}
	for i := uint32(1); i <= chunkCount; i++ {
		if !seen[i] {
			return errors.ChunkNotCompleteError("存在缺片")
		}
	}
	if sum != sess.TotalSize {
		return errors.SizeMismatchError()
	}
	return nil
}

// checkDigest 按分片 ETag 重算 md5 并与会话声明值比对。
func (s *Upload) checkDigest(sess *entity.Upload, parts []repository.UploadedPart) error {
	digParts := make([]chunk.Part, 0, len(parts))
	for _, p := range parts {
		digParts = append(digParts, chunk.Part{Number: p.Number, Digest: chunk.NormalizeETag(p.ETag)})
	}
	got, err := chunk.Digest(digParts)
	if err != nil {
		return errors.ChunkNotCompleteError("分片摘要非法")
	}
	if got != sess.Md5 {
		return errors.SizeMismatchError("内容摘要核验失败")
	}
	return nil
}

// completeTransaction 确保对象行 + 建 file + 删会话，单事务。
func (s *Upload) completeTransaction(ctx core.Context, sess *entity.Upload) (*entity.Object, *entity.File, error) {
	var obj *entity.Object
	var file *entity.File
	err := ctx.Transaction(func(txCtx core.Context) error {
		o, e := s.ensureObject(txCtx, &ensureObjectParams{StoreKeyword: sess.StoreKeyword, Md5: sess.Md5, Ext: sess.Ext, Size: sess.TotalSize})
		if e != nil {
			return e
		}
		dir, e := s.directory.GetDirectoryById(txCtx, sess.DirectoryId)
		if e != nil {
			ctx.Logger().Warnw("msg", "get directory in complete tx error", "directoryId", sess.DirectoryId, "err", e.Error())
			return errors.GetError()
		}
		f := &entity.File{
			App:         dir.App,
			UserId:      sess.UserId,
			DirectoryId: sess.DirectoryId,
			Name:        sess.Name,
			ObjectId:    o.Id,
			UploadId:    &sess.UploadId,
			Status:      value.Pointer(statusFileNormal),
		}
		id, e := s.fileRepo.CreateFile(txCtx, f)
		if e != nil {
			if isDuplicate(e) {
				if existing, ge := s.fileRepo.GetFileByUploadId(txCtx, sess.UploadId); ge == nil && existing != nil {
					file = existing
					obj = o
					return nil
				}
			}
			ctx.Logger().Warnw("msg", "create file in complete tx error", "uploadId", sess.UploadId, "err", e.Error())
			return errors.CreateError()
		}
		f.Id = id
		file = f
		obj = o
		// 新建 file 行 → 占用配额；超限则返回错误回滚建行与会话删除
		ok, qe := s.appConfig.ReserveQuota(txCtx, dir.App, o.Size)
		if qe != nil {
			ctx.Logger().Warnw("msg", "reserve quota error", "app", dir.App, "err", qe.Error())
			return errors.SystemError()
		}
		if !ok {
			return errors.QuotaExceededError()
		}
		if e := s.repo.DeleteUpload(txCtx, sess.Id); e != nil {
			ctx.Logger().Warnw("msg", "delete upload in complete tx error", "id", sess.Id, "err", e.Error())
			return errors.DeleteError()
		}
		return nil
	})
	return obj, file, err
}

// ensureObjectParams 确保对象行存在的参数
type ensureObjectParams struct {
	StoreKeyword string
	Md5          string
	Ext          string
	Size         uint64
}

// ensureObject 确保对象行存在（幂等）；size 不符即碰撞护栏。
func (s *Upload) ensureObject(ctx core.Context, p *ensureObjectParams) (*entity.Object, error) {
	storeKeyword, md5, ext, size := p.StoreKeyword, p.Md5, p.Ext, p.Size
	if existing, e := s.objectRepo.GetObjectByMd5(ctx, storeKeyword, md5); e == nil {
		if existing.Size != size {
			return nil, errors.SizeMismatchError()
		}
		return existing, nil
	} else if !isNotFound(e) {
		ctx.Logger().Warnw("msg", "ensure object query error", "keyword", storeKeyword, "md5", md5, "err", e.Error())
		return nil, errors.GetError()
	}
	created, e := s.objectRepo.CreateObject(ctx, &entity.Object{
		StoreKeyword: storeKeyword,
		Md5:          md5,
		Ext:          ext,
		Size:         size,
	})
	if e != nil {
		if isDuplicate(e) {
			if existing, ge := s.objectRepo.GetObjectByMd5(ctx, storeKeyword, md5); ge == nil {
				if existing.Size != size {
					return nil, errors.SizeMismatchError()
				}
				return existing, nil
			}
		}
		ctx.Logger().Warnw("msg", "create object error", "keyword", storeKeyword, "md5", md5, "err", e.Error())
		return nil, errors.CreateError()
	}
	return created, nil
}

// finishFileParams 建文件行参数
type finishFileParams struct {
	Obj      *entity.Object
	UploadId *string // 可为 nil（秒传无会话）
	App      string
	UserID   uint32
	DirID    uint32
	Name     string
}

// finishFile 建文件行（秒传复用，uploadId 可为 nil）。
func (s *Upload) finishFile(ctx core.Context, p *finishFileParams) (*entity.File, error) {
	obj, uploadId, app, userId, dirId, name := p.Obj, p.UploadId, p.App, p.UserID, p.DirID, p.Name
	var file *entity.File
	err := ctx.Transaction(func(txCtx core.Context) error {
		f := &entity.File{
			App:         app,
			UserId:      userId,
			DirectoryId: dirId,
			Name:        name,
			ObjectId:    obj.Id,
			UploadId:    uploadId,
			Status:      value.Pointer(statusFileNormal),
		}
		id, e := s.fileRepo.CreateFile(txCtx, f)
		if e != nil {
			if isDuplicate(e) && uploadId != nil {
				if existing, ge := s.fileRepo.GetFileByUploadId(txCtx, *uploadId); ge == nil && existing != nil {
					file = existing
					return nil
				}
			}
			ctx.Logger().Warnw("msg", "create file error", "app", app, "objectId", obj.Id, "err", e.Error())
			return errors.CreateError()
		}
		f.Id = id
		file = f
		// 新建 file 行 → 占用配额；超限则返回错误回滚建行（秒传同样计量，不绕过）
		ok, qe := s.appConfig.ReserveQuota(txCtx, app, obj.Size)
		if qe != nil {
			ctx.Logger().Warnw("msg", "reserve quota error", "app", app, "err", qe.Error())
			return errors.SystemError()
		}
		if !ok {
			return errors.QuotaExceededError()
		}
		return nil
	})
	return file, err
}

// genFileURL 由对象派生只读 URL（公有读拼 base，其余走临时签名）。
func genFileURL(ctx core.Context, storeRepo repository.Store, obj *entity.Object, ttl time.Duration) string {
	client, ok := storeRepo.GetClient(obj.StoreKeyword)
	if !ok {
		return ""
	}
	key := objectkey.ObjectName(obj.Md5, obj.Ext)
	url, err := client.GenTemporaryURL(ctx, key, ttl)
	if err != nil {
		ctx.Logger().Warnw("msg", "gen file url error", "err", err.Error())
		return ""
	}
	return url
}

// urlTtlFromCfg 从已加载的 app_config 取文件 URL 有效期。
func urlTtlFromCfg(cfg *entity.AppConfig) time.Duration {
	if cfg != nil && cfg.TempUrlTtl > 0 {
		return time.Duration(cfg.TempUrlTtl) * time.Second
	}
	return defaultURLTtl
}

// resolveURLTtl 取文件 URL 有效期（优先 app_config.temp_url_ttl）。
func resolveURLTtl(ctx core.Context, appConfig repository.AppConfig, app string) time.Duration {
	if cfg, err := appConfig.GetAppConfig(ctx, app); err == nil && cfg != nil && cfg.TempUrlTtl > 0 {
		return time.Duration(cfg.TempUrlTtl) * time.Second
	}
	return defaultURLTtl
}

// partURLTtl 取分片上传地址有效期（读配置，缺失时给默认值）。
func (s *Upload) partURLTtl(ctx core.Context) time.Duration {
	if c := ctx.Config(); c != nil && c.Upload != nil && c.Upload.PartUrlTtl > 0 {
		return c.Upload.PartUrlTtl
	}
	return time.Hour
}

// mapChunkErr 把 chunk 包的错误映射到对外错误码。
func mapChunkErr(err error) error {
	switch err {
	case chunk.ErrEmptyFile:
		return errors.ParamsError("文件大小不能为 0")
	case chunk.ErrFileTooLarge:
		return errors.FileTooLargeError()
	default:
		return errors.ParamsError()
	}
}

// UploadFileByURL 服务端从 URL 下载并上传：限长下载 → 单对象 Put → 建 object/file 行。
// 与分片链路共用：应用/存储启用态核验、目录自动建、后缀与大小校验、对象去重（md5 秒传）、配额占用（ReserveQuota）。
func (s *Upload) UploadFileByURL(ctx core.Context, req *types.UploadFileByURLRequest) (*types.UploadFileByURLResponse, error) {
	userID := uint32(ctx.Auth().UserId)
	if userID == 0 {
		return nil, errors.ParamsError("用户身份异常")
	}
	if req.Url == "" || req.Name == "" {
		return nil, errors.ParamsError()
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(req.Name), "."))
	if !objectkey.IsValidExt(ext) {
		return nil, errors.ParamsError("文件名后缀不合法")
	}

	// 应用配置 + 存储启用态核验（与 PrepareUploadFile 一致）
	cfg, err := s.appConfig.GetAppConfig(ctx, req.App)
	if err != nil {
		if isNotFound(err) {
			return nil, errors.AppNotConfiguredError()
		}
		ctx.Logger().Warnw("msg", "get app config error", "app", req.App, "err", err.Error())
		return nil, errors.GetError()
	}
	if value.Value(cfg.Status) == 0 {
		return nil, errors.AppNotConfiguredError()
	}
	store, err := s.storeRepo.GetStoreByKeyword(ctx, cfg.StoreKeyword)
	if err != nil {
		if isNotFound(err) {
			return nil, errors.StoreDisabledError()
		}
		ctx.Logger().Warnw("msg", "get store by keyword error", "keyword", cfg.StoreKeyword, "err", err.Error())
		return nil, errors.GetError()
	}
	if value.Value(store.Status) == 0 {
		return nil, errors.StoreDisabledError()
	}

	// 解析目录（不存在自动建；后缀前置校验，大小在下载后判定）
	dir, err := s.directory.EnsurePath(ctx, req.App, value.Value(req.DirectoryPath), cfg)
	if err != nil {
		return nil, err
	}
	if !dir.AllowExt(ext) {
		return nil, errors.FileTypeNotAllowedError()
	}

	// 下载限长 = min(目录单文件上限, 应用默认上限, 硬上限)
	maxBytes := uint64(maxURLDownloadSize)
	if dir.MaxSize != entity.MaxSizeUnlimited && uint64(dir.MaxSize)*1024*1024 < maxBytes {
		maxBytes = uint64(dir.MaxSize) * 1024 * 1024
	}
	if cfg.DefaultMaxSize != 0 && cfg.DefaultMaxSize != entity.MaxSizeUnlimited && uint64(cfg.DefaultMaxSize)*1024*1024 < maxBytes {
		maxBytes = uint64(cfg.DefaultMaxSize) * 1024 * 1024
	}

	body, size, md5hex, err := s.downloadURL(ctx, req.Url, maxBytes)
	if err != nil {
		return nil, err
	}
	if !dir.WithinSize(size) {
		return nil, errors.FileTooLargeError()
	}

	// 配额软预检（权威在 finishFile 的 ReserveQuota；used_bytes 可能偏小，仅快速失败）
	if cfg.Quota != nil && *cfg.Quota > 0 && cfg.UsedBytes+size > *cfg.Quota {
		return nil, errors.QuotaExceededError()
	}

	client, ok := s.storeRepo.GetClient(cfg.StoreKeyword)
	if !ok {
		return nil, errors.StoreDisabledError()
	}
	key := objectkey.ObjectName(md5hex, ext)

	// 秒传：对象已存在（大小一致）直接建 file 行，零流量复用
	if obj, gerr := s.objectRepo.GetObjectByMd5(ctx, cfg.StoreKeyword, md5hex); gerr == nil {
		if obj.Size != size {
			return nil, errors.SizeMismatchError()
		}
		if _, ferr := s.finishFile(ctx, &finishFileParams{Obj: obj, App: dir.App, UserID: userID, DirID: dir.Id, Name: req.Name}); ferr != nil {
			return nil, ferr
		}
		return &types.UploadFileByURLResponse{Key: key}, nil
	} else if !isNotFound(gerr) {
		ctx.Logger().Warnw("msg", "get object by md5 error", "keyword", cfg.StoreKeyword, "md5", md5hex, "err", gerr.Error())
		return nil, errors.GetError()
	}

	// 写入对象（单对象直接 Put，不经分片）
	if perr := client.Put(ctx, &repository.PutInput{
		Key:         key,
		Body:        bytes.NewReader(body),
		Size:        int64(size),
		ContentType: objectkey.ContentType(ext),
	}); perr != nil {
		ctx.Logger().Warnw("msg", "put object by url error", "key", key, "err", perr.Error())
		return nil, errors.SystemError()
	}

	// 建 object + file 行（与完成事务同一口径：对象去重、配额占用）
	obj, err := s.ensureObject(ctx, &ensureObjectParams{StoreKeyword: cfg.StoreKeyword, Md5: md5hex, Ext: ext, Size: size})
	if err != nil {
		return nil, err
	}
	if _, ferr := s.finishFile(ctx, &finishFileParams{Obj: obj, App: dir.App, UserID: userID, DirID: dir.Id, Name: req.Name}); ferr != nil {
		return nil, ferr
	}
	return &types.UploadFileByURLResponse{Key: key}, nil
}

// downloadURL 下载远端内容到内存（仅 http/https，限长上限保护），返回（字节, 大小, md5hex）。
func (s *Upload) downloadURL(ctx core.Context, rawURL string, maxBytes uint64) ([]byte, uint64, string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, 0, "", errors.ParamsError("URL 协议不支持")
	}

	req, err := http.NewRequestWithContext(ctx.Ctx(), http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, "", errors.SystemError()
	}
	resp, err := urlDownloadClient.Do(req)
	if err != nil {
		ctx.Logger().Warnw("msg", "download url error", "url", rawURL, "err", err.Error())
		return nil, 0, "", errors.SystemError("下载失败")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, 0, "", errors.SystemError(fmt.Sprintf("下载失败: HTTP %d", resp.StatusCode))
	}

	// Content-Length 快检 + 限量读取（TeeReader 边读边算 md5）
	if maxBytes > 0 && resp.ContentLength > int64(maxBytes) {
		return nil, 0, "", errors.FileTooLargeError()
	}
	hasher := md5.New()
	buf, err := io.ReadAll(io.LimitReader(io.TeeReader(resp.Body, hasher), int64(maxBytes)+1))
	if err != nil {
		ctx.Logger().Warnw("msg", "read download body error", "url", rawURL, "err", err.Error())
		return nil, 0, "", errors.SystemError("下载失败")
	}
	if uint64(len(buf)) > maxBytes {
		return nil, 0, "", errors.FileTooLargeError()
	}
	if len(buf) == 0 {
		return nil, 0, "", errors.ParamsError("文件内容为空")
	}
	return buf, uint64(len(buf)), hex.EncodeToString(hasher.Sum(nil)), nil
}
