package local

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Chunk struct {
	UploadID string `json:"upload_id"`
	Index    int    `json:"index"`
	Sha      string `json:"sha"`
	Data     string `json:"data"`
	Size     int    `json:"size"`
}

func (c *Chunk) dir(p string) string {
	return p + "/chunk/" + c.UploadID
}

func (c *Chunk) file(dir string, index int) string {
	return fmt.Sprintf("%s/%d.chunk", dir, index)
}

func (c *Chunk) Add(dir string) error {
	dir = c.dir(dir)
	_ = os.MkdirAll(dir, os.ModePerm)
	return os.WriteFile(c.file(dir, c.Index), []byte(c.Data), 0666)
}

func (c *Chunk) Parts(dir string, uploadId string) ([]*Chunk, error) {
	c.UploadID = uploadId
	var chunks []*Chunk
	// 遍历目录
	dir = c.dir(dir)
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, item := range files {
		if item.IsDir() {
			continue
		}
		// 解析文件名 .chunk的保留
		arr := strings.Split(item.Name(), ".")
		if len(arr) < 2 {
			continue
		}
		if arr[1] != "chunk" {
			continue
		}
		index, _ := strconv.Atoi(arr[0])
		b, _ := os.ReadFile(c.file(dir, index))
		chunks = append(chunks, &Chunk{
			UploadID: uploadId,
			Index:    index,
			Data:     string(b),
		})
	}
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].Index < chunks[j].Index
	})
	return chunks, nil
}

func (c *Chunk) UploadedChunkIndex(dir string, uploadId string) ([]int, error) {
	c.UploadID = uploadId
	var chunks []int
	// 遍历目录
	dir = c.dir(dir)
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, item := range files {
		if item.IsDir() {
			continue
		}
		// 解析文件名 .chunk的保留
		arr := strings.Split(item.Name(), ".")
		if len(arr) < 2 {
			continue
		}
		if arr[1] != "chunk" {
			continue
		}
		index, _ := strconv.Atoi(arr[0])
		chunks = append(chunks, index)
	}
	sort.Ints(chunks)
	return chunks, nil
}

func (c *Chunk) Delete(dir string, uploadId string) error {
	c.UploadID = uploadId
	dir = c.dir(dir)
	return os.RemoveAll(dir)
}
