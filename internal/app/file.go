package app

import (
	"context"
	"io"

	"github.com/go-kratos/kratos/v2/transport/grpc"
	thttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/limes-cloud/kratosx/model"
	"github.com/spf13/cast"

	"github.com/limes-cloud/resource/api/errors"
	"github.com/limes-cloud/resource/api/file"
	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/service"
	"github.com/limes-cloud/resource/internal/infra/dbs"
	"github.com/limes-cloud/resource/internal/infra/store"
	"github.com/limes-cloud/resource/internal/types"
)

type File struct {
	file.UnimplementedFileServer
	srv *service.File
}

func NewFile() *File {
	return &File{
		srv: service.NewFile(dbs.NewFile(), dbs.NewFile(), dbs.NewDirectory(), store.NewStore),
	}
}

func init() {
	register(func(hs *thttp.Server, gs *grpc.Server) {
		app := NewFile()
		file.RegisterFileHTTPServer(hs, app)
		file.RegisterFileServer(gs, app)

		cr := hs.Route("/")
		cr.GET("/resource/redirect/{store}/{key}", app.Redirect())
		cr.GET("/resource/api/file/download", app.Download())

		cr.POST("/resource/api/chunk_upload", app.ChunkUpload())
		cr.POST("/resource/api/upload", app.Upload())
	})
}

func (s *File) GetUserFile(c context.Context, req *file.GetUserFileRequest) (*file.GetUserFileReply, error) {
	res, err := s.srv.GetUserFile(core.MustContext(c), &types.GetUserFileRequest{
		Directory: req.Directory,
		Id:        req.Id,
		Key:       req.Key,
	})
	if err != nil {
		return nil, err
	}
	return &file.GetUserFileReply{
		Id:          res.Id,
		DirectoryId: res.DirectoryId,
		Name:        res.Name,
		Type:        res.File.Type,
		Size:        res.File.Size,
		Sha:         res.File.Sha,
		Key:         res.File.Key,
		CreatedAt:   uint32(res.CreatedAt),
		UpdatedAt:   uint32(res.UpdatedAt),
	}, nil
}

func (s *File) GetFileBytes(req *file.GetFileBytesRequest, reply file.File_GetFileBytesServer) error {
	return s.srv.GetFileBytes(
		core.MustContext(reply.Context()),
		req.Key,
		func(bytes []byte) error {
			return reply.Send(&file.GetFileBytesReply{
				Data: bytes,
			})
		},
	)
}

func (s *File) ListUserFile(c context.Context, req *file.ListUserFileRequest) (*file.ListUserFileReply, error) {
	list, total, err := s.srv.ListUserFile(core.MustContext(c), &types.ListFileRequest{
		Page:        req.Page,
		PageSize:    req.PageSize,
		Order:       req.Order,
		OrderBy:     req.OrderBy,
		DirectoryId: req.DirectoryId,
		Status:      req.Status,
		Name:        req.Name,
		KeyList:     req.KeyList,
	})
	if err != nil {
		return nil, err
	}

	reply := file.ListUserFileReply{Total: total}
	for _, item := range list {
		reply.List = append(reply.List, &file.ListUserFileReply_File{
			Id:          item.Id,
			DirectoryId: item.DirectoryId,
			Name:        item.Name,
			Type:        item.File.Type,
			Size:        item.File.Size,
			Sha:         item.File.Sha,
			Key:         item.File.Key,
			CreatedAt:   uint32(item.CreatedAt),
			UpdatedAt:   uint32(item.UpdatedAt),
		})
	}
	return &reply, nil
}

func (s *File) PrepareUploadFile(c context.Context, req *file.PrepareUploadFileRequest) (*file.PrepareUploadFileReply, error) {
	if req.DirectoryPath == nil && req.DirectoryId == nil {
		return nil, errors.ParamsError()
	}

	res, err := s.srv.PrepareUploadFile(core.MustContext(c), &types.PrepareUploadFileRequest{
		DirectoryId:   req.DirectoryId,
		DirectoryPath: req.DirectoryPath,
		Name:          req.Name,
		Size:          req.Size,
		Sha:           req.Sha,
		Store:         req.Store,
	})
	if err != nil {
		return nil, err
	}

	return &file.PrepareUploadFileReply{
		Uploaded:     res.Uploaded,
		ChunkSize:    res.ChunkSize,
		ChunkCount:   res.ChunkCount,
		UploadId:     res.UploadId,
		UploadChunks: res.UploadChunks,
		Sha:          res.Sha,
		Key:          res.Key,
	}, nil
}

func (s *File) UploadFile(c context.Context, req *file.UploadFileRequest) (*file.UploadFileReply, error) {
	reply, err := s.srv.UploadFile(core.MustContext(c), &types.UploadFileRequest{
		DirectoryId:   req.DirectoryId,
		DirectoryPath: req.DirectoryPath,
		Data:          req.Data,
		Sha:           req.Sha,
		Name:          req.Name,
		Store:         req.Store,
	})
	if err != nil {
		return nil, err
	}
	return &file.UploadFileReply{
		Sha: reply.Sha,
		Key: reply.Key,
	}, nil
}

