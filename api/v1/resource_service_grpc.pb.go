// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v4.24.4
// source: resource_service.proto

package v1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	Service_GetDirectory_FullMethodName      = "/v1.Service/GetDirectory"
	Service_AddDirectory_FullMethodName      = "/v1.Service/AddDirectory"
	Service_UpdateDirectory_FullMethodName   = "/v1.Service/UpdateDirectory"
	Service_DeleteDirectory_FullMethodName   = "/v1.Service/DeleteDirectory"
	Service_PrepareUploadFile_FullMethodName = "/v1.Service/PrepareUploadFile"
	Service_UploadFile_FullMethodName        = "/v1.Service/UploadFile"
	Service_PageFile_FullMethodName          = "/v1.Service/PageFile"
	Service_UpdateFile_FullMethodName        = "/v1.Service/UpdateFile"
	Service_DeleteFile_FullMethodName        = "/v1.Service/DeleteFile"
)

// ServiceClient is the client API for Service service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ServiceClient interface {
	GetDirectory(ctx context.Context, in *GetDirectoryRequest, opts ...grpc.CallOption) (*GetDirectoryReply, error)
	AddDirectory(ctx context.Context, in *AddDirectoryRequest, opts ...grpc.CallOption) (*Directory, error)
	UpdateDirectory(ctx context.Context, in *UpdateDirectoryRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	DeleteDirectory(ctx context.Context, in *DeleteDirectoryRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	PrepareUploadFile(ctx context.Context, in *PrepareUploadFileRequest, opts ...grpc.CallOption) (*PrepareUploadFileReply, error)
	UploadFile(ctx context.Context, in *UploadFileRequest, opts ...grpc.CallOption) (*UploadFileReply, error)
	PageFile(ctx context.Context, in *PageFileRequest, opts ...grpc.CallOption) (*PageFileReply, error)
	UpdateFile(ctx context.Context, in *UpdateFileRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	DeleteFile(ctx context.Context, in *DeleteFileRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type serviceClient struct {
	cc grpc.ClientConnInterface
}

func NewServiceClient(cc grpc.ClientConnInterface) ServiceClient {
	return &serviceClient{cc}
}

func (c *serviceClient) GetDirectory(ctx context.Context, in *GetDirectoryRequest, opts ...grpc.CallOption) (*GetDirectoryReply, error) {
	out := new(GetDirectoryReply)
	err := c.cc.Invoke(ctx, Service_GetDirectory_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceClient) AddDirectory(ctx context.Context, in *AddDirectoryRequest, opts ...grpc.CallOption) (*Directory, error) {
	out := new(Directory)
	err := c.cc.Invoke(ctx, Service_AddDirectory_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceClient) UpdateDirectory(ctx context.Context, in *UpdateDirectoryRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, Service_UpdateDirectory_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceClient) DeleteDirectory(ctx context.Context, in *DeleteDirectoryRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, Service_DeleteDirectory_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceClient) PrepareUploadFile(ctx context.Context, in *PrepareUploadFileRequest, opts ...grpc.CallOption) (*PrepareUploadFileReply, error) {
	out := new(PrepareUploadFileReply)
	err := c.cc.Invoke(ctx, Service_PrepareUploadFile_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceClient) UploadFile(ctx context.Context, in *UploadFileRequest, opts ...grpc.CallOption) (*UploadFileReply, error) {
	out := new(UploadFileReply)
	err := c.cc.Invoke(ctx, Service_UploadFile_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceClient) PageFile(ctx context.Context, in *PageFileRequest, opts ...grpc.CallOption) (*PageFileReply, error) {
	out := new(PageFileReply)
	err := c.cc.Invoke(ctx, Service_PageFile_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceClient) UpdateFile(ctx context.Context, in *UpdateFileRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, Service_UpdateFile_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceClient) DeleteFile(ctx context.Context, in *DeleteFileRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, Service_DeleteFile_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ServiceServer is the server API for Service service.
// All implementations must embed UnimplementedServiceServer
// for forward compatibility
type ServiceServer interface {
	GetDirectory(context.Context, *GetDirectoryRequest) (*GetDirectoryReply, error)
	AddDirectory(context.Context, *AddDirectoryRequest) (*Directory, error)
	UpdateDirectory(context.Context, *UpdateDirectoryRequest) (*emptypb.Empty, error)
	DeleteDirectory(context.Context, *DeleteDirectoryRequest) (*emptypb.Empty, error)
	PrepareUploadFile(context.Context, *PrepareUploadFileRequest) (*PrepareUploadFileReply, error)
	UploadFile(context.Context, *UploadFileRequest) (*UploadFileReply, error)
	PageFile(context.Context, *PageFileRequest) (*PageFileReply, error)
	UpdateFile(context.Context, *UpdateFileRequest) (*emptypb.Empty, error)
	DeleteFile(context.Context, *DeleteFileRequest) (*emptypb.Empty, error)
	mustEmbedUnimplementedServiceServer()
}

// UnimplementedServiceServer must be embedded to have forward compatible implementations.
type UnimplementedServiceServer struct {
}

func (UnimplementedServiceServer) GetDirectory(context.Context, *GetDirectoryRequest) (*GetDirectoryReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetDirectory not implemented")
}
func (UnimplementedServiceServer) AddDirectory(context.Context, *AddDirectoryRequest) (*Directory, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddDirectory not implemented")
}
func (UnimplementedServiceServer) UpdateDirectory(context.Context, *UpdateDirectoryRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateDirectory not implemented")
}
func (UnimplementedServiceServer) DeleteDirectory(context.Context, *DeleteDirectoryRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteDirectory not implemented")
}
func (UnimplementedServiceServer) PrepareUploadFile(context.Context, *PrepareUploadFileRequest) (*PrepareUploadFileReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PrepareUploadFile not implemented")
}
func (UnimplementedServiceServer) UploadFile(context.Context, *UploadFileRequest) (*UploadFileReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UploadFile not implemented")
}
func (UnimplementedServiceServer) PageFile(context.Context, *PageFileRequest) (*PageFileReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PageFile not implemented")
}
func (UnimplementedServiceServer) UpdateFile(context.Context, *UpdateFileRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateFile not implemented")
}
func (UnimplementedServiceServer) DeleteFile(context.Context, *DeleteFileRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteFile not implemented")
}
func (UnimplementedServiceServer) mustEmbedUnimplementedServiceServer() {}

// UnsafeServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ServiceServer will
// result in compilation errors.
type UnsafeServiceServer interface {
	mustEmbedUnimplementedServiceServer()
}

func RegisterServiceServer(s grpc.ServiceRegistrar, srv ServiceServer) {
	s.RegisterService(&Service_ServiceDesc, srv)
}

func _Service_GetDirectory_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetDirectoryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).GetDirectory(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Service_GetDirectory_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).GetDirectory(ctx, req.(*GetDirectoryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Service_AddDirectory_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddDirectoryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).AddDirectory(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Service_AddDirectory_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).AddDirectory(ctx, req.(*AddDirectoryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Service_UpdateDirectory_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateDirectoryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).UpdateDirectory(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Service_UpdateDirectory_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).UpdateDirectory(ctx, req.(*UpdateDirectoryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Service_DeleteDirectory_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteDirectoryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).DeleteDirectory(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Service_DeleteDirectory_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).DeleteDirectory(ctx, req.(*DeleteDirectoryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Service_PrepareUploadFile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PrepareUploadFileRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).PrepareUploadFile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Service_PrepareUploadFile_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).PrepareUploadFile(ctx, req.(*PrepareUploadFileRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Service_UploadFile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UploadFileRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).UploadFile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Service_UploadFile_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).UploadFile(ctx, req.(*UploadFileRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Service_PageFile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PageFileRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).PageFile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Service_PageFile_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).PageFile(ctx, req.(*PageFileRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Service_UpdateFile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateFileRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).UpdateFile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Service_UpdateFile_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).UpdateFile(ctx, req.(*UpdateFileRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Service_DeleteFile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteFileRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).DeleteFile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Service_DeleteFile_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).DeleteFile(ctx, req.(*DeleteFileRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Service_ServiceDesc is the grpc.ServiceDesc for Service service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Service_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "v1.Service",
	HandlerType: (*ServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetDirectory",
			Handler:    _Service_GetDirectory_Handler,
		},
		{
			MethodName: "AddDirectory",
			Handler:    _Service_AddDirectory_Handler,
		},
		{
			MethodName: "UpdateDirectory",
			Handler:    _Service_UpdateDirectory_Handler,
		},
		{
			MethodName: "DeleteDirectory",
			Handler:    _Service_DeleteDirectory_Handler,
		},
		{
			MethodName: "PrepareUploadFile",
			Handler:    _Service_PrepareUploadFile_Handler,
		},
		{
			MethodName: "UploadFile",
			Handler:    _Service_UploadFile_Handler,
		},
		{
			MethodName: "PageFile",
			Handler:    _Service_PageFile_Handler,
		},
		{
			MethodName: "UpdateFile",
			Handler:    _Service_UpdateFile_Handler,
		},
		{
			MethodName: "DeleteFile",
			Handler:    _Service_DeleteFile_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "resource_service.proto",
}
