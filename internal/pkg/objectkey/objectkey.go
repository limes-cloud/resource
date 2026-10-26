// Package objectkey 是对象名的唯一构造与解析入口。
package objectkey

import (
	"errors"
	"path/filepath"
	"strings"
)

// exportPrefix 导出产物的功能命名空间。
const exportPrefix = "export/"

// exportExt 导出产物恒为 .zip
const exportExt = "zip"

// ErrInvalidObjectName 对象名不合法。
var ErrInvalidObjectName = errors.New("objectkey: 对象名不合法")

// ObjectName 由 (md5, ext) 构造对象名；ext 为空时不拼点。
func ObjectName(md5, ext string) string {
	if ext == "" {
		return md5
	}
	return md5 + "." + ext
}

// ExportObjectName 由 md5 构造导出产物名：export/{md5}.zip。
func ExportObjectName(md5 string) string {
	return exportPrefix + md5 + "." + exportExt
}

// ParseObjectName 解析用户对象名，返回 (md5, ext)；只允许用 filepath.Ext 取后缀段，按段数切分会把多点的名字判错。
func ParseObjectName(name string) (md5, ext string, err error) {
	suffix := filepath.Ext(name)
	md5 = strings.TrimSuffix(name, suffix)
	ext = strings.TrimPrefix(suffix, ".")
	if !IsLowerHex32(md5) || !IsValidExt(ext) {
		return "", "", ErrInvalidObjectName
	}
	return md5, ext, nil
}

// IsLowerHex32 判断 s 是否为 32 位小写 hex（内容摘要的合法形态）。
func IsLowerHex32(s string) bool {
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

// IsValidExt 判断 ext 是否为合法的归一化后缀：空串，或 1..16 位 [a-z0-9]。
func IsValidExt(ext string) bool {
	if len(ext) > 16 {
		return false
	}
	for i := 0; i < len(ext); i++ {
		c := ext[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'z') {
			return false
		}
	}
	return true
}

// mimeTypes 由后缀派生 Content-Type 的常用映射（建会话时写入分片上传元数据）。
var mimeTypes = map[string]string{
	"jpg": "image/jpeg", "jpeg": "image/jpeg", "png": "image/png", "gif": "image/gif",
	"webp": "image/webp", "bmp": "image/bmp", "svg": "image/svg+xml", "ico": "image/x-icon",
	"pdf": "application/pdf", "txt": "text/plain", "html": "text/html", "htm": "text/html",
	"css": "text/css", "js": "application/javascript", "json": "application/json", "xml": "application/xml",
	"mp4": "video/mp4", "webm": "video/webm", "mov": "video/quicktime",
	"mp3": "audio/mpeg", "wav": "audio/wav", "ogg": "audio/ogg",
	"zip": "application/zip", "rar": "application/x-rar-compressed", "7z": "application/x-7z-compressed",
	"tar": "application/x-tar", "gz": "application/gzip",
	"doc": "application/msword", "docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"xls": "application/vnd.ms-excel", "xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"ppt": "application/vnd.ms-powerpoint", "pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	"csv": "text/csv", "md": "text/markdown",
}

// ContentType 由 ext 派生 Content-Type；未知后缀回落二进制流。
func ContentType(ext string) string {
	if ct, ok := mimeTypes[strings.ToLower(ext)]; ok {
		return ct
	}
	return "application/octet-stream"
}
