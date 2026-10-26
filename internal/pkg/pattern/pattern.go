// Package pattern 提供子树前缀查询所需的 LIKE 转义，是全项目唯一的实现。
package pattern

import "strings"

// escaper 单遍替换：NewReplacer 不回头处理刚插入的字符，因此结果与"先转 \\、再转 %、再转 _"完全一致。
var escaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// EscapeLike 转义 LIKE 元字符：\ → \\，% → \%，_ → \_。
func EscapeLike(s string) string { return escaper.Replace(s) }

// PrefixPattern 返回子树前缀模式：EscapeLike(path) + "/%"。
func PrefixPattern(path string) string { return EscapeLike(path) + "/%" }
