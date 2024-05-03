// Code generated by protoc-gen-go-http. DO NOT EDIT.
// versions:
// - protoc-gen-go-http v2.7.0
// - protoc             v4.24.4
// source: resource_export_service.proto

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

const OperationServiceAddExport = "/export.Service/AddExport"
const OperationServiceAddExportExcel = "/export.Service/AddExportExcel"
const OperationServiceDeleteExport = "/export.Service/DeleteExport"
const OperationServicePageExport = "/export.Service/PageExport"

type ServiceHTTPServer interface {
	AddExport(context.Context, *AddExportRequest) (*AddExportReply, error)
	AddExportExcel(context.Context, *AddExportExcelRequest) (*AddExportExcelReply, error)
	DeleteExport(context.Context, *DeleteExportRequest) (*emptypb.Empty, error)
	PageExport(context.Context, *PageExportRequest) (*PageExportReply, error)
}

func RegisterServiceHTTPServer(s *http.Server, srv ServiceHTTPServer) {
	r := s.Route("/")
	r.GET("/resource/v1/exports", _Service_PageExport0_HTTP_Handler(srv))
	r.POST("/resource/v1/export", _Service_AddExport0_HTTP_Handler(srv))
	r.POST("/resource/v1/export/excel", _Service_AddExportExcel0_HTTP_Handler(srv))
	r.DELETE("/resource/v1/export", _Service_DeleteExport0_HTTP_Handler(srv))
}

func _Service_PageExport0_HTTP_Handler(srv ServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in PageExportRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationServicePageExport)
		h := ctx.Middleware(func(ctx context.Context, req any) (any, error) {
			return srv.PageExport(ctx, req.(*PageExportRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*PageExportReply)
		return ctx.Result(200, reply)
	}
}

func _Service_AddExport0_HTTP_Handler(srv ServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in AddExportRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationServiceAddExport)
		h := ctx.Middleware(func(ctx context.Context, req any) (any, error) {
			return srv.AddExport(ctx, req.(*AddExportRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*AddExportReply)
		return ctx.Result(200, reply)
	}
}

func _Service_AddExportExcel0_HTTP_Handler(srv ServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in AddExportExcelRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationServiceAddExportExcel)
		h := ctx.Middleware(func(ctx context.Context, req any) (any, error) {
			return srv.AddExportExcel(ctx, req.(*AddExportExcelRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*AddExportExcelReply)
		return ctx.Result(200, reply)
	}
}

func _Service_DeleteExport0_HTTP_Handler(srv ServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in DeleteExportRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationServiceDeleteExport)
		h := ctx.Middleware(func(ctx context.Context, req any) (any, error) {
			return srv.DeleteExport(ctx, req.(*DeleteExportRequest))
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
	AddExport(ctx context.Context, req *AddExportRequest, opts ...http.CallOption) (rsp *AddExportReply, err error)
	AddExportExcel(ctx context.Context, req *AddExportExcelRequest, opts ...http.CallOption) (rsp *AddExportExcelReply, err error)
	DeleteExport(ctx context.Context, req *DeleteExportRequest, opts ...http.CallOption) (rsp *emptypb.Empty, err error)
	PageExport(ctx context.Context, req *PageExportRequest, opts ...http.CallOption) (rsp *PageExportReply, err error)
}

type ServiceHTTPClientImpl struct {
	cc *http.Client
}

func NewServiceHTTPClient(client *http.Client) ServiceHTTPClient {
	return &ServiceHTTPClientImpl{client}
}

func (c *ServiceHTTPClientImpl) AddExport(ctx context.Context, in *AddExportRequest, opts ...http.CallOption) (*AddExportReply, error) {
	var out AddExportReply
	pattern := "/resource/v1/export"
	path := binding.EncodeURL(pattern, in, false)
	opts = append(opts, http.Operation(OperationServiceAddExport))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "POST", path, in, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, err
}

func (c *ServiceHTTPClientImpl) AddExportExcel(ctx context.Context, in *AddExportExcelRequest, opts ...http.CallOption) (*AddExportExcelReply, error) {
	var out AddExportExcelReply
	pattern := "/resource/v1/export/excel"
	path := binding.EncodeURL(pattern, in, false)
	opts = append(opts, http.Operation(OperationServiceAddExportExcel))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "POST", path, in, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, err
}

func (c *ServiceHTTPClientImpl) DeleteExport(ctx context.Context, in *DeleteExportRequest, opts ...http.CallOption) (*emptypb.Empty, error) {
	var out emptypb.Empty
	pattern := "/resource/v1/export"
	path := binding.EncodeURL(pattern, in, true)
	opts = append(opts, http.Operation(OperationServiceDeleteExport))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "DELETE", path, nil, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, err
}

func (c *ServiceHTTPClientImpl) PageExport(ctx context.Context, in *PageExportRequest, opts ...http.CallOption) (*PageExportReply, error) {
	var out PageExportReply
	pattern := "/resource/v1/exports"
	path := binding.EncodeURL(pattern, in, true)
	opts = append(opts, http.Operation(OperationServicePageExport))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "GET", path, nil, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, err
}
