// Code generated by protoc-gen-go-errors. DO NOT EDIT.

package errors

import (
	fmt "fmt"
	errors "github.com/go-kratos/kratos/v2/errors"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the kratos package it is being compiled against.
const _ = errors.SupportPackageIsVersion1

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == Reason_NotFound.String() && e.Code == 200
}

func NotFoundFormat(format string, args ...any) *errors.Error {
	return errors.New(200, Reason_NotFound.String(), "不存在数据:"+fmt.Sprintf(format, args...))
}

func NotFound() *errors.Error {
	return errors.New(200, Reason_NotFound.String(), "不存在数据")
}

func IsTransform(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == Reason_Transform.String() && e.Code == 200
}

func TransformFormat(format string, args ...any) *errors.Error {
	return errors.New(200, Reason_Transform.String(), "数据转换失败:"+fmt.Sprintf(format, args...))
}

func Transform() *errors.Error {
	return errors.New(200, Reason_Transform.String(), "数据转换失败")
}

func IsNoSupportStore(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == Reason_NoSupportStore.String() && e.Code == 200
}

func NoSupportStoreFormat(format string, args ...any) *errors.Error {
	return errors.New(200, Reason_NoSupportStore.String(), "不支持的存储引擎:"+fmt.Sprintf(format, args...))
}

func NoSupportStore() *errors.Error {
	return errors.New(200, Reason_NoSupportStore.String(), "不支持的存储引擎")
}

func IsSystem(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == Reason_System.String() && e.Code == 200
}

func SystemFormat(format string, args ...any) *errors.Error {
	return errors.New(200, Reason_System.String(), "系统错误:"+fmt.Sprintf(format, args...))
}

func System() *errors.Error {
	return errors.New(200, Reason_System.String(), "系统错误")
}

func IsChunkUpload(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == Reason_ChunkUpload.String() && e.Code == 200
}

func ChunkUploadFormat(format string, args ...any) *errors.Error {
	return errors.New(200, Reason_ChunkUpload.String(), "分片上传失败:"+fmt.Sprintf(format, args...))
}

func ChunkUpload() *errors.Error {
	return errors.New(200, Reason_ChunkUpload.String(), "分片上传失败")
}

func IsDatabase(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == Reason_Database.String() && e.Code == 200
}

func DatabaseFormat(format string, args ...any) *errors.Error {
	return errors.New(200, Reason_Database.String(), "数据库错误:"+fmt.Sprintf(format, args...))
}

func Database() *errors.Error {
	return errors.New(200, Reason_Database.String(), "数据库错误")
}

func IsStatusProgress(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == Reason_StatusProgress.String() && e.Code == 200
}

func StatusProgressFormat(format string, args ...any) *errors.Error {
	return errors.New(200, Reason_StatusProgress.String(), "文件上传中:"+fmt.Sprintf(format, args...))
}

func StatusProgress() *errors.Error {
	return errors.New(200, Reason_StatusProgress.String(), "文件上传中")
}

func IsUploadFile(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == Reason_UploadFile.String() && e.Code == 200
}

func UploadFileFormat(format string, args ...any) *errors.Error {
	return errors.New(200, Reason_UploadFile.String(), "文件上传失败:"+fmt.Sprintf(format, args...))
}

func UploadFile() *errors.Error {
	return errors.New(200, Reason_UploadFile.String(), "文件上传失败")
}

func IsInitStore(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == Reason_InitStore.String() && e.Code == 200
}

func InitStoreFormat(format string, args ...any) *errors.Error {
	return errors.New(200, Reason_InitStore.String(), "存储引擎初始化失败:"+fmt.Sprintf(format, args...))
}

func InitStore() *errors.Error {
	return errors.New(200, Reason_InitStore.String(), "存储引擎初始化失败")
}

func IsFileFormat(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == Reason_FileFormat.String() && e.Code == 200
}

func FileFormatFormat(format string, args ...any) *errors.Error {
	return errors.New(200, Reason_FileFormat.String(), "文件格式错误:"+fmt.Sprintf(format, args...))
}

