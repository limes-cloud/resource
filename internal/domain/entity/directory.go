package entity

import (
	"fmt"
	"github.com/limes-cloud/kratosx/model"
	"strings"
	"unicode/utf8"
)

// accept 取值与 max_size 哨兵（目录策略的领域常量）
const (
	// AcceptAll accept 的"允许全部"取值
	AcceptAll = "*"
	// AcceptNone accept 的"不允许任何类型"取值（空串）
	AcceptNone = ""
	// MaxSizeUnlimited max_size 的"不限"取值
	MaxSizeUnlimited uint32 = 4294967295

	// MaxDepth 目录限深
	MaxDepth = 10
	// MaxNameLen 目录名最大字符数（CHAR_LENGTH 语义，一个汉字算 1 个字符）
	MaxNameLen = 64

	// sizeMB max_size 以 MB 计，换算成字节
	sizeMB = 1 << 20
)

// ValidateName 校验目录 / 文件名文法：非空、≤64 字符、不含 /、非 . 与 ..、无控制字符与四字节字符（proto 表达不了）。
func ValidateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("名称为空")
	}
	if utf8.RuneCountInString(name) > MaxNameLen {
		return fmt.Errorf("名称超过 %d 个字符", MaxNameLen)
	}
	if strings.Contains(name, "/") {
		return fmt.Errorf("名称不能含 /")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("名称不能是 . 或 ..")
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7F {
			return fmt.Errorf("名称含控制字符")
		}
		if r > 0xFFFF {
			return fmt.Errorf("名称含四字节字符（含绝大多数 emoji）")
		}
	}
	return nil
}

// Directory 应用目录树（物化路径，无 parent_id）
type Directory struct {
	App       string  `json:"app"       gorm:"column:app"`        // 应用标识
	Name      string  `json:"name"      gorm:"column:name"`       // 目录名
	Path      string  `json:"path"      gorm:"column:path"`       // 物化路径，形如 /图片/头像（限深 10）
	Accept    *string `json:"accept"    gorm:"column:accept"`     // 允许后缀：'*'全部|''不允许任何类型|后缀白名单；'' 是合法零值，用指针避免更新被跳过
	MaxSize   uint32  `json:"maxSize"   gorm:"column:max_size"`   // 单文件上限（MB）；4294967295 = 不限（0 非法）
	UpdatedBy uint32  `json:"updatedBy" gorm:"column:updated_by"` // 最后修改人
	model.BaseModel
}

// AllowExt 目录当前 accept 策略是否允许该后缀（上传策略判定，非参数校验）。
func (d *Directory) AllowExt(ext string) bool {
	accept := ""
	if d.Accept != nil {
		accept = *d.Accept
	}
	switch accept {
	case AcceptAll:
		return true
	case AcceptNone:
		return false
	}
	for _, seg := range strings.Split(accept, ",") {
		if strings.TrimSpace(seg) == ext {
			return true
		}
	}
	return false
}

// WithinSize 字节数是否在目录单文件上限内（MaxSizeUnlimited 表示不限）。
func (d *Directory) WithinSize(bytes uint64) bool {
	if d.MaxSize == MaxSizeUnlimited {
		return true
	}
	return bytes <= uint64(d.MaxSize)*sizeMB
}
