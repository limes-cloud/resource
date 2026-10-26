package entity

import (
	"fmt"
	"github.com/limes-cloud/kratosx/model"

	"github.com/limes-cloud/resource/internal/pkg/provider"
)

// store.cdn_auth_type 的取值
const (
	CdnAuthNone      = "none"       // 公有读
	CdnAuthTypeA     = "cdn_a"      // CDN Type-A：路径 /{YYYYMMDDHHMM}/{md5}/{key} 签名
	CdnAuthTypeB     = "cdn_b"      // CDN Type-B：query ?auth_key={ts}-0-0-{md5} 签名
	CdnAuthTypeC     = "cdn_c"      // CDN Type-C：路径 /{md5}/{hex(ts)}/{key} 签名
	CdnAuthS3Presign = "s3_presign" // 厂商 SigV4 预签名
)

// Store 对象存储后端
type Store struct {
	Keyword       string  `json:"keyword"       gorm:"column:keyword"`         // 后端标识
	Provider      string  `json:"provider"      gorm:"column:provider"`        // 厂商标识：aws|bos|oss|cos|obs|minio|uss
	Name          string  `json:"name"          gorm:"column:name"`            // 显示名称
	Endpoint      string  `json:"endpoint"      gorm:"column:endpoint"`        // 厂商API域名（归一化后的形态）
	Region        string  `json:"region"        gorm:"column:region"`          // 地域
	Bucket        string  `json:"bucket"        gorm:"column:bucket"`          // 桶名
	AK            string  `json:"ak"            gorm:"column:ak"`              // 访问密钥ID
	SK            string  `json:"sk"            gorm:"column:sk"`              // 访问密钥；永不进响应
	PublicBaseUrl string  `json:"publicBaseUrl" gorm:"column:public_base_url"` // 公网访问地址前缀
	CdnAuthType   string  `json:"cdnAuthType"   gorm:"column:cdn_auth_type"`   // 临时URL鉴权方式：none|cdn_a|cdn_b|cdn_c|s3_presign
	CdnAuthKey    *string `json:"cdnAuthKey"    gorm:"column:cdn_auth_key"`    // CDN 鉴权 key；NULL = 未配置
	Status        *uint8  `json:"status"        gorm:"column:status"`          // 状态：1启用 0停用；0 是合法零值，用指针避免更新被跳过
	Description   *string `json:"description"   gorm:"column:description"`     // 备注；NULL = 无
	UpdatedBy     uint32  `json:"updatedBy"     gorm:"column:updated_by"`      // 最后修改人
	model.BaseModel
}

// CheckCDN 校验 endpoint / public_base_url / cdn_auth_type 跨字段自洽性（proto 表达不了，在 app 层调用）。
func (s *Store) CheckCDN() error {
	base := provider.CanonicalBaseURL(s.PublicBaseUrl)
	if base == "" {
		return fmt.Errorf("public_base_url 为空")
	}
	host := provider.HostOf(base)

	switch s.CdnAuthType {
	case CdnAuthTypeA, CdnAuthTypeB, CdnAuthTypeC:
		if s.CdnAuthKey == nil || *s.CdnAuthKey == "" {
			return fmt.Errorf("%s 必须配置 cdn_auth_key", s.CdnAuthType)
		}
		if host == s.Endpoint {
			return fmt.Errorf("%s 配厂商域名：没有执行方，cdn_auth_key 白配", s.CdnAuthType)
		}
	case CdnAuthS3Presign:
		if host != s.Endpoint {
			return fmt.Errorf("s3_presign 配 CDN 域名：SigV4 的 host 签名会失效、整域 403")
		}
	}
	return nil
}
