package aliyun

import (
	"bytes"
	"fmt"
	"resource/pkg/store"
	"testing"
)

func TestAliyun_NewPutChunk(t *testing.T) {
	store, err := New(&store.Config{
		Endpoint: "oss-cn-beijing.aliyuncs.com",
		Key:      "LTAI5tMqaVkXRaJgVB26SBkB",
		Secret:   "pQyWOEmhLL54vQvAaILCJd9bOEayFi",
		Bucket:   "limes-cloud",
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
		data := bytes.Repeat([]byte(fmt.Sprint(i)), 1024*100)
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
