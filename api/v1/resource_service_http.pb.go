// Code generated by protoc-gen-go-http. DO NOT EDIT.
// versions:
// - protoc-gen-go-http v2.7.0
// - protoc             v4.24.4
// source: resource_service.proto

package v1

import (
	context "context"

	http "github.com/go-kratos/kratos/v2/transport/http"
	binding "github.com/go-kratos/kratos/v2/transport/http/binding"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the kratos package it is being compiled against.
var _ = new(context.Context)
var _ = binding.EncodeURL

const _ = http.SupportPackageIsVersion1

const OperationServiceAddDirectory = "/v1.Service/AddDirectory"
const OperationServiceDeleteDirectory = "/v1.Service/DeleteDirectory"
const OperationServiceDeleteFile = "/v1.Service/DeleteFile"
const OperationServiceGetDirectory = "/v1.Service/GetDirectory"
const OperationServicePageFile = "/v1.Service/PageFile"
const OperationServicePrepareUploadFile = "/v1.Service/PrepareUploadFile"
const OperationServiceUpdateDirectory = "/v1.Service/UpdateDirectory"
const OperationServiceUpdateFile = "/v1.Service/UpdateFile"
const OperationServiceUploadFile = "/v1.Service/UploadFile"

type ServiceHTTPServer interface {
	AddDirectory(context.Context, *AddDirectoryRequest) (*Directory, error)
	DeleteDirectory(context.Context, *DeleteDirectoryRequest) (*emptypb.Empty, error)
	DeleteFile(context.Context, *DeleteFileRequest) (*emptypb.Empty, error)
	GetDirectory(context.Context, *GetDirectoryRequest) (*GetDirectoryReply, error)
	PageFile(context.Context, *PageFileRequest) (*PageFileReply, error)
	PrepareUploadFile(context.Context, *PrepareUploadFileRequest) (*PrepareUploadFileReply, error)
	UpdateDirectory(context.Context, *UpdateDirectoryRequest) (*emptypb.Empty, error)
	UpdateFile(context.Context, *UpdateFileRequest) (*emptypb.Empty, error)
	UploadFile(context.Context, *UploadFileRequest) (*UploadFileReply, error)
}

func RegisterServiceHTTPServer(s *http.Server, srv ServiceHTTPServer) {
	r := s.Route("/")
	r.GET("/resource/v1/directory", _Service_GetDirectory0_HTTP_Handler(srv))
	r.POST("/resource/v1/directory", _Service_AddDirectory0_HTTP_Handler(srv))
	r.PUT("/resource/v1/directory", _Service_UpdateDirectory0_HTTP_Handler(srv))
	r.DELETE("/resource/v1/directory", _Service_DeleteDirectory0_HTTP_Handler(srv))
	r.POST("/resource/v1/upload/prepare", _Service_PrepareUploadFile0_HTTP_Handler(srv))
	r.POST("/resource/v1/upload", _Service_UploadFile0_HTTP_Handler(srv))
	r.GET("/resource/v1/files", _Service_PageFile0_HTTP_Handler(srv))
	r.PUT("/resource/v1/file", _Service_UpdateFile0_HTTP_Handler(srv))
	r.POST("/resource/v1/file", _Service_DeleteFile0_HTTP_Handler(srv))
}

func _Service_GetDirectory0_HTTP_Handler(srv ServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in GetDirectoryRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationServiceGetDirectory)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.GetDirectory(ctx, req.(*GetDirectoryRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*GetDirectoryReply)
		return ctx.Result(200, reply.List)
	}
}

func _Service_AddDirectory0_HTTP_Handler(srv ServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in AddDirectoryRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationServiceAddDirectory)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.AddDirectory(ctx, req.(*AddDirectoryRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*Directory)
		return ctx.Result(200, reply)
	}
}

func _Service_UpdateDirectory0_HTTP_Handler(srv ServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in UpdateDirectoryRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationServiceUpdateDirectory)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.UpdateDirectory(ctx, req.(*UpdateDirectoryRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*emptypb.Empty)
		return ctx.Result(200, reply)
	}
}

func _Service_DeleteDirectory0_HTTP_Handler(srv ServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in DeleteDirectoryRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationServiceDeleteDirectory)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.DeleteDirectory(ctx, req.(*DeleteDirectoryRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*emptypb.Empty)
		return ctx.Result(200, reply)
	}
}

func _Service_PrepareUploadFile0_HTTP_Handler(srv ServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in PrepareUploadFileRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationServicePrepareUploadFile)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.PrepareUploadFile(ctx, req.(*PrepareUploadFileRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*PrepareUploadFileReply)
		return ctx.Result(200, reply)
	}
}

func _Service_UploadFile0_HTTP_Handler(srv ServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in UploadFileRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationServiceUploadFile)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.UploadFile(ctx, req.(*UploadFileRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*UploadFileReply)
		return ctx.Result(200, reply)
	}
}

func _Service_PageFile0_HTTP_Handler(srv ServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in PageFileRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationServicePageFile)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.PageFile(ctx, req.(*PageFileRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*PageFileReply)
		return ctx.Result(200, reply)
	}
}

func _Service_UpdateFile0_HTTP_Handler(srv ServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in UpdateFileRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationServiceUpdateFile)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.UpdateFile(ctx, req.(*UpdateFileRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*emptypb.Empty)
		return ctx.Result(200, reply)
	}
}

