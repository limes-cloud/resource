package provider

import (
	"errors"
	"testing"
)

func TestCanonicalEndpointBOSSpellings(t *testing.T) {
	// 四种拼写差异（域名族 / vhost / scheme / 尾斜杠）必须归一到同一值
	const want = "s3.bj.bcebos.com"
	inputs := []string{
		"s3.bj.bcebos.com",             // S3 兼容形态（规范值）
		"bj.bcebos.com",                // 原生域名族
		"bdemo.bj.bcebos.com",          // vhost 形态（桶名进了 host）
		"bdemo.s3.bj.bcebos.com",       // vhost over S3 形态
		"https://s3.bj.bcebos.com",     // 带 scheme
		"https://bdemo.bj.bcebos.com/", // 带 scheme + 尾斜杠
		"HTTPS://S3.BJ.BCEBOS.COM/",    // 大写
	}
	for _, in := range inputs {
		got, err := CanonicalEndpoint(BOS, in)
		if err != nil {
			t.Fatalf("CanonicalEndpoint(bos,%q) err=%v", in, err)
		}
		if got != want {
			t.Fatalf("CanonicalEndpoint(bos,%q)=%q want %q", in, got, want)
		}
	}
}

func TestCanonicalEndpointAWS(t *testing.T) {
	const want = "s3.us-east-1.amazonaws.com"
	for _, in := range []string{
		"s3.us-east-1.amazonaws.com",
		"s3-us-east-1.amazonaws.com",
		"bdemo.s3.us-east-1.amazonaws.com",
		"https://bdemo.s3-us-east-1.amazonaws.com/",
	} {
		got, err := CanonicalEndpoint(AWS, in)
		if err != nil {
			t.Fatalf("CanonicalEndpoint(aws,%q) err=%v", in, err)
		}
		if got != want {
			t.Fatalf("CanonicalEndpoint(aws,%q)=%q want %q", in, got, want)
		}
	}
}

func TestCanonicalEndpointMinioKeepsPort(t *testing.T) {
	got, err := CanonicalEndpoint(MINIO, "HTTP://MinIO.Local:9000/")
	if err != nil {
		t.Fatalf("CanonicalEndpoint(minio) err=%v", err)
	}
	if want := "minio.local:9000"; got != want {
		t.Fatalf("CanonicalEndpoint(minio)=%q want %q", got, want)
	}
}

func TestCanonicalEndpointOSS(t *testing.T) {
	// 国内 oss-cn-{region} / 国际 oss-{region} 两族，桶前缀 vhost 形态剥离回裸端点
	inputs := []string{
		"oss-cn-beijing.aliyuncs.com",
		"bdemo.oss-cn-beijing.aliyuncs.com",
		"https://bdemo.oss-cn-beijing.aliyuncs.com/",
		"oss-ap-southeast-1.aliyuncs.com",
		"bdemo.oss-ap-southeast-1.aliyuncs.com",
	}
	wants := []string{
		"oss-cn-beijing.aliyuncs.com",
		"oss-cn-beijing.aliyuncs.com",
		"oss-cn-beijing.aliyuncs.com",
		"oss-ap-southeast-1.aliyuncs.com",
		"oss-ap-southeast-1.aliyuncs.com",
	}
	for i, in := range inputs {
		got, err := CanonicalEndpoint(OSS, in)
		if err != nil {
			t.Fatalf("CanonicalEndpoint(oss,%q) err=%v", in, err)
		}
		if got != wants[i] {
			t.Fatalf("CanonicalEndpoint(oss,%q)=%q want %q", in, got, wants[i])
		}
	}
}

func TestCanonicalEndpointCOS(t *testing.T) {
	const want = "cos.ap-guangzhou.myqcloud.com"
	for _, in := range []string{
		"cos.ap-guangzhou.myqcloud.com",
		"demo-1300000000.cos.ap-guangzhou.myqcloud.com", // 桶名带 -{appid} 后缀
		"HTTPS://demo-1300000000.cos.ap-guangzhou.myqcloud.com/",
	} {
		got, err := CanonicalEndpoint(COS, in)
		if err != nil {
			t.Fatalf("CanonicalEndpoint(cos,%q) err=%v", in, err)
		}
		if got != want {
			t.Fatalf("CanonicalEndpoint(cos,%q)=%q want %q", in, got, want)
		}
	}
}

func TestCanonicalEndpointOBS(t *testing.T) {
	const want = "obs.cn-north-4.myhuaweicloud.com"
	for _, in := range []string{
		"obs.cn-north-4.myhuaweicloud.com",
		"bdemo.obs.cn-north-4.myhuaweicloud.com",
		"https://bdemo.obs.cn-north-4.myhuaweicloud.com/",
	} {
		got, err := CanonicalEndpoint(OBS, in)
		if err != nil {
			t.Fatalf("CanonicalEndpoint(obs,%q) err=%v", in, err)
		}
		if got != want {
			t.Fatalf("CanonicalEndpoint(obs,%q)=%q want %q", in, got, want)
		}
	}
}

func TestCanonicalEndpointUSS(t *testing.T) {
	inputs := []string{
		"s3.cn-bj.ufileos.com",
		"bdemo.cn-bj.ufileos.com", // 原生桶域名（vhost），剥离桶前缀
		"https://bdemo.cn-bj.ufileos.com/",
	}
	wants := []string{
		"s3.cn-bj.ufileos.com",
		"s3.cn-bj.ufileos.com",
		"s3.cn-bj.ufileos.com",
	}
	for i, in := range inputs {
		got, err := CanonicalEndpoint(USS, in)
		if err != nil {
			t.Fatalf("CanonicalEndpoint(uss,%q) err=%v", in, err)
		}
		if got != wants[i] {
			t.Fatalf("CanonicalEndpoint(uss,%q)=%q want %q", in, got, wants[i])
		}
	}
}

