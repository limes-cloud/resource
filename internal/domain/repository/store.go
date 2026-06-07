package repository

import (
	"io"

	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/types"
)

// Store 文件存储操作接口
type Store interface {
	// ParserQuery 将图片/下载查询参数编码为 URL query 字符串
	ParserQuery(req *types.ParserQuery) string
	// Config 返回存储后端配置
	Config() *core.Storage
	// GenTemporaryURL 为指定 key 生成带时效的访问 URL
	GenTemporaryURL(key string) (string, error)
	// VerifyTemporaryURL 校验临时 URL 的签名与有效期
	VerifyTemporaryURL(key string, expire string, sign string) error
	// PutBytes 将字节数组写入指定 key
	PutBytes(key string, in []byte) error
	// Put 将 Reader 中的数据写入指定 key
	Put(key string, r io.Reader) error
	// PutFromLocal 将本地文件上传到指定 key
	PutFromLocal(key string, localPath string) error
	// Get 读取指定 key 的文件内容
	Get(key string) (io.ReadCloser, error)
	// Delete 删除指定 key 的文件
	Delete(key string) error
	// Size 返回指定 key 文件的字节大小
	Size(key string) (int64, error)
	// Exists 判断指定 key 的文件是否存在
	Exists(key string) (bool, error)
	// NewPutChunk 为指定 key 发起新的分片上传
	NewPutChunk(key string) (PutChunk, error)
	// NewPutChunkByUploadID 通过已有 uploadID 恢复分片上传
	NewPutChunkByUploadID(key string, id string) (PutChunk, error)
}

// PutChunk 分片上传会话接口
type PutChunk interface {
	// UploadedChunkIndex 返回已上传的分片索引列表
	UploadedChunkIndex() []uint32
	// ChunkCount 返回当前已上传的分片数量
	ChunkCount() int
	// UploadID 返回当前分片上传的 ID
	UploadID() string
	// AppendBytes 上传指定索引的分片字节数据
	AppendBytes(in []byte, index int) error
	// Append 上传指定索引的分片流数据
	Append(r io.Reader, index int) error
	// Abort 取消分片上传并释放已上传的部分数据
	Abort() error
	// Complete 完成分片上传并合并文件
	Complete() error
}