func FileFormat() *errors.Error {
	return errors.New(200, Reason_FileFormat.String(), "文件格式错误")
}

func IsAddDirectory(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == Reason_AddDirectory.String() && e.Code == 200
}

func AddDirectoryFormat(format string, args ...any) *errors.Error {
	return errors.New(200, Reason_AddDirectory.String(), "目录创建失败:"+fmt.Sprintf(format, args...))
}

func AddDirectory() *errors.Error {
	return errors.New(200, Reason_AddDirectory.String(), "目录创建失败")
}

func IsUpdateDirectory(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == Reason_UpdateDirectory.String() && e.Code == 200
}

func UpdateDirectoryFormat(format string, args ...any) *errors.Error {
	return errors.New(200, Reason_UpdateDirectory.String(), "目录更新失败:"+fmt.Sprintf(format, args...))
}

func UpdateDirectory() *errors.Error {
	return errors.New(200, Reason_UpdateDirectory.String(), "目录更新失败")
}

func IsDeleteDirectory(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == Reason_DeleteDirectory.String() && e.Code == 200
}

func DeleteDirectoryFormat(format string, args ...any) *errors.Error {
	return errors.New(200, Reason_DeleteDirectory.String(), "目录删除失败:"+fmt.Sprintf(format, args...))
}

func DeleteDirectory() *errors.Error {
	return errors.New(200, Reason_DeleteDirectory.String(), "目录删除失败")
}

func IsNotExistFile(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == Reason_NotExistFile.String() && e.Code == 200
}

func NotExistFileFormat(format string, args ...any) *errors.Error {
	return errors.New(200, Reason_NotExistFile.String(), "文件不存在:"+fmt.Sprintf(format, args...))
}

func NotExistFile() *errors.Error {
	return errors.New(200, Reason_NotExistFile.String(), "文件不存在")
}

func IsAlreadyExistFileName(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == Reason_AlreadyExistFileName.String() && e.Code == 200
}

func AlreadyExistFileNameFormat(format string, args ...any) *errors.Error {
	return errors.New(200, Reason_AlreadyExistFileName.String(), "文件名已存在:"+fmt.Sprintf(format, args...))
}

func AlreadyExistFileName() *errors.Error {
	return errors.New(200, Reason_AlreadyExistFileName.String(), "文件名已存在")
}

func IsNotExistDirectory(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == Reason_NotExistDirectory.String() && e.Code == 200
}

func NotExistDirectoryFormat(format string, args ...any) *errors.Error {
	return errors.New(200, Reason_NotExistDirectory.String(), "文件夹不存在:"+fmt.Sprintf(format, args...))
}

func NotExistDirectory() *errors.Error {
	return errors.New(200, Reason_NotExistDirectory.String(), "文件夹不存在")
}

func IsNotExistResource(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == Reason_NotExistResource.String() && e.Code == 200
}

func NotExistResourceFormat(format string, args ...any) *errors.Error {
	return errors.New(200, Reason_NotExistResource.String(), "资源不存在:"+fmt.Sprintf(format, args...))
}

func NotExistResource() *errors.Error {
	return errors.New(200, Reason_NotExistResource.String(), "资源不存在")
}

func IsParams(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == Reason_Params.String() && e.Code == 200
}

func ParamsFormat(format string, args ...any) *errors.Error {
	return errors.New(200, Reason_Params.String(), "参数错误:"+fmt.Sprintf(format, args...))
}

func Params() *errors.Error {
	return errors.New(200, Reason_Params.String(), "参数错误")
}

func IsAccessResource(err error) bool {
	if err == nil {
		return false
	}
	e := errors.FromError(err)
	return e.Reason == Reason_AccessResource.String() && e.Code == 200
}

func AccessResourceFormat(format string, args ...any) *errors.Error {
	return errors.New(200, Reason_AccessResource.String(), "访问资源文件异常:"+fmt.Sprintf(format, args...))
}

func AccessResource() *errors.Error {
	return errors.New(200, Reason_AccessResource.String(), "访问资源文件异常")
}
