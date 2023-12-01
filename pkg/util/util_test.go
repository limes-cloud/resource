package util

import (
	"fmt"
	"io"
	"os"
	"testing"
)

func TestSha256(t *testing.T) {
	file, err := os.Open("1.png")
	if err != nil {
		t.Error(err.Error())
		return
	}
	rb, _ := io.ReadAll(file)
	rt := string(rb)
	fmt.Println(rt)
	res := Sha256(rb)
	t.Log(res)
}
