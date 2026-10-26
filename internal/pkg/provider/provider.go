// Package provider 是厂商知识的唯一入口：白名单、(provider, region) → 行为表、
package provider

import (
	"errors"
	"regexp"
	"strings"
)

// 厂商标识：store.provider 的合法取值，即代码里的白名单。
const (
	AWS   = "aws"   // AWS S3
	BOS   = "bos"   // 百度云 BOS
	OSS   = "oss"   // 阿里云 OSS
	COS   = "cos"   // 腾讯云 COS
	OBS   = "obs"   // 华为云 OBS
	MINIO = "minio" // MinIO（自建，S3 兼容）
	USS   = "uss"   // UCloud US3
)

// 寻址方式
const (
	AddressingPath  = "path"  // 路径寻址：{endpoint}/{bucket}/{key}
	AddressingVhost = "vhost" // 虚拟主机寻址：{bucket}.{endpoint}/{key}
)

var (
	// ErrUnknownProvider provider 不在白名单内
	ErrUnknownProvider = errors.New("provider: 不在白名单内")
	// ErrNotImplemented provider 在白名单内但行为表未实现
	ErrNotImplemented = errors.New("provider: 行为表未实现")
	// ErrInvalidEndpoint endpoint 为空或非法
	ErrInvalidEndpoint = errors.New("provider: endpoint 为空或非法")
	// ErrEndpointMismatch host 不匹配该 provider 的域名族
	ErrEndpointMismatch = errors.New("provider: endpoint 不匹配该 provider 的域名族")
)

// Provider 厂商描述
type Provider struct {
	Value       string // 写进 store.provider 的字面量
	Label       string // 展示名
	Implemented bool   // 行为表是否已实现（false 让前端置灰，而不是让用户选完再被拒）
}

// providerOrder 白名单与展示名；顺序即 List 返回顺序。
var providerOrder = []Provider{
	{Value: AWS, Label: "AWS S3"},
	{Value: BOS, Label: "百度云 BOS"},
	{Value: OSS, Label: "阿里云 OSS"},
	{Value: COS, Label: "腾讯云 COS"},
	{Value: OBS, Label: "华为云 OBS"},
	{Value: MINIO, Label: "MinIO"},
	{Value: USS, Label: "UCloud US3"},
}

// domainRule 一条 endpoint 归一化规则：pattern 匹配归一化后的 host，replace 用 ${n} 引用捕获组。
type domainRule struct {
	pattern *regexp.Regexp
	replace string
}

// behavior 单厂商行为表（厂商知识，属代码常量）。
type behavior struct {
	defaultAddressing    string       // 默认寻址方式
	vhostOnlyRegions     []string     // 强制 vhost 的区域（默认 path 的厂商用）
	domainRules          []domainRule // endpoint 归一化规则，顺序敏感（先匹配先返回）
	partETagIsContentMD5 bool         // 分片 ETag 是否等于分片内容 MD5（未开服务端加密时成立）
}

