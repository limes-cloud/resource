package chunk

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"testing"
)

// hexOf 计算字符串的 MD5 hex（模拟"某一片内容的摘要"）
func hexOf(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

// digestOf 计算"分片摘要的摘要"（测试内的独立实现，用于交叉验证）
func digestOf(hexes ...string) string {
	joined := ""
	for _, h := range hexes {
		joined += h
	}
	return hexOf(joined)
}

func TestSize(t *testing.T) {
	const fiveMB = 5 * MB
	cases := []struct {
		name      string
		totalSize uint64
		want      uint32
		wantErr   error
	}{
		{"空文件直接拒绝", 0, 0, ErrEmptyFile},
		{"1 字节", 1, fiveMB, nil},
		{"恰好 5MB", fiveMB, fiveMB, nil},
		{"10000×5MB 恰好分界", 10000 * fiveMB, fiveMB, nil},
		{"10000×5MB+1 跨分界（片数仍≤10000）", 10000*fiveMB + 1, 6 * MB, nil},
		{"物理天花板", MaxTotalSize, 4095 * MB, nil},
		{"超天花板即报错不截断", MaxTotalSize + 1, 0, ErrFileTooLarge},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Size(c.totalSize)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("Size(%d) err=%v want %v", c.totalSize, err, c.wantErr)
			}
			if err == nil && got != c.want {
				t.Fatalf("Size(%d)=%d want %d", c.totalSize, got, c.want)
			}
		})
	}
}

func TestSizeResultIsMBAligned(t *testing.T) {
	for _, total := range []uint64{1, fiveMBBytes() - 1, fiveMBBytes(), 10000*fiveMBBytes() + 1, MaxTotalSize} {
		got, err := Size(total)
		if err != nil {
			t.Fatalf("Size(%d) unexpected err=%v", total, err)
		}
		if got%MB != 0 {
			t.Fatalf("Size(%d)=%d 不是 1MB 的整数倍", total, got)
		}
		if got < MinChunkSize {
			t.Fatalf("Size(%d)=%d 低于 5MB 下限", total, got)
		}
	}
}

func fiveMBBytes() uint64 { return 5 * MB }

func TestCount(t *testing.T) {
	const fiveMB = 5 * MB
	cases := []struct {
		name      string
		totalSize uint64
		chunkSize uint32
		want      uint32
	}{
		{"1 字节 1 片", 1, fiveMB, 1},
		{"恰好 5MB 1 片", fiveMB, fiveMB, 1},
		{"10000×5MB 恰好 10000 片", 10000 * fiveMB, fiveMB, 10000},
		{"跨分界后 8334 片", 10000*fiveMB + 1, 6 * MB, 8334},
		{"天花板 10000 片", MaxTotalSize, 4095 * MB, 10000},
		{"chunkSize 为 0", 100, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Count(c.totalSize, c.chunkSize); got != c.want {
				t.Fatalf("Count(%d,%d)=%d want %d", c.totalSize, c.chunkSize, got, c.want)
			}
		})
	}
}

// TestSizeCountAlwaysWithinChunkCount 不变量：任意合法 totalSize 下片数必然 ≤ 10000
func TestSizeCountAlwaysWithinChunkCount(t *testing.T) {
	samples := []uint64{
		1, MB, 5 * MB, 10000 * 5 * MB, 10000*5*MB + 1,
		100 * 1024 * MB, 10 * 1024 * 1024 * MB, MaxTotalSize,
	}
	for _, total := range samples {
		size, err := Size(total)
		if err != nil {
			t.Fatalf("Size(%d) unexpected err=%v", total, err)
		}
		if got := Count(total, size); got > MaxChunkCount {
			t.Fatalf("totalSize=%d size=%d count=%d 超过 %d", total, size, got, MaxChunkCount)
		}
	}
}

func TestDigestSinglePartIsHashOfHash(t *testing.T) {
	fileDigest := hexOf("hello world")
	got, err := Digest([]Part{{Number: 1, Digest: fileDigest}})
	if err != nil {
		t.Fatalf("Digest err=%v", err)
	}
	// 单片同规：对文件 md5 再取一次 md5，而不是直接用文件 md5
	if want := hexOf(fileDigest); got != want {
		t.Fatalf("Digest(N=1)=%q want %q", got, want)
	}
	if got == fileDigest {
		t.Fatal("Digest(N=1) 退化成文件 md5——单片与分片规则不一致，去重与秒传会同时失效")
	}
}

func TestDigestMultiPartIsHexTextConcat(t *testing.T) {
	a, b := hexOf("part-1"), hexOf("part-2")
	got, err := Digest([]Part{{Number: 1, Digest: a}, {Number: 2, Digest: b}})
	if err != nil {
		t.Fatalf("Digest err=%v", err)
	}
	if want := digestOf(a, b); got != want {
		t.Fatalf("Digest=%q want %q", got, want)
	}
}

func TestDigestOutOfOrderNormalized(t *testing.T) {
	a, b, c := hexOf("1"), hexOf("2"), hexOf("3")
	ordered, err := Digest([]Part{{Number: 1, Digest: a}, {Number: 2, Digest: b}, {Number: 3, Digest: c}})
	if err != nil {
		t.Fatalf("Digest ordered err=%v", err)
	}
	shuffled, err := Digest([]Part{{Number: 3, Digest: c}, {Number: 1, Digest: a}, {Number: 2, Digest: b}})
	if err != nil {
		t.Fatalf("Digest shuffled err=%v", err)
	}
	if ordered != shuffled {
		t.Fatalf("乱序输入未按片号升序归一：%q vs %q", ordered, shuffled)
	}
}

func TestDigestErrors(t *testing.T) {
	h := hexOf("x")
	cases := []struct {
		name  string
		parts []Part
	}{
		{"空输入", nil},
		{"摘要非法", []Part{{Number: 1, Digest: "not-hex"}}},
		{"片号为 0", []Part{{Number: 0, Digest: h}}},
		{"片号重复", []Part{{Number: 1, Digest: h}, {Number: 1, Digest: h}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := Digest(c.parts); err == nil {
				t.Fatalf("Digest(%v) 应报错", c.parts)
			}
		})
	}
}

func TestNormalizeETag(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`"abc123"`, "abc123"},
		{`ABC123`, "abc123"},
		{`  "ABC123"  `, "abc123"},
	}
	for _, c := range cases {
		if got := NormalizeETag(c.in); got != c.want {
			t.Fatalf("NormalizeETag(%q)=%q want %q", c.in, got, c.want)
		}
	}
}
