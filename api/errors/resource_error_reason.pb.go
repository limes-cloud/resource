// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v4.24.4
// source: resource_error_reason.proto

package errors

import (
	_ "github.com/go-kratos/kratos/v2/errors"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Reason int32

const (
	Reason_NotFound             Reason = 0
	Reason_Transform            Reason = 1
	Reason_NoSupportStore       Reason = 2
	Reason_System               Reason = 3
	Reason_ChunkUpload          Reason = 4
	Reason_Database             Reason = 5
	Reason_StatusProgress       Reason = 6
	Reason_UploadFile           Reason = 7
	Reason_InitStore            Reason = 8
	Reason_FileFormat           Reason = 9
	Reason_AddDirectory         Reason = 10
	Reason_UpdateDirectory      Reason = 11
	Reason_DeleteDirectory      Reason = 12
	Reason_NotExistFile         Reason = 13
	Reason_AlreadyExistFileName Reason = 14
	Reason_NotExistDirectory    Reason = 15
	Reason_NotExistResource     Reason = 16
	Reason_Params               Reason = 17
)

// Enum value maps for Reason.
var (
	Reason_name = map[int32]string{
		0:  "NotFound",
		1:  "Transform",
		2:  "NoSupportStore",
		3:  "System",
		4:  "ChunkUpload",
		5:  "Database",
		6:  "StatusProgress",
		7:  "UploadFile",
		8:  "InitStore",
		9:  "FileFormat",
		10: "AddDirectory",
		11: "UpdateDirectory",
		12: "DeleteDirectory",
		13: "NotExistFile",
		14: "AlreadyExistFileName",
		15: "NotExistDirectory",
		16: "NotExistResource",
		17: "Params",
	}
	Reason_value = map[string]int32{
		"NotFound":             0,
		"Transform":            1,
		"NoSupportStore":       2,
		"System":               3,
		"ChunkUpload":          4,
		"Database":             5,
		"StatusProgress":       6,
		"UploadFile":           7,
		"InitStore":            8,
		"FileFormat":           9,
		"AddDirectory":         10,
		"UpdateDirectory":      11,
		"DeleteDirectory":      12,
		"NotExistFile":         13,
		"AlreadyExistFileName": 14,
		"NotExistDirectory":    15,
		"NotExistResource":     16,
		"Params":               17,
	}
)

func (x Reason) Enum() *Reason {
	p := new(Reason)
	*p = x
	return p
}

func (x Reason) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Reason) Descriptor() protoreflect.EnumDescriptor {
	return file_resource_error_reason_proto_enumTypes[0].Descriptor()
}

func (Reason) Type() protoreflect.EnumType {
	return &file_resource_error_reason_proto_enumTypes[0]
}

func (x Reason) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Reason.Descriptor instead.
func (Reason) EnumDescriptor() ([]byte, []int) {
	return file_resource_error_reason_proto_rawDescGZIP(), []int{0}
}

var File_resource_error_reason_proto protoreflect.FileDescriptor

