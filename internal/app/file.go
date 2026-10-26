package app

import (
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/go-kratos/kratos/v2/transport/grpc"
	khttp "github.com/go-kratos/kratos/v2/transport/http"

	reserrors "github.com/limes-cloud/resource/api/errors"
	fileapi "github.com/limes-cloud/resource/api/file"
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/service"
	"github.com/limes-cloud/resource/internal/infra/dbs"
	"github.com/limes-cloud/resource/internal/types"
)

// fileHandler 实现 file.FileServiceHTTPServer / file.FileServiceServer（同一批方法签名）。
type fileHandler struct {
	fileapi.UnimplementedFileServiceServer
	file   *service.File
	upload *service.Upload
}

// init 自注册 FileService（HTTP 与 gRPC）。
func init() {
	register(func(hs *khttp.Server, gs *grpc.Server) {
		dir := service.NewDirectory(dbs.NewDirectory(), dbs.NewAppConfig())
		h := &fileHandler{
			file:   service.NewFile(dbs.NewFile(), dbs.NewObject(), dbs.NewStore(), dbs.NewAppConfig(), dir),
			upload: service.NewUpload(dbs.NewUpload(), dbs.NewObject(), dbs.NewFile(), dbs.NewStore(), dbs.NewAppConfig(), dir),
		}
		fileapi.RegisterFileServiceHTTPServer(hs, h)
		fileapi.RegisterFileServiceServer(gs, h)
		// 302 快速展示：<img src=".../object-by-key?key=xxx"> 直接跟随重定向到真实地址
		// （对象按 md5 全局内容寻址，无需 app 定位存储；仅读操作，无副作用）
		hs.Route("/resource/api/file").GET("/object-by-key", h.redirectByKey)
		// 服务端读流：存储（如 OSS）未开 CORS 时 <img> 可显示但 fetch 取不到字节，经此接口原样返回内容
		hs.Route("/resource/api/file").GET("/read-by-key", h.readByKey)
	})
}

// GetUserFile 取单个文件（响应含完整 url 与 size）。
func (h *fileHandler) GetUserFile(ctx context.Context, req *fileapi.GetUserFileRequest) (*fileapi.GetUserFileResponse, error) {
	item, err := h.file.GetFile(core.MustContext(ctx), uint32(req.Id))
	if err != nil {
		return nil, err
	}
	return &fileapi.GetUserFileResponse{File: toFileItem(item)}, nil
}

// ListUserFile 我的文件分页（app/user_id 取自登录态，忽略请求体，防越权列举他人文件）。
func (h *fileHandler) ListUserFile(ctx context.Context, req *fileapi.ListUserFileRequest) (*fileapi.ListUserFileResponse, error) {
	app, err := currentApp(ctx)
	if err != nil {
		return nil, err
	}
	list, total, err := h.file.ListFileByUser(core.MustContext(ctx), app, uint32(core.MustContext(ctx).Auth().UserId), &types.PageRequest{Page: req.Page, PageSize: req.PageSize})
	if err != nil {
		return nil, err
	}
	return &fileapi.ListUserFileResponse{List: toFileItems(list), Total: total}, nil
}

// ListFile 目录内文件分页。
func (h *fileHandler) ListFile(ctx context.Context, req *fileapi.ListFileRequest) (*fileapi.ListFileResponse, error) {
	app, err := currentApp(ctx)
	if err != nil {
		return nil, err
	}
	list, total, err := h.file.ListFileByDirectory(core.MustContext(ctx), app, uint32(req.DirectoryId), &types.PageRequest{Page: req.Page, PageSize: req.PageSize})
	if err != nil {
		return nil, err
	}
	return &fileapi.ListFileResponse{List: toFileItems(list), Total: total}, nil
}