func _Service_DeleteFile0_HTTP_Handler(srv ServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in DeleteFileRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationServiceDeleteFile)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.DeleteFile(ctx, req.(*DeleteFileRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*emptypb.Empty)
		return ctx.Result(200, reply)
	}
}

type ServiceHTTPClient interface {
	AddDirectory(ctx context.Context, req *AddDirectoryRequest, opts ...http.CallOption) (rsp *Directory, err error)
	DeleteDirectory(ctx context.Context, req *DeleteDirectoryRequest, opts ...http.CallOption) (rsp *emptypb.Empty, err error)
	DeleteFile(ctx context.Context, req *DeleteFileRequest, opts ...http.CallOption) (rsp *emptypb.Empty, err error)
	GetDirectory(ctx context.Context, req *GetDirectoryRequest, opts ...http.CallOption) (rsp *GetDirectoryReply, err error)
	PageFile(ctx context.Context, req *PageFileRequest, opts ...http.CallOption) (rsp *PageFileReply, err error)
	PrepareUploadFile(ctx context.Context, req *PrepareUploadFileRequest, opts ...http.CallOption) (rsp *PrepareUploadFileReply, err error)
	UpdateDirectory(ctx context.Context, req *UpdateDirectoryRequest, opts ...http.CallOption) (rsp *emptypb.Empty, err error)
	UpdateFile(ctx context.Context, req *UpdateFileRequest, opts ...http.CallOption) (rsp *emptypb.Empty, err error)
	UploadFile(ctx context.Context, req *UploadFileRequest, opts ...http.CallOption) (rsp *UploadFileReply, err error)
}

type ServiceHTTPClientImpl struct {
	cc *http.Client
}

func NewServiceHTTPClient(client *http.Client) ServiceHTTPClient {
	return &ServiceHTTPClientImpl{client}
}

func (c *ServiceHTTPClientImpl) AddDirectory(ctx context.Context, in *AddDirectoryRequest, opts ...http.CallOption) (*Directory, error) {
	var out Directory
	pattern := "/resource/v1/directory"
	path := binding.EncodeURL(pattern, in, false)
	opts = append(opts, http.Operation(OperationServiceAddDirectory))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "POST", path, in, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, err
}

func (c *ServiceHTTPClientImpl) DeleteDirectory(ctx context.Context, in *DeleteDirectoryRequest, opts ...http.CallOption) (*emptypb.Empty, error) {
	var out emptypb.Empty
	pattern := "/resource/v1/directory"
	path := binding.EncodeURL(pattern, in, true)
	opts = append(opts, http.Operation(OperationServiceDeleteDirectory))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "DELETE", path, nil, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, err
}

func (c *ServiceHTTPClientImpl) DeleteFile(ctx context.Context, in *DeleteFileRequest, opts ...http.CallOption) (*emptypb.Empty, error) {
	var out emptypb.Empty
	pattern := "/resource/v1/file"
	path := binding.EncodeURL(pattern, in, false)
	opts = append(opts, http.Operation(OperationServiceDeleteFile))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "POST", path, in, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, err
}

func (c *ServiceHTTPClientImpl) GetDirectory(ctx context.Context, in *GetDirectoryRequest, opts ...http.CallOption) (*GetDirectoryReply, error) {
	var out GetDirectoryReply
	pattern := "/resource/v1/directory"
	path := binding.EncodeURL(pattern, in, true)
	opts = append(opts, http.Operation(OperationServiceGetDirectory))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "GET", path, nil, &out.List, opts...)
	if err != nil {
		return nil, err
	}
	return &out, err
}

func (c *ServiceHTTPClientImpl) PageFile(ctx context.Context, in *PageFileRequest, opts ...http.CallOption) (*PageFileReply, error) {
	var out PageFileReply
	pattern := "/resource/v1/files"
	path := binding.EncodeURL(pattern, in, true)
	opts = append(opts, http.Operation(OperationServicePageFile))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "GET", path, nil, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, err
}

func (c *ServiceHTTPClientImpl) PrepareUploadFile(ctx context.Context, in *PrepareUploadFileRequest, opts ...http.CallOption) (*PrepareUploadFileReply, error) {
	var out PrepareUploadFileReply
	pattern := "/resource/v1/upload/prepare"
	path := binding.EncodeURL(pattern, in, false)
	opts = append(opts, http.Operation(OperationServicePrepareUploadFile))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "POST", path, in, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, err
}

func (c *ServiceHTTPClientImpl) UpdateDirectory(ctx context.Context, in *UpdateDirectoryRequest, opts ...http.CallOption) (*emptypb.Empty, error) {
	var out emptypb.Empty
	pattern := "/resource/v1/directory"
	path := binding.EncodeURL(pattern, in, false)
	opts = append(opts, http.Operation(OperationServiceUpdateDirectory))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "PUT", path, in, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, err
}

func (c *ServiceHTTPClientImpl) UpdateFile(ctx context.Context, in *UpdateFileRequest, opts ...http.CallOption) (*emptypb.Empty, error) {
	var out emptypb.Empty
	pattern := "/resource/v1/file"
	path := binding.EncodeURL(pattern, in, false)
	opts = append(opts, http.Operation(OperationServiceUpdateFile))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "PUT", path, in, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, err
}

func (c *ServiceHTTPClientImpl) UploadFile(ctx context.Context, in *UploadFileRequest, opts ...http.CallOption) (*UploadFileReply, error) {
	var out UploadFileReply
	pattern := "/resource/v1/upload"
	path := binding.EncodeURL(pattern, in, false)
	opts = append(opts, http.Operation(OperationServiceUploadFile))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "POST", path, in, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, err
}
