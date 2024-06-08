package util

import (
	"strings"
)

// GetFileType 获取文件类型
func GetFileType(name string) string {
	index := strings.LastIndex(name, ".")
	suffix := ""
	if index != -1 {
		suffix = name[index+1:]
	}
	return suffix
}

func GetKBSize(mSize uint32) uint32 {
	return mSize * 1024
}