// UploadFileByURL 服务端从 URL 下载并上传（生成内容落库；应用标识取自 X-App-Key）。
func (h *fileHandler) UploadFileByURL(ctx context.Context, req *fileapi.UploadFileByURLRequest) (*fileapi.UploadFileByURLResponse, error) {
	app, err := currentApp(ctx)
	if err != nil {
		return nil, err
	}
	reply, err := h.upload.UploadFileByURL(core.MustContext(ctx), &types.UploadFileByURLRequest{
		App:           app,
		Url:           req.Url,
		Name:          req.Name,
		DirectoryPath: req.DirectoryPath,
	})
	if err != nil {
		return nil, err
	}
	return &fileapi.UploadFileByURLResponse{Key: reply.Key}, nil
}

// GetFileURLByKey 按对象 key 取可访问地址。
func (h *fileHandler) GetFileURLByKey(ctx context.Context, req *fileapi.GetFileURLByKeyRequest) (*fileapi.GetFileURLByKeyReply, error) {
	reply, err := h.file.GetFileURLByKey(core.MustContext(ctx), req.GetKey())
	if err != nil {
		return nil, err
	}
	return &fileapi.GetFileURLByKeyReply{Url: reply.Url}, nil
}

// redirectByKey 按对象 key 取可访问地址（图像快速展示 302 重定向；仅 key 即可，无需 app）。
func (h *fileHandler) redirectByKey(ctx khttp.Context) error {
	key := ctx.Query().Get("key")
	if key == "" {
		return reserrors.ParamsError("缺少对象key")
	}
	reply, err := h.file.GetFileURLByKey(core.MustContext(ctx), key)
	if err != nil {
		return err
	}
	w := ctx.Response()
	w.Header().Set("Location", reply.Url)
	w.WriteHeader(http.StatusFound)
	return nil
}

// readByKey 按对象 key 服务端读取并返回文件内容（同源返回字节流，规避对象存储 CORS 限制）。
func (h *fileHandler) readByKey(ctx khttp.Context) error {
	key := ctx.Query().Get("key")
	if key == "" {
		return reserrors.ParamsError("缺少对象key")
	}
	rc, contentType, err := h.file.ReadFileByKey(core.MustContext(ctx), key)
	if err != nil {
		return err
	}
	defer rc.Close()
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w := ctx.Response()
	w.Header().Set("Content-Type", contentType)
	if _, err := io.Copy(w, rc); err != nil {
		return reserrors.SystemError()
	}
	return nil
}

// GetUploadChunkSize 取分片大小与片数（纯函数）。
func (h *fileHandler) GetUploadChunkSize(ctx context.Context, req *fileapi.GetUploadChunkSizeRequest) (*fileapi.GetUploadChunkSizeResponse, error) {
	total, err := strconv.ParseUint(req.TotalSize, 10, 64)
	if err != nil {
		return nil, reserrors.ParamsError()
	}
	reply, err := h.upload.GetUploadChunkSize(core.MustContext(ctx), &types.GetUploadChunkSizeRequest{TotalSize: total})
	if err != nil {
		return nil, err
	}
	return &fileapi.GetUploadChunkSizeResponse{ChunkSize: reply.ChunkSize, ChunkCount: reply.ChunkCount}, nil
}

// PrepareUploadFile 预上传：秒传判定 / 召回 / 建会话。
func (h *fileHandler) PrepareUploadFile(ctx context.Context, req *fileapi.PrepareUploadFileRequest) (*fileapi.PrepareUploadFileResponse, error) {
	size, err := strconv.ParseUint(req.Size, 10, 64)
	if err != nil {
		return nil, reserrors.ParamsError()
	}
	app, err := currentApp(ctx)
	if err != nil {
		return nil, err
	}
	reply, err := h.upload.PrepareUploadFile(core.MustContext(ctx), &types.PrepareUploadFileRequest{
		App:  app,
		Path: req.Path,
		Name: req.Name,
		Ext:  req.Ext,
		Md5:  req.Md5,
		Size: size,
	})
	if err != nil {
		return nil, err
	}
	return &fileapi.PrepareUploadFileResponse{
		UploadId:     reply.UploadId,
		Md5:          reply.Md5,
		ChunkSize:    reply.ChunkSize,
		ChunkCount:   reply.ChunkCount,
		UploadChunks: reply.UploadedParts,
		Skipped:      reply.Skipped,
		Url:          reply.Url,
		Key:          reply.Key,
	}, nil
}

