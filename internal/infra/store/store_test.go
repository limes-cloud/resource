package store

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/pkg/provider"
)

// testCtx 构造可用的 core.Context
func testCtx() core.Context { return core.MustContext(context.Background()) }

func storeRow(keyword string, status uint8) *entity.Store {
	cdnKey := "cdn-key"
	return &entity.Store{
		Keyword:       keyword,
		Name:          keyword,
		Provider:      provider.BOS,
		Endpoint:      "s3.bj.bcebos.com",
		Region:        "bj",
		Bucket:        keyword,
		AK:            "ak",
		SK:            "sk",
		PublicBaseUrl: "https://" + keyword + ".cdn.example.com",
		CdnAuthType:   entity.CdnAuthNone,
		CdnAuthKey:    &cdnKey,
		Status:        &status,
	}
}

// TestGenTemporaryURLNone none 鉴权：公有读，直接拼 base/key。
func TestGenTemporaryURLNone(t *testing.T) {
	row := storeRow("bdemo", 1)
	client, err := NewClient(row)
	if err != nil {
		t.Fatalf("NewClient err=%v", err)
	}
	got, err := client.GenTemporaryURL(testCtx(), "abc.png", time.Hour)
	if err != nil {
		t.Fatalf("GenTemporaryURL err=%v", err)
	}
	if want := "https://bdemo.cdn.example.com/abc.png"; got != want {
		t.Fatalf("GenTemporaryURL=%q want %q", got, want)
	}
}

func TestGenTemporaryURLTypeA(t *testing.T) {
	row := storeRow("bdemo", 1)
	row.CdnAuthType = entity.CdnAuthTypeA
	row.PublicBaseUrl = "https://cdn.example.com/"
	client, err := NewClient(row)
	if err != nil {
		t.Fatalf("NewClient err=%v", err)
	}

	got, err := client.GenTemporaryURL(testCtx(), "abc.png", time.Hour)
	if err != nil {
		t.Fatalf("GenTemporaryURL err=%v", err)
	}
	// 形态：{base}/{12 位过期时间}/{32 位签名}/{对象名}；base 已去尾斜杠（不出现双斜杠）
	parts := strings.Split(got, "/")
	if len(parts) != 6 || parts[0] != "https:" || parts[1] != "" || parts[2] != "cdn.example.com" {
		t.Fatalf("GenTemporaryURL=%q 形态不符合预期（base 可能未去尾斜杠）", got)
	}
	if len(parts[3]) != 12 || len(parts[4]) != 32 || parts[5] != "abc.png" {
		t.Fatalf("GenTemporaryURL=%q 的 过期时间/签名/对象名 段不符合预期", got)
	}
}

func TestGenTemporaryURLTypeB(t *testing.T) {
	row := storeRow("bdemo", 1)
	row.CdnAuthType = entity.CdnAuthTypeB
	row.PublicBaseUrl = "https://cdn.example.com/"
	client, err := NewClient(row)
	if err != nil {
		t.Fatalf("NewClient err=%v", err)
	}

	got, err := client.GenTemporaryURL(testCtx(), "abc.png", time.Hour)
	if err != nil {
		t.Fatalf("GenTemporaryURL err=%v", err)
	}
	// 形态：{base}/{对象名}?auth_key={ts}-0-0-{md5}；base 去尾斜杠
	base, query, ok := strings.Cut(got, "?")
	if !ok || base != "https://cdn.example.com/abc.png" {
		t.Fatalf("GenTemporaryURL=%q 基础段不符合预期", got)
	}
	authKey, ok := strings.CutPrefix(query, "auth_key=")
	if !ok {
		t.Fatalf("GenTemporaryURL=%q 缺少 auth_key", got)
	}
	seg := strings.Split(authKey, "-")
	if len(seg) != 4 || seg[1] != "0" || seg[2] != "0" {
		t.Fatalf("auth_key=%q 形态不符合预期", authKey)
	}
	ts, sign := seg[0], seg[3]
	if want := md5Hex("/abc.png-" + ts + "-0-0-cdn-key"); sign != want {
		t.Fatalf("Type-B 签名=%q want %q", sign, want)
	}
}

func TestGenTemporaryURLTypeC(t *testing.T) {
	row := storeRow("bdemo", 1)
	row.CdnAuthType = entity.CdnAuthTypeC
	row.PublicBaseUrl = "https://cdn.example.com/"
	client, err := NewClient(row)
	if err != nil {
		t.Fatalf("NewClient err=%v", err)
	}

	got, err := client.GenTemporaryURL(testCtx(), "abc.png", time.Hour)
	if err != nil {
		t.Fatalf("GenTemporaryURL err=%v", err)
	}
	// 形态：{base}/{md5}/{hex(ts)}/{对象名}
	parts := strings.Split(got, "/")
	if len(parts) != 6 || parts[0] != "https:" || parts[1] != "" || parts[2] != "cdn.example.com" || parts[5] != "abc.png" {
		t.Fatalf("GenTemporaryURL=%q 形态不符合预期", got)
	}
	sign, hexTS := parts[3], parts[4]
	ts, err := strconv.ParseInt(hexTS, 16, 64)
	if err != nil {
		t.Fatalf("时间戳段 %q 非 16 进制：%v", hexTS, err)
	}
	if want := md5Hex("cdn-key/abc.png" + strconv.FormatInt(ts, 10)); sign != want {
		t.Fatalf("Type-C 签名=%q want %q", sign, want)
	}
}
