package pattern

import "testing"

func TestEscapeLike(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"abc", "abc"},
		{`\`, `\\`},
		{`%`, `\%`},
		{`_`, `\_`},
		{`a%b_c\d`, `a\%b\_c\\d`},
		// 顺序不能反：必须先转义 \，再转义 %/_，否则新加的转义符会被再转一遍
		{`\%`, `\\\%`},
		{`\_`, `\\\_`},
		{`/图片/头像`, `/图片/头像`},
	}
	for _, c := range cases {
		if got := EscapeLike(c.in); got != c.want {
			t.Fatalf("EscapeLike(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestPrefixPattern(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"/a", `/a/%`},
		{"/图_片", `/图\_片/%`},
		{"/100%", `/100\%/%`},
		{`/a\b`, `/a\\b/%`},
	}
	for _, c := range cases {
		if got := PrefixPattern(c.path); got != c.want {
			t.Fatalf("PrefixPattern(%q)=%q want %q", c.path, got, c.want)
		}
	}
}