// GetUploadPartURLs 按批取分片上传预签名地址。
func (h *fileHandler) GetUploadPartURLs(ctx context.Context, req *fileapi.GetUploadPartURLsRequest) (*fileapi.GetUploadPartURLsResponse, error) {
	reply, err := h.upload.GetUploadPartURLs(core.MustContext(ctx), &types.GetUploadPartURLsRequest{
		UploadId:    req.UploadId,
		PartNumbers: req.PartNumbers,
	})
	if err != nil {
		return nil, err
	}
	urls := make([]*fileapi.PartURL, 0, len(reply.Urls))
	for _, u := range reply.Urls {
		urls = append(urls, &fileapi.PartURL{PartNumber: u.PartNumber, Url: u.Url})
	}
	return &fileapi.GetUploadPartURLsResponse{Urls: urls}, nil
}

// CompleteUpload 完成分片上传。
func (h *fileHandler) CompleteUpload(ctx context.Context, req *fileapi.CompleteUploadRequest) (*fileapi.CompleteUploadResponse, error) {
	reply, err := h.upload.CompleteUpload(core.MustContext(ctx), &types.CompleteUploadRequest{UploadId: req.UploadId})
	if err != nil {
		return nil, err
	}
	return &fileapi.CompleteUploadResponse{Url: reply.Url, Key: reply.Key}, nil
}

// UpdateUserFile 改名 / 移动。
func (h *fileHandler) UpdateUserFile(ctx context.Context, req *fileapi.UpdateUserFileRequest) (*fileapi.UpdateUserFileResponse, error) {
	// 文件名文法 proto 表达不了，传了才校验
	if req.Name != nil {
		if err := entity.ValidateName(*req.Name); err != nil {
			return nil, reserrors.ParamsError(err.Error())
		}
	}
	app, err := currentApp(ctx)
	if err != nil {
		return nil, err
	}
	item, err := h.file.UpdateFile(core.MustContext(ctx), &types.UpdateFileRequest{
		App:         app,
		Id:          uint32(req.Id),
		Name:        req.Name,
		DirectoryId: req.DirectoryId,
	})
	if err != nil {
		return nil, err
	}
	return &fileapi.UpdateUserFileResponse{File: toFileItem(item)}, nil
}

// DeleteUserFile 删除文件（触发对象回收）。
func (h *fileHandler) DeleteUserFile(ctx context.Context, req *fileapi.DeleteUserFileRequest) (*fileapi.DeleteUserFileResponse, error) {
	app, err := currentApp(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.file.DeleteFile(core.MustContext(ctx), app, uint32(req.Id)); err != nil {
		return nil, err
	}
	return &fileapi.DeleteUserFileResponse{Id: req.Id}, nil
}

// toFileItems 批量转 proto 列表项。
func toFileItems(items []*types.FileItem) []*fileapi.FileItem {
	out := make([]*fileapi.FileItem, 0, len(items))
	for _, it := range items {
		out = append(out, toFileItem(it))
	}
	return out
}

// toFileItem 领域 FileItem → proto FileItem（size uint32→string、时间戳→uint32）。
func toFileItem(it *types.FileItem) *fileapi.FileItem {
	return &fileapi.FileItem{
		Id:          uint32(it.Id),
		App:         it.App,
		UserId:      uint32(it.UserId),
		DirectoryId: uint32(it.DirectoryId),
		Name:        it.Name,
		ObjectId:    uint32(it.ObjectId),
		Size:        strconv.FormatUint(it.Size, 10),
		Ext:         it.Ext,
		Url:         it.Url,
		Key:         it.Key,
		CreatedAt:   uint32(it.CreatedAt),
	}
}