func (s *File) UploadFileByURL(c context.Context, req *file.UploadFileByURLRequest) (*file.UploadFileByURLReply, error) {
	reply, err := s.srv.UploadFileByURL(core.MustContext(c), &types.UploadFileByURLRequest{
		DirectoryPath: req.DirectoryPath,
		Store:         req.Store,
		URL:           req.Url,
		Name:          req.Name,
	})
	if err != nil {
		return nil, err
	}
	return &file.UploadFileByURLReply{
		Sha: reply.Sha,
		Key: reply.Key,
	}, nil
}

func (s *File) UploadChunkFile(c context.Context, req *file.UploadChunkFileRequest) (*file.UploadFileReply, error) {
	reply, err := s.srv.UploadChunkFile(core.MustContext(c), &types.UploadChunkFileRequest{
		UploadId: req.UploadId,
		Index:    req.Index,
		Data:     req.Data,
	})
	if err != nil {
		return nil, err
	}
	return &file.UploadFileReply{
		Sha: reply.Sha,
		Key: reply.Key,
	}, nil
}

func (s *File) UpdateUserFile(c context.Context, req *file.UpdateUserFileRequest) (*file.UpdateUserFileReply, error) {
	if err := s.srv.UpdateUserFile(core.MustContext(c), &entity.UserFile{
		BaseTenantUserModel: model.BaseTenantUserModel{Id: req.Id},
		DirectoryId:         req.DirectoryId,
		Name:                req.Name,
	}); err != nil {
		return nil, err
	}
	return &file.UpdateUserFileReply{}, nil
}

func (s *File) DeleteUserFile(c context.Context, req *file.DeleteUserFileRequest) (*file.DeleteUserFileReply, error) {
	total, err := s.srv.DeleteUserFile(core.MustContext(c), req.Ids)
	if err != nil {
		return nil, err
	}
	return &file.DeleteUserFileReply{Total: total}, nil
}

func (s *File) ChunkUpload() thttp.HandlerFunc {
	return func(ctx thttp.Context) error {
		var in file.UploadChunkFileRequest

		in.UploadId = ctx.Request().FormValue("uploadId")
		in.Index = cast.ToUint32(ctx.Request().FormValue("index"))
		fileByte, _, err := ctx.Request().FormFile("data")
		if err != nil {
			return errors.UploadFileError(err.Error())
		}

		in.Data, err = io.ReadAll(fileByte)
		if err != nil {
			return errors.UploadFileError(err.Error())
		}
		if in.UploadId == "" || int(in.Index) <= 0 || len(in.Data) == 0 {
			return errors.ParamsError()
		}

		h := ctx.Middleware(func(ctx context.Context, req any) (any, error) {
			return s.UploadChunkFile(ctx, req.(*file.UploadChunkFileRequest))
		})

		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*file.UploadFileReply)
		return ctx.Result(200, reply)
	}
}

func (s *File) Upload() thttp.HandlerFunc {
	return func(ctx thttp.Context) error {
		var in file.UploadFileRequest

		if ctx.Request().FormValue("directoryId") != "" {
			v := cast.ToUint32(ctx.Request().FormValue("directoryId"))
			in.DirectoryId = &v
		}
		if ctx.Request().FormValue("directoryPath") != "" {
			v := ctx.Request().FormValue("directoryPath")
			in.DirectoryPath = &v
		}
		in.Sha = ctx.Request().FormValue("sha")
		in.Name = ctx.Request().FormValue("name")
		fileByte, _, err := ctx.Request().FormFile("data")
		if err != nil {
			return errors.UploadFileError(err.Error())
		}

		in.Data, err = io.ReadAll(fileByte)
		if err != nil {
			return errors.UploadFileError(err.Error())
		}

		h := ctx.Middleware(func(ctx context.Context, req any) (any, error) {
			return s.UploadFile(ctx, req.(*file.UploadFileRequest))
		})

		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*file.UploadFileReply)
		return ctx.Result(200, reply)
	}
}

func (s *File) Download() thttp.HandlerFunc {
	return func(ctx thttp.Context) error {
		key := ctx.Request().URL.Query().Get("key")
		if key == "" {
			return errors.ParamsError()
		}
		ctx.Response().Header().Set("Content-Type", "application/octet-stream")
		return s.srv.GetFileBytes(core.MustContext(ctx.Request().Context()), key, func(b []byte) error {
			_, err := ctx.Response().Write(b)
			return err
		})
	}
}

func (s *File) Redirect() thttp.HandlerFunc {
	return func(ctx thttp.Context) error {
		var req file.StaticFileRequest
		if err := ctx.BindQuery(&req); err != nil {
			return err
		}
		if err := ctx.BindVars(&req); err != nil {
			return err
		}

		st, err := store.NewStore(req.Store)
		if err != nil {
			return err
		}

		queryEncode := st.ParserQuery(&types.ParserQuery{
			Expire:   req.Expire,
			Sign:     req.Sign,
			Width:    req.Width,
			Height:   req.Height,
			Mode:     req.Mode,
			Download: req.Download,
			SaveName: req.SaveName,
		})

		url := st.Config().ServerURL + "/" + req.Store + "/" + req.Key + "?" + queryEncode
		ctx.Response().Header().Set("Location", url)
		ctx.Response().WriteHeader(301)
		return nil
	}
}