// behaviors 已实现行为表的厂商。表里没有的厂商 = 未实现（implemented=false）。
// 说明：以下四项除 AWS/BOS/MINIO 外，均基于各厂商公开的 S3 兼容端点文档实现；
// 未在真实环境校准（尤其 endpoint 归一化），如实际环境域名族有出入，请按"校准"流程修订本表。
var behaviors = map[string]behavior{
	BOS: {
		defaultAddressing:    AddressingPath,
		partETagIsContentMD5: true,
		// 四种拼写（域名族 / vhost / scheme / 尾斜杠）统一到 S3 兼容形态 s3.{region}.bcebos.com
		domainRules: []domainRule{
			rule(`^s3\.([a-z0-9-]+)\.bcebos\.com$`, "s3.${1}.bcebos.com"),
			rule(`^[a-z0-9-]+\.s3\.([a-z0-9-]+)\.bcebos\.com$`, "s3.${1}.bcebos.com"),
			rule(`^([a-z0-9-]+)\.bcebos\.com$`, "s3.${1}.bcebos.com"),
			rule(`^[a-z0-9-]+\.([a-z0-9-]+)\.bcebos\.com$`, "s3.${1}.bcebos.com"),
		},
	},
	AWS: {
		defaultAddressing:    AddressingPath,
		partETagIsContentMD5: true,
		domainRules: []domainRule{
			rule(`^s3\.([a-z0-9-]+)\.amazonaws\.com$`, "s3.${1}.amazonaws.com"),
			rule(`^s3-([a-z0-9-]+)\.amazonaws\.com$`, "s3.${1}.amazonaws.com"),
			rule(`^[a-z0-9-]+\.s3\.([a-z0-9-]+)\.amazonaws\.com$`, "s3.${1}.amazonaws.com"),
			rule(`^[a-z0-9-]+\.s3-([a-z0-9-]+)\.amazonaws\.com$`, "s3.${1}.amazonaws.com"),
			rule(`^[a-z0-9-]+\.s3\.amazonaws\.com$`, "s3.amazonaws.com"),
			rule(`^s3\.amazonaws\.com$`, "s3.amazonaws.com"),
		},
	},
	MINIO: {
		defaultAddressing:    AddressingPath,
		partETagIsContentMD5: true,
		// 自建服务，域名任意（可带端口）；按 path 寻址使用，不做桶前缀剥离
		domainRules: []domainRule{
			rule(`^([a-z0-9.-]+)(:[0-9]+)?$`, "${1}${2}"),
		},
	},
	OSS: {
		// 阿里云 OSS 的 S3 兼容接口必须使用三级域名（vhost）寻址：{bucket}.oss-cn-{region}.aliyuncs.com；
		// path 寻址（二级域名 + /{bucket}）会返回 SecondLevelDomainForbidden（实测确认）
		defaultAddressing:    AddressingVhost,
		partETagIsContentMD5: true, // 未开服务端加密时，分片 ETag = 该分片内容 MD5（对象级 ETag 为 {parts_md5}-{count}，不影响分片校验）
		// 阿里云 OSS：国内 oss-cn-{region}.aliyuncs.com / 国际 oss-{region}.aliyuncs.com；
		// 筒前缀 vhost 形态剥离回裸端点（vhost 下 endpoint 恒为不带桶的裸端点）；-internal 内网端点物理不同，不投影（未命中即 ErrEndpointMismatch）
		domainRules: []domainRule{
			rule(`^oss-cn-([a-z0-9-]+)\.aliyuncs\.com$`, "oss-cn-${1}.aliyuncs.com"),
			rule(`^[a-z0-9-]+\.oss-cn-([a-z0-9-]+)\.aliyuncs\.com$`, "oss-cn-${1}.aliyuncs.com"),
			rule(`^oss-([a-z0-9-]+)\.aliyuncs\.com$`, "oss-${1}.aliyuncs.com"),
			rule(`^[a-z0-9-]+\.oss-([a-z0-9-]+)\.aliyuncs\.com$`, "oss-${1}.aliyuncs.com"),
		},
	},
	COS: {
		defaultAddressing:    AddressingPath,
		partETagIsContentMD5: true,
		// 腾讯云 COS：cos.{region}.myqcloud.com（region 含 ap- 等前缀）；桶名常带 -{appid} 后缀，仍按前缀剥离
		domainRules: []domainRule{
			rule(`^cos\.([a-z0-9-]+)\.myqcloud\.com$`, "cos.${1}.myqcloud.com"),
			rule(`^[a-z0-9-]+\.cos\.([a-z0-9-]+)\.myqcloud\.com$`, "cos.${1}.myqcloud.com"),
		},
	},
	OBS: {
		defaultAddressing:    AddressingPath,
		partETagIsContentMD5: true,
		// 华为云 OBS：obs.{region}.myhuaweicloud.com
		domainRules: []domainRule{
			rule(`^obs\.([a-z0-9-]+)\.myhuaweicloud\.com$`, "obs.${1}.myhuaweicloud.com"),
			rule(`^[a-z0-9-]+\.obs\.([a-z0-9-]+)\.myhuaweicloud\.com$`, "obs.${1}.myhuaweicloud.com"),
		},
	},
	USS: {
		defaultAddressing:    AddressingPath,
		partETagIsContentMD5: true,
		// UCloud UFile：官方 S3 兼容端点 s3.{region}.ufileos.com；同时兼容 {bucket}.{region}.ufileos.com 原生桶域名（剥离桶前缀）
		// 注意：该厂商 S3 兼容端点基于公开信息实现，建议真实环境校准
		domainRules: []domainRule{
			rule(`^s3\.([a-z0-9-]+)\.ufileos\.com$`, "s3.${1}.ufileos.com"),
			rule(`^[a-z0-9-]+\.s3\.([a-z0-9-]+)\.ufileos\.com$`, "s3.${1}.ufileos.com"),
			rule(`^[a-z0-9-]+\.([a-z0-9-]+)\.ufileos\.com$`, "s3.${1}.ufileos.com"),
		},
	},
}

