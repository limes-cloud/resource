package store

import "io"

type Store interface {
	// PutBytes 上传文件
	PutBytes(key string, in []byte) error
	// Put 上传文件
	Put(key string, r io.Reader) error
	// PutFromLocal 从本地上传文件
	PutFromLocal(key string, localPath string) error
	// Get 查询文件
	Get(key string) (io.ReadCloser, error)
	// Delete 删除文件
	Delete(key string) error
	// Size 获取文件大小
	Size(key string) (int64, error)
	// Exists 判断文件大小
	Exists(key string) (bool, error)
	// NewPutChunk 创建上传切片对象
	NewPutChunk(key string) (PutChunk, error)
	// NewPutChunkByUploadID 通过upload_id创建切片对象
	NewPutChunkByUploadID(key string, id string) (PutChunk, error)
}

type PutChunk interface {
	// UploadedChunkIndex 已经上传的切片下标
	UploadedChunkIndex() []int
	// ChunkCount 查询切片数量
	ChunkCount() int
	// UploadID 获取上传id
	UploadID() string
	// AppendBytes 添加字节
	AppendBytes(in []byte, index int) error
	// Append 添加io
	Append(r io.Reader, index int) error
	// Abort 取消上传
	Abort() error
	// Complete 完成合并
	Complete() error
}
