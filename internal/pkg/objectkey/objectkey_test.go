package objectkey

import (
	"strings"
	"testing"
)

// md5Hex 造一个合法的 32 位小写 hex 摘要
func md5Hex() string { return strings.Repeat("a", 32) }

func TestObjectName(t *testing.T) {
	md5 := md5Hex()
	cases := []struct {
		name string
		md5  string
		ext  string
		want string
	}{
		{"带后缀", md5, "png", md5 + ".png"},
		{"无后缀", md5, "", md5},
		{"空 md5 无后缀", "", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ObjectName(c.md5, c.ext); got != c.want {
				t.Fatalf("ObjectName(%q,%q)=%q want %q", c.md5, c.ext, got, c.want)
			}
		})
	}
}

func TestExportObjectName(t *testing.T) {
	md5 := md5Hex()
	if got, want := ExportObjectName(md5), "export/"+md5+".zip"; got != want {
		t.Fatalf("ExportObjectName=%q want %q", got, want)
	}
}

func TestParseObjectNameRoundTrip(t *testing.T) {
	md5 := md5Hex()
	for _, ext := range []string{"", "png", "zip", "mp4"} {
		name := ObjectName(md5, ext)
		gotMd5, gotExt, err := ParseObjectName(name)
		if err != nil {
			t.Fatalf("ParseObjectName(%q) err=%v", name, err)
		}
		if gotMd5 != md5 || gotExt != ext {
			t.Fatalf("ParseObjectName(%q)=(%q,%q) want (%q,%q)", name, gotMd5, gotExt, md5, ext)
		}
	}
}

// TestParseObjectNameUsesFilepathExt 锁死"禁止按段数切分"：
func TestParseObjectNameUsesFilepathExt(t *testing.T) {
	name := md5Hex() + ".tar.gz"
	if _, _, err := ParseObjectName(name); err == nil {
		t.Fatalf("ParseObjectName(%q) 应报错（多段后缀整体属于 md5 段）", name)
	}
}

func TestParseObjectNameInvalid(t *testing.T) {
	md5 := md5Hex()
	cases := []struct {
		name  string
		input string
	}{
		{"空串", ""},
		{"md5 太短", "abc"},
		{"md5 含大写", strings.Repeat("A", 32)},
		{"md5 非 hex", strings.Repeat("z", 32)},
		{"后缀含大写", md5 + ".PNG"},
		{"后缀含点", md5 + ".b.c"},
		{"后缀超长", md5 + "." + strings.Repeat("a", 17)},
		{"带路径分隔符", "export/" + md5 + ".zip"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, _, err := ParseObjectName(c.input); err == nil {
				t.Fatalf("ParseObjectName(%q) 应报错", c.input)
			}
		})
	}
}

func TestIsLowerHex32(t *testing.T) {
	if !IsLowerHex32(md5Hex()) {
		t.Fatal("32 位小写 hex 应通过")
	}
	for _, s := range []string{"", "abc", strings.Repeat("A", 32), strings.Repeat("g", 32), md5Hex() + "a"} {
		if IsLowerHex32(s) {
			t.Fatalf("IsLowerHex32(%q) 应为 false", s)
		}
	}
}
