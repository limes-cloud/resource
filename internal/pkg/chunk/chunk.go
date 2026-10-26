// Package chunk 提供分片大小推导与内容摘要计算。
package chunk

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
)

const (
	// MB 1MB 字节数
	MB = 1 << 20

	// MinChunkSize 全局分片下限 5MB。
	MinChunkSize = 5 * MB

	// MaxChunkCount 单个 multipart 上传的最大片数（厂商硬上限，已吸收进 Size 公式而非事后判）
	MaxChunkCount = 10000

	// MaxTotalSize 单文件物理天花板（字节）= 4095 × 1MB × 10000 ≈ 42.94 TB。
	MaxTotalSize = 4095 * 10000 * MB
)

var (
	// ErrEmptyFile 空文件
	ErrEmptyFile = errors.New("chunk: 不支持空文件")

	// ErrFileTooLarge 超过 MaxTotalSize（物理天花板）
	ErrFileTooLarge = errors.New("chunk: 文件过大")

	// ErrEmptyDigest 分片摘要序列为空
	ErrEmptyDigest = errors.New("chunk: 分片摘要为空")

	// ErrInvalidDigest 分片摘要非法：不是 32 位小写 hex，或片号重复/为 0
	ErrInvalidDigest = errors.New("chunk: 分片摘要非法")
)

// Size 推导分片大小：max(5MB, ceilDiv(totalSize, 10000*MB) * MB)；超过天花板返回 error，不静默截断。
func Size(totalSize uint64) (uint32, error) {
	if totalSize == 0 {
		return 0, ErrEmptyFile
	}
	if totalSize > MaxTotalSize {
		return 0, ErrFileTooLarge
	}
	size := ceilDiv(totalSize, uint64(MaxChunkCount)*MB) * MB
	if size < MinChunkSize {
		size = MinChunkSize
	}
	return uint32(size), nil
}

// Count 推导片数：ceilDiv(totalSize, chunkSize)；chunkSize 为 0 时返回 0。
func Count(totalSize uint64, chunkSize uint32) uint32 {
	if chunkSize == 0 {
		return 0
	}
	return uint32(ceilDiv(totalSize, uint64(chunkSize)))
}

// Part 分片摘要：片号（从 1 起）+ 该片内容摘要（32 位小写 hex）。
type Part struct {
	Number uint32 // 片号，从 1 起
	Digest string // 该片内容摘要（32 位小写 hex）
}

// Digest 计算内容摘要：按片号升序拼接各片摘要的 hex 文本再取 MD5；单片 N=1 同规（对文件 md5 再取一次 md5）。
func Digest(parts []Part) (string, error) {
	if len(parts) == 0 {
		return "", ErrEmptyDigest
	}

	sorted := make([]Part, len(parts))
	copy(sorted, parts)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Number < sorted[j].Number })

	var b strings.Builder
	b.Grow(len(sorted) * 32)
	var prev uint32
	for _, p := range sorted {
		if p.Number == 0 || p.Number == prev || !isLowerHex32(p.Digest) {
			return "", ErrInvalidDigest
		}
		b.WriteString(p.Digest)
		prev = p.Number
	}

	sum := md5.Sum([]byte(b.String()))
	return hex.EncodeToString(sum[:]), nil
}

// NormalizeETag 归一化厂商分片 ETag：剥掉包裹的双引号、去空白、转小写，得到可直接参与聚合的 hex。
func NormalizeETag(etag string) string {
	return strings.ToLower(strings.Trim(strings.TrimSpace(etag), `"`))
}

// ceilDiv 整数除法向上取整
func ceilDiv(a, b uint64) uint64 { return (a + b - 1) / b }

// isLowerHex32 判断 s 是否为 32 位小写 hex
func isLowerHex32(s string) bool {
	if len(s) != 32 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
