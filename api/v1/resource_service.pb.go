// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v4.24.4
// source: resource_service.proto

package v1

import (
	_ "google.golang.org/genproto/googleapis/api/annotations"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	reflect "reflect"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

var File_resource_service_proto protoreflect.FileDescriptor

var file_resource_service_proto_rawDesc = []byte{
	0x0a, 0x16, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x69,
	0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x02, 0x76, 0x31, 0x1a, 0x1c, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x13, 0x72, 0x65, 0x73, 0x6f,
	0x75, 0x72, 0x63, 0x65, 0x5f, 0x66, 0x69, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a,
	0x18, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x5f, 0x64, 0x69, 0x72, 0x65, 0x63, 0x74,
	0x6f, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x32, 0xa2, 0x07, 0x0a, 0x07, 0x53, 0x65, 0x72, 0x76, 0x69,
	0x63, 0x65, 0x12, 0x6a, 0x0a, 0x0c, 0x47, 0x65, 0x74, 0x44, 0x69, 0x72, 0x65, 0x63, 0x74, 0x6f,
	0x72, 0x79, 0x12, 0x17, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x44, 0x69, 0x72, 0x65, 0x63,
	0x74, 0x6f, 0x72, 0x79, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x15, 0x2e, 0x76, 0x31,
	0x2e, 0x47, 0x65, 0x74, 0x44, 0x69, 0x72, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x79, 0x52, 0x65, 0x70,
	0x6c, 0x79, 0x22, 0x2a, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x24, 0x62, 0x04, 0x6c, 0x69, 0x73, 0x74,
	0x12, 0x1c, 0x2f, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x2f, 0x76, 0x31, 0x2f, 0x7b,
	0x61, 0x70, 0x70, 0x7d, 0x2f, 0x64, 0x69, 0x72, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x79, 0x12, 0x5f,
	0x0a, 0x0c, 0x41, 0x64, 0x64, 0x44, 0x69, 0x72, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x79, 0x12, 0x17,
	0x2e, 0x76, 0x31, 0x2e, 0x41, 0x64, 0x64, 0x44, 0x69, 0x72, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x79,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x0d, 0x2e, 0x76, 0x31, 0x2e, 0x44, 0x69, 0x72,
	0x65, 0x63, 0x74, 0x6f, 0x72, 0x79, 0x22, 0x27, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x21, 0x3a, 0x01,
	0x2a, 0x22, 0x1c, 0x2f, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x2f, 0x76, 0x31, 0x2f,
	0x7b, 0x61, 0x70, 0x70, 0x7d, 0x2f, 0x64, 0x69, 0x72, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x79, 0x12,
	0x6e, 0x0a, 0x0f, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x44, 0x69, 0x72, 0x65, 0x63, 0x74, 0x6f,
	0x72, 0x79, 0x12, 0x1a, 0x2e, 0x76, 0x31, 0x2e, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x44, 0x69,
	0x72, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x79, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x16,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x27, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x21, 0x3a, 0x01,
	0x2a, 0x1a, 0x1c, 0x2f, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x2f, 0x76, 0x31, 0x2f,
	0x7b, 0x61, 0x70, 0x70, 0x7d, 0x2f, 0x64, 0x69, 0x72, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x79, 0x12,
	0x6b, 0x0a, 0x0f, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x44, 0x69, 0x72, 0x65, 0x63, 0x74, 0x6f,
	0x72, 0x79, 0x12, 0x1a, 0x2e, 0x76, 0x31, 0x2e, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x44, 0x69,
	0x72, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x79, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x16,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x24, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x1e, 0x2a, 0x1c,
	0x2f, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x2f, 0x76, 0x31, 0x2f, 0x7b, 0x61, 0x70,
	0x70, 0x7d, 0x2f, 0x64, 0x69, 0x72, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x79, 0x12, 0x7b, 0x0a, 0x11,
	0x50, 0x72, 0x65, 0x70, 0x61, 0x72, 0x65, 0x55, 0x70, 0x6c, 0x6f, 0x61, 0x64, 0x46, 0x69, 0x6c,
	0x65, 0x12, 0x1c, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x72, 0x65, 0x70, 0x61, 0x72, 0x65, 0x55, 0x70,
	0x6c, 0x6f, 0x61, 0x64, 0x46, 0x69, 0x6c, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a,
	0x1a, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x72, 0x65, 0x70, 0x61, 0x72, 0x65, 0x55, 0x70, 0x6c, 0x6f,
	0x61, 0x64, 0x46, 0x69, 0x6c, 0x65, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x22, 0x2c, 0x82, 0xd3, 0xe4,
	0x93, 0x02, 0x26, 0x3a, 0x01, 0x2a, 0x22, 0x21, 0x2f, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63,
	0x65, 0x2f, 0x76, 0x31, 0x2f, 0x7b, 0x61, 0x70, 0x70, 0x7d, 0x2f, 0x75, 0x70, 0x6c, 0x6f, 0x61,
	0x64, 0x2f, 0x70, 0x72, 0x65, 0x70, 0x61, 0x72, 0x65, 0x12, 0x58, 0x0a, 0x0a, 0x55, 0x70, 0x6c,
	0x6f, 0x61, 0x64, 0x46, 0x69, 0x6c, 0x65, 0x12, 0x15, 0x2e, 0x76, 0x31, 0x2e, 0x55, 0x70, 0x6c,
	0x6f, 0x61, 0x64, 0x46, 0x69, 0x6c, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x13,
	0x2e, 0x76, 0x31, 0x2e, 0x55, 0x70, 0x6c, 0x6f, 0x61, 0x64, 0x46, 0x69, 0x6c, 0x65, 0x52, 0x65,
	0x70, 0x6c, 0x79, 0x22, 0x1e, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x18, 0x3a, 0x01, 0x2a, 0x22, 0x13,
	0x2f, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x2f, 0x76, 0x31, 0x2f, 0x75, 0x70, 0x6c,
	0x6f, 0x61, 0x64, 0x12, 0x54, 0x0a, 0x08, 0x50, 0x61, 0x67, 0x65, 0x46, 0x69, 0x6c, 0x65, 0x12,
	0x13, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x61, 0x67, 0x65, 0x46, 0x69, 0x6c, 0x65, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x1a, 0x11, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x61, 0x67, 0x65, 0x46, 0x69,
	0x6c, 0x65, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x22, 0x20, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x1a, 0x12,
	0x18, 0x2f, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x2f, 0x76, 0x31, 0x2f, 0x7b, 0x61,
	0x70, 0x70, 0x7d, 0x2f, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x12, 0x5f, 0x0a, 0x0a, 0x55, 0x70, 0x64,
	0x61, 0x74, 0x65, 0x46, 0x69, 0x6c, 0x65, 0x12, 0x15, 0x2e, 0x76, 0x31, 0x2e, 0x55, 0x70, 0x64,
	0x61, 0x74, 0x65, 0x46, 0x69, 0x6c, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x16,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x22, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x1c, 0x3a, 0x01,
	0x2a, 0x1a, 0x17, 0x2f, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x2f, 0x76, 0x31, 0x2f,
	0x7b, 0x61, 0x70, 0x70, 0x7d, 0x2f, 0x66, 0x69, 0x6c, 0x65, 0x12, 0x5f, 0x0a, 0x0a, 0x44, 0x65,
	0x6c, 0x65, 0x74, 0x65, 0x46, 0x69, 0x6c, 0x65, 0x12, 0x15, 0x2e, 0x76, 0x31, 0x2e, 0x44, 0x65,
	0x6c, 0x65, 0x74, 0x65, 0x46, 0x69, 0x6c, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a,
	0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x22, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x1c, 0x3a,
	0x01, 0x2a, 0x22, 0x17, 0x2f, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x2f, 0x76, 0x31,
	0x2f, 0x7b, 0x61, 0x70, 0x70, 0x7d, 0x2f, 0x66, 0x69, 0x6c, 0x65, 0x42, 0x09, 0x5a, 0x07, 0x2e,
	0x2f, 0x76, 0x31, 0x3b, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var file_resource_service_proto_goTypes = []interface{}{
	(*GetDirectoryRequest)(nil),      // 0: v1.GetDirectoryRequest
	(*AddDirectoryRequest)(nil),      // 1: v1.AddDirectoryRequest
	(*UpdateDirectoryRequest)(nil),   // 2: v1.UpdateDirectoryRequest
	(*DeleteDirectoryRequest)(nil),   // 3: v1.DeleteDirectoryRequest
	(*PrepareUploadFileRequest)(nil), // 4: v1.PrepareUploadFileRequest
	(*UploadFileRequest)(nil),        // 5: v1.UploadFileRequest
	(*PageFileRequest)(nil),          // 6: v1.PageFileRequest
	(*UpdateFileRequest)(nil),        // 7: v1.UpdateFileRequest
	(*DeleteFileRequest)(nil),        // 8: v1.DeleteFileRequest
	(*GetDirectoryReply)(nil),        // 9: v1.GetDirectoryReply
	(*Directory)(nil),                // 10: v1.Directory
	(*emptypb.Empty)(nil),            // 11: google.protobuf.Empty
	(*PrepareUploadFileReply)(nil),   // 12: v1.PrepareUploadFileReply
	(*UploadFileReply)(nil),          // 13: v1.UploadFileReply
	(*PageFileReply)(nil),            // 14: v1.PageFileReply
}
var file_resource_service_proto_depIdxs = []int32{
	0,  // 0: v1.Service.GetDirectory:input_type -> v1.GetDirectoryRequest
	1,  // 1: v1.Service.AddDirectory:input_type -> v1.AddDirectoryRequest
	2,  // 2: v1.Service.UpdateDirectory:input_type -> v1.UpdateDirectoryRequest
	3,  // 3: v1.Service.DeleteDirectory:input_type -> v1.DeleteDirectoryRequest
	4,  // 4: v1.Service.PrepareUploadFile:input_type -> v1.PrepareUploadFileRequest
	5,  // 5: v1.Service.UploadFile:input_type -> v1.UploadFileRequest
	6,  // 6: v1.Service.PageFile:input_type -> v1.PageFileRequest
	7,  // 7: v1.Service.UpdateFile:input_type -> v1.UpdateFileRequest
	8,  // 8: v1.Service.DeleteFile:input_type -> v1.DeleteFileRequest
	9,  // 9: v1.Service.GetDirectory:output_type -> v1.GetDirectoryReply
	10, // 10: v1.Service.AddDirectory:output_type -> v1.Directory
	11, // 11: v1.Service.UpdateDirectory:output_type -> google.protobuf.Empty
	11, // 12: v1.Service.DeleteDirectory:output_type -> google.protobuf.Empty
	12, // 13: v1.Service.PrepareUploadFile:output_type -> v1.PrepareUploadFileReply
	13, // 14: v1.Service.UploadFile:output_type -> v1.UploadFileReply
	14, // 15: v1.Service.PageFile:output_type -> v1.PageFileReply
	11, // 16: v1.Service.UpdateFile:output_type -> google.protobuf.Empty
	11, // 17: v1.Service.DeleteFile:output_type -> google.protobuf.Empty
	9,  // [9:18] is the sub-list for method output_type
	0,  // [0:9] is the sub-list for method input_type
	0,  // [0:0] is the sub-list for extension type_name
	0,  // [0:0] is the sub-list for extension extendee
	0,  // [0:0] is the sub-list for field type_name
}

func init() { file_resource_service_proto_init() }
func file_resource_service_proto_init() {
	if File_resource_service_proto != nil {
		return
	}
	file_resource_file_proto_init()
	file_resource_directory_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_resource_service_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_resource_service_proto_goTypes,
		DependencyIndexes: file_resource_service_proto_depIdxs,
	}.Build()
	File_resource_service_proto = out.File
	file_resource_service_proto_rawDesc = nil
	file_resource_service_proto_goTypes = nil
	file_resource_service_proto_depIdxs = nil
}
