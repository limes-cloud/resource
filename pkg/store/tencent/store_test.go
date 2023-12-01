package tencent

import (
	"bytes"
	"fmt"
	"resource/pkg/store"
	"testing"
)

func TestTencent_NewPutChunk(t *testing.T) {
	store, err := New(&store.Config{
		Endpoint: "https://interact-1301828925.cos.ap-chengdu.myqcloud.com",
		Key:      "71uMlvfL6jDZp9VhsumR8j9Mh5Qckuxo",
		Secret:   "AKIDqdIpGdw772sa8ayFLH5mJtMhopdQ3Wpx",
		Bucket:   "interact-1301828925",
	})

	if err != nil {
		t.Error(err.Error())
		return
	}

	chunk, err := store.NewPutChunk("hello.txt")
	if err != nil {
		t.Error(err.Error())
		return
	}

	for i := 0; i < 10; i++ {
		data := bytes.Repeat([]byte(fmt.Sprint(i)), 1024*1024)
		if err := chunk.Append(bytes.NewReader(data), i+1); err != nil {
			t.Error(err.Error())
			chunk.Abort()
			return
		}
	}

	if err := chunk.Complete(); err != nil {
		t.Error(err.Error())
		chunk.Abort()
	}
}