// rule 构造一条归一化规则（pattern 必须是编译期常量，编译失败即 panic）
func rule(pattern, replace string) domainRule {
	return domainRule{pattern: regexp.MustCompile(pattern), replace: replace}
}

// List 返回代码里已注册的厂商清单（顺序固定），implemented 由行为表推导。
func List() []Provider {
	out := make([]Provider, 0, len(providerOrder))
	for _, p := range providerOrder {
		_, ok := behaviors[p.Value]
		p.Implemented = ok
		out = append(out, p)
	}
	return out
}

// Known 判断 provider 是否在白名单内。
func Known(p string) bool {
	for _, item := range providerOrder {
		if item.Value == p {
			return true
		}
	}
	return false
}

// Implemented 判断 provider 的行为表是否已实现。
func Implemented(p string) bool {
	_, ok := behaviors[p]
	return ok
}

// CanonicalEndpoint 归一化 endpoint 到该 provider 的规范形态（不归一化会让同一物理桶出现两行）。
func CanonicalEndpoint(p, endpoint string) (string, error) {
	b, ok := behaviors[p]
	if !ok {
		if Known(p) {
			return "", ErrNotImplemented
		}
		return "", ErrUnknownProvider
	}

	host := normalizeHost(endpoint)
	if host == "" {
		return "", ErrInvalidEndpoint
	}

	for _, r := range b.domainRules {
		if r.pattern.MatchString(host) {
			return r.pattern.ReplaceAllString(host, r.replace), nil
		}
	}
	return "", ErrEndpointMismatch
}

// AddressingStyle 推导寻址方式：按 provider 默认，命中 vhost-only 区域白名单则强制 vhost。
func AddressingStyle(p, region string) (string, error) {
	b, ok := behaviors[p]
	if !ok {
		if Known(p) {
			return "", ErrNotImplemented
		}
		return "", ErrUnknownProvider
	}

	for _, r := range b.vhostOnlyRegions {
		if strings.EqualFold(r, region) {
			return AddressingVhost, nil
		}
	}
	return b.defaultAddressing, nil
}

// NormalizeRegion 归一化区域：剥掉 {provider}- / s3- 前缀（容错把端点主机名前缀当 region 的情况）。
// 例：oss-cn-beijing → cn-beijing、cos-ap-guangzhou → ap-guangzhou；无已知前缀原样返回。
func NormalizeRegion(p, region string) string {
	r := strings.ToLower(strings.TrimSpace(region))
	for _, prefix := range []string{p + "-", "s3-"} {
		if strings.HasPrefix(r, prefix) {
			if trimmed := strings.TrimPrefix(r, prefix); trimmed != "" {
				return trimmed
			}
		}
	}
	return r
}

// PartETagIsContentMD5 判断该 provider 的分片 ETag 是否等于分片内容 MD5（未开服务端加密时成立）。
func PartETagIsContentMD5(p string) bool {
	b, ok := behaviors[p]
	return ok && b.partETagIsContentMD5
}

// CanonicalBaseURL 归一化公网访问地址前缀：去首尾空白与尾斜杠。
func CanonicalBaseURL(u string) string {
	return strings.TrimRight(strings.TrimSpace(u), "/")
}

// HostOf 从 URL 或裸域名中取出小写 host（去空白、去 scheme、去 path/query/fragment）。
func HostOf(u string) string { return normalizeHost(u) }

// normalizeHost 把 endpoint 归一化为纯 host：去空白、小写、去 scheme、去 path/query/fragment。
func normalizeHost(endpoint string) string {
	s := strings.ToLower(strings.TrimSpace(endpoint))
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}
	return s
}