func TestCanonicalEndpointErrors(t *testing.T) {
	cases := []struct {
		name     string
		provider string
		endpoint string
		want     error
	}{
		{"不在白名单", "not-a-provider", "s3.bj.bcebos.com", ErrUnknownProvider},
		{"host 不匹配域名族", BOS, "not-bcebos.example.com", ErrEndpointMismatch},
		{"空 endpoint", BOS, "   ", ErrInvalidEndpoint},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := CanonicalEndpoint(c.provider, c.endpoint); !errors.Is(err, c.want) {
				t.Fatalf("CanonicalEndpoint(%q,%q) err=%v want %v", c.provider, c.endpoint, err, c.want)
			}
		})
	}
}

func TestCanonicalEndpointIdempotent(t *testing.T) {
	// 归一化必须是幂等的：启动自检断言 CanonicalEndpoint(provider, endpoint) == endpoint
	first, err := CanonicalEndpoint(BOS, "bdemo.bj.bcebos.com")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	second, err := CanonicalEndpoint(BOS, first)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if first != second {
		t.Fatalf("归一化不幂等：%q -> %q", first, second)
	}
}

func TestAddressingStyle(t *testing.T) {
	cases := []struct {
		name     string
		provider string
		region   string
		want     string
		wantErr  error
	}{
		{"bos 默认 path", BOS, "bj", AddressingPath, nil},
		{"aws 默认 path", AWS, "us-east-1", AddressingPath, nil},
		{"minio 默认 path", MINIO, "", AddressingPath, nil},
		{"oss 默认 vhost（阿里云强制三级域名）", OSS, "cn-beijing", AddressingVhost, nil},
		{"cos 默认 path", COS, "ap-guangzhou", AddressingPath, nil},
		{"obs 默认 path", OBS, "cn-north-4", AddressingPath, nil},
		{"uss 默认 path", USS, "cn-bj", AddressingPath, nil},
		{"不在白名单", "not-a-provider", "x", "", ErrUnknownProvider},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := AddressingStyle(c.provider, c.region)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("AddressingStyle(%q,%q) err=%v want %v", c.provider, c.region, err, c.wantErr)
			}
			if err == nil && got != c.want {
				t.Fatalf("AddressingStyle(%q,%q)=%q want %q", c.provider, c.region, got, c.want)
			}
		})
	}
}

func TestPartETagIsContentMD5(t *testing.T) {
	// 全部厂商均为 S3 兼容协议：未开服务端加密时，分片 ETag = 该分片内容 MD5
	for _, p := range []string{AWS, BOS, OSS, COS, OBS, MINIO, USS} {
		if !PartETagIsContentMD5(p) {
			t.Fatalf("%s 未开 SSE 时能力位应为 true", p)
		}
	}
	if PartETagIsContentMD5("not-a-provider") {
		t.Fatal("不在白名单的厂商能力位应为 false")
	}
}

func TestListDerivesImplementedFromBehaviorTable(t *testing.T) {
	list := List()
	if len(list) != len(providerOrder) {
		t.Fatalf("List() 返回 %d 项，白名单 %d 项", len(list), len(providerOrder))
	}
	for _, p := range list {
		if want := Implemented(p.Value); p.Implemented != want {
			t.Fatalf("%s implemented=%v 与行为表推导值 %v 不一致", p.Value, p.Implemented, want)
		}
		if p.Label == "" {
			t.Fatalf("%s 缺少展示名", p.Value)
		}
	}
}

func TestKnownAndImplemented(t *testing.T) {
	for _, p := range []string{AWS, BOS, OSS, COS, OBS, MINIO, USS} {
		if !Known(p) {
			t.Fatalf("%s 应在白名单内", p)
		}
	}
	if Known("not-a-provider") {
		t.Fatal("白名单外应为 false")
	}
	// 全部厂商行为表已实现
	for _, p := range []string{AWS, BOS, OSS, COS, OBS, MINIO, USS} {
		if !Implemented(p) {
			t.Fatalf("%s 行为表应已实现", p)
		}
	}
}

func TestNormalizeRegion(t *testing.T) {
	cases := []struct{ name, provider, in, want string }{
		{"oss 带前缀", OSS, "oss-cn-beijing", "cn-beijing"},
		{"oss 裸 region", OSS, "cn-beijing", "cn-beijing"},
		{"oss 国际带前缀", OSS, "oss-ap-southeast-1", "ap-southeast-1"},
		{"cos 带前缀", COS, "cos-ap-guangzhou", "ap-guangzhou"},
		{"obs 带前缀", OBS, "obs-cn-north-4", "cn-north-4"},
		{"uss 带前缀", USS, "uss-cn-bj", "cn-bj"},
		{"s3 前缀兜底", BOS, "s3-bj", "bj"},
		{"大小写/空白", OSS, "  OSS-CN-SHANGHAI ", "cn-shanghai"},
		{"无前缀原样", AWS, "us-east-1", "us-east-1"},
		{"空串", OSS, "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := NormalizeRegion(c.provider, c.in); got != c.want {
				t.Fatalf("NormalizeRegion(%q,%q)=%q want %q", c.provider, c.in, got, c.want)
			}
		})
	}
}

func TestCanonicalBaseURL(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://cdn.example.com/", "https://cdn.example.com"},
		{"  https://cdn.example.com  ", "https://cdn.example.com"},
		{"https://cdn.example.com", "https://cdn.example.com"},
	}
	for _, c := range cases {
		if got := CanonicalBaseURL(c.in); got != c.want {
			t.Fatalf("CanonicalBaseURL(%q)=%q want %q", c.in, got, c.want)
		}
	}
}
