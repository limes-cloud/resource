package service

import (
	"gorm.io/gorm"

	"github.com/limes-cloud/kratosx/library/db/gormtranserror"
)

// isNotFound 判断底层错误是否为 gorm 的"记录不存在"
func isNotFound(err error) bool { return gormtranserror.Is(err, gorm.ErrRecordNotFound) }

// isDuplicate 判断底层错误是否为 gorm 的"唯一键冲突"
func isDuplicate(err error) bool { return gormtranserror.Is(err, gorm.ErrDuplicatedKey) }