var file_resource_error_reason_proto_rawDesc = []byte{
	0x0a, 0x1b, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x5f, 0x65, 0x72, 0x72, 0x6f, 0x72,
	0x5f, 0x72, 0x65, 0x61, 0x73, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x02, 0x76,
	0x31, 0x1a, 0x13, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x73, 0x2f, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x73,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2a, 0xe0, 0x05, 0x0a, 0x06, 0x52, 0x65, 0x61, 0x73, 0x6f,
	0x6e, 0x12, 0x20, 0x0a, 0x08, 0x4e, 0x6f, 0x74, 0x46, 0x6f, 0x75, 0x6e, 0x64, 0x10, 0x00, 0x1a,
	0x12, 0xb2, 0x45, 0x0f, 0xe4, 0xb8, 0x8d, 0xe5, 0xad, 0x98, 0xe5, 0x9c, 0xa8, 0xe6, 0x95, 0xb0,
	0xe6, 0x8d, 0xae, 0x12, 0x24, 0x0a, 0x09, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x66, 0x6f, 0x72, 0x6d,
	0x10, 0x01, 0x1a, 0x15, 0xb2, 0x45, 0x12, 0xe6, 0x95, 0xb0, 0xe6, 0x8d, 0xae, 0xe8, 0xbd, 0xac,
	0xe6, 0x8d, 0xa2, 0xe5, 0xa4, 0xb1, 0xe8, 0xb4, 0xa5, 0x12, 0x2f, 0x0a, 0x0e, 0x4e, 0x6f, 0x53,
	0x75, 0x70, 0x70, 0x6f, 0x72, 0x74, 0x53, 0x74, 0x6f, 0x72, 0x65, 0x10, 0x02, 0x1a, 0x1b, 0xb2,
	0x45, 0x18, 0xe4, 0xb8, 0x8d, 0xe6, 0x94, 0xaf, 0xe6, 0x8c, 0x81, 0xe7, 0x9a, 0x84, 0xe5, 0xad,
	0x98, 0xe5, 0x82, 0xa8, 0xe5, 0xbc, 0x95, 0xe6, 0x93, 0x8e, 0x12, 0x1b, 0x0a, 0x06, 0x53, 0x79,
	0x73, 0x74, 0x65, 0x6d, 0x10, 0x03, 0x1a, 0x0f, 0xb2, 0x45, 0x0c, 0xe7, 0xb3, 0xbb, 0xe7, 0xbb,
	0x9f, 0xe9, 0x94, 0x99, 0xe8, 0xaf, 0xaf, 0x12, 0x26, 0x0a, 0x0b, 0x43, 0x68, 0x75, 0x6e, 0x6b,
	0x55, 0x70, 0x6c, 0x6f, 0x61, 0x64, 0x10, 0x04, 0x1a, 0x15, 0xb2, 0x45, 0x12, 0xe5, 0x88, 0x86,
	0xe7, 0x89, 0x87, 0xe4, 0xb8, 0x8a, 0xe4, 0xbc, 0xa0, 0xe5, 0xa4, 0xb1, 0xe8, 0xb4, 0xa5, 0x12,
	0x20, 0x0a, 0x08, 0x44, 0x61, 0x74, 0x61, 0x62, 0x61, 0x73, 0x65, 0x10, 0x05, 0x1a, 0x12, 0xb2,
	0x45, 0x0f, 0xe6, 0x95, 0xb0, 0xe6, 0x8d, 0xae, 0xe5, 0xba, 0x93, 0xe9, 0x94, 0x99, 0xe8, 0xaf,
	0xaf, 0x12, 0x26, 0x0a, 0x0e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x50, 0x72, 0x6f, 0x67, 0x72,
	0x65, 0x73, 0x73, 0x10, 0x06, 0x1a, 0x12, 0xb2, 0x45, 0x0f, 0xe6, 0x96, 0x87, 0xe4, 0xbb, 0xb6,
	0xe4, 0xb8, 0x8a, 0xe4, 0xbc, 0xa0, 0xe4, 0xb8, 0xad, 0x12, 0x25, 0x0a, 0x0a, 0x55, 0x70, 0x6c,
	0x6f, 0x61, 0x64, 0x46, 0x69, 0x6c, 0x65, 0x10, 0x07, 0x1a, 0x15, 0xb2, 0x45, 0x12, 0xe6, 0x96,
	0x87, 0xe4, 0xbb, 0xb6, 0xe4, 0xb8, 0x8a, 0xe4, 0xbc, 0xa0, 0xe5, 0xa4, 0xb1, 0xe8, 0xb4, 0xa5,
	0x12, 0x2d, 0x0a, 0x09, 0x49, 0x6e, 0x69, 0x74, 0x53, 0x74, 0x6f, 0x72, 0x65, 0x10, 0x08, 0x1a,
	0x1e, 0xb2, 0x45, 0x1b, 0xe5, 0xad, 0x98, 0xe5, 0x82, 0xa8, 0xe5, 0xbc, 0x95, 0xe6, 0x93, 0x8e,
	0xe5, 0x88, 0x9d, 0xe5, 0xa7, 0x8b, 0xe5, 0x8c, 0x96, 0xe5, 0xa4, 0xb1, 0xe8, 0xb4, 0xa5, 0x12,
	0x25, 0x0a, 0x0a, 0x46, 0x69, 0x6c, 0x65, 0x46, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x10, 0x09, 0x1a,
	0x15, 0xb2, 0x45, 0x12, 0xe6, 0x96, 0x87, 0xe4, 0xbb, 0xb6, 0xe6, 0xa0, 0xbc, 0xe5, 0xbc, 0x8f,
	0xe9, 0x94, 0x99, 0xe8, 0xaf, 0xaf, 0x12, 0x27, 0x0a, 0x0c, 0x41, 0x64, 0x64, 0x44, 0x69, 0x72,
	0x65, 0x63, 0x74, 0x6f, 0x72, 0x79, 0x10, 0x0a, 0x1a, 0x15, 0xb2, 0x45, 0x12, 0xe7, 0x9b, 0xae,
	0xe5, 0xbd, 0x95, 0xe5, 0x88, 0x9b, 0xe5, 0xbb, 0xba, 0xe5, 0xa4, 0xb1, 0xe8, 0xb4, 0xa5, 0x12,
	0x2a, 0x0a, 0x0f, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x44, 0x69, 0x72, 0x65, 0x63, 0x74, 0x6f,
	0x72, 0x79, 0x10, 0x0b, 0x1a, 0x15, 0xb2, 0x45, 0x12, 0xe7, 0x9b, 0xae, 0xe5, 0xbd, 0x95, 0xe6,
	0x9b, 0xb4, 0xe6, 0x96, 0xb0, 0xe5, 0xa4, 0xb1, 0xe8, 0xb4, 0xa5, 0x12, 0x2a, 0x0a, 0x0f, 0x44,
	0x65, 0x6c, 0x65, 0x74, 0x65, 0x44, 0x69, 0x72, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x79, 0x10, 0x0c,
	0x1a, 0x15, 0xb2, 0x45, 0x12, 0xe7, 0x9b, 0xae, 0xe5, 0xbd, 0x95, 0xe5, 0x88, 0xa0, 0xe9, 0x99,
	0xa4, 0xe5, 0xa4, 0xb1, 0xe8, 0xb4, 0xa5, 0x12, 0x24, 0x0a, 0x0c, 0x4e, 0x6f, 0x74, 0x45, 0x78,
	0x69, 0x73, 0x74, 0x46, 0x69, 0x6c, 0x65, 0x10, 0x0d, 0x1a, 0x12, 0xb2, 0x45, 0x0f, 0xe6, 0x96,
	0x87, 0xe4, 0xbb, 0xb6, 0xe4, 0xb8, 0x8d, 0xe5, 0xad, 0x98, 0xe5, 0x9c, 0xa8, 0x12, 0x2f, 0x0a,
	0x14, 0x41, 0x6c, 0x72, 0x65, 0x61, 0x64, 0x79, 0x45, 0x78, 0x69, 0x73, 0x74, 0x46, 0x69, 0x6c,
	0x65, 0x4e, 0x61, 0x6d, 0x65, 0x10, 0x0e, 0x1a, 0x15, 0xb2, 0x45, 0x12, 0xe6, 0x96, 0x87, 0xe4,
	0xbb, 0xb6, 0xe5, 0x90, 0x8d, 0xe5, 0xb7, 0xb2, 0xe5, 0xad, 0x98, 0xe5, 0x9c, 0xa8, 0x12, 0x2c,
	0x0a, 0x11, 0x4e, 0x6f, 0x74, 0x45, 0x78, 0x69, 0x73, 0x74, 0x44, 0x69, 0x72, 0x65, 0x63, 0x74,
	0x6f, 0x72, 0x79, 0x10, 0x0f, 0x1a, 0x15, 0xb2, 0x45, 0x12, 0xe6, 0x96, 0x87, 0xe4, 0xbb, 0xb6,
	0xe5, 0xa4, 0xb9, 0xe4, 0xb8, 0x8d, 0xe5, 0xad, 0x98, 0xe5, 0x9c, 0xa8, 0x12, 0x28, 0x0a, 0x10,
	0x4e, 0x6f, 0x74, 0x45, 0x78, 0x69, 0x73, 0x74, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65,
	0x10, 0x10, 0x1a, 0x12, 0xb2, 0x45, 0x0f, 0xe8, 0xb5, 0x84, 0xe6, 0xba, 0x90, 0xe4, 0xb8, 0x8d,
	0xe5, 0xad, 0x98, 0xe5, 0x9c, 0xa8, 0x12, 0x1b, 0x0a, 0x06, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x73,
	0x10, 0x11, 0x1a, 0x0f, 0xb2, 0x45, 0x0c, 0xe5, 0x8f, 0x82, 0xe6, 0x95, 0xb0, 0xe9, 0x94, 0x99,
	0xe8, 0xaf, 0xaf, 0x1a, 0x04, 0xa0, 0x45, 0xc8, 0x01, 0x42, 0x0b, 0x5a, 0x09, 0x2e, 0x2f, 0x3b,
	0x65, 0x72, 0x72, 0x6f, 0x72, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_resource_error_reason_proto_rawDescOnce sync.Once
	file_resource_error_reason_proto_rawDescData = file_resource_error_reason_proto_rawDesc
)

func file_resource_error_reason_proto_rawDescGZIP() []byte {
	file_resource_error_reason_proto_rawDescOnce.Do(func() {
		file_resource_error_reason_proto_rawDescData = protoimpl.X.CompressGZIP(file_resource_error_reason_proto_rawDescData)
	})
	return file_resource_error_reason_proto_rawDescData
}

var file_resource_error_reason_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_resource_error_reason_proto_goTypes = []interface{}{
	(Reason)(0), // 0: v1.Reason
}
var file_resource_error_reason_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_resource_error_reason_proto_init() }
func file_resource_error_reason_proto_init() {
	if File_resource_error_reason_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_resource_error_reason_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_resource_error_reason_proto_goTypes,
		DependencyIndexes: file_resource_error_reason_proto_depIdxs,
		EnumInfos:         file_resource_error_reason_proto_enumTypes,
	}.Build()
	File_resource_error_reason_proto = out.File
	file_resource_error_reason_proto_rawDesc = nil
	file_resource_error_reason_proto_goTypes = nil
	file_resource_error_reason_proto_depIdxs = nil
}
