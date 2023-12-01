package aliyun

import (
	"bytes"
	"errors"
	"io"
	"resource/pkg/store"
	"strconv"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type aliyun struct {
	bucket *oss.Bucket
}

type upload struct {
	bucket *oss.Bucket
	upload oss.InitiateMultipartUploadResult
}

func New(conf *store.Config) (store.Store, error) {
	if conf.Endpoint == "" || conf.Key == "" || conf.Secret == "" {
		return nil, errors.New("store config error")
	}

	client, err := oss.New(conf.Endpoint, conf.Key, conf.Secret)
	if err != nil {
		return nil, err
	}

	bucket, err := client.Bucket(conf.Bucket)
	if err != nil {
		return nil, err
	}

	return &aliyun{
		bucket: bucket,
	}, nil
}

func (s *aliyun) PutBytes(key string, in []byte) error {
	return s.bucket.PutObject(key, bytes.NewReader(in))
}

func (s *aliyun) Put(key string, r io.Reader) error {
	return s.bucket.PutObject(key, r)
}

func (s *aliyun) PutFromLocal(key string, localPath string) error {
	return s.bucket.PutObjectFromFile(key, localPath)
}

func (s *aliyun) Get(key string) (io.ReadCloser, error) {
	return s.bucket.GetObject(key)
}

func (s *aliyun) Delete(key string) error {
	return s.bucket.DeleteObject(key)
}

func (s *aliyun) Size(key string) (int64, error) {
	header, err := s.bucket.GetObjectDetailedMeta(key)
	if err != nil {
		return 0, err
	}

	length := header.Get("Content-Length")
	return strconv.ParseInt(length, 10, 64)
}

func (s *aliyun) Exists(key string) (bool, error) {
	return s.bucket.IsObjectExist(key)
}

func (s *aliyun) NewPutChunk(key string) (store.PutChunk, error) {
	up, err := s.bucket.InitiateMultipartUpload(key)
	if err != nil {
		return nil, err
	}
	return &upload{
		upload: up,
		bucket: s.bucket,
	}, nil
}

func (s *aliyun) NewPutChunkByUploadID(key, id string) (store.PutChunk, error) {
	up := oss.InitiateMultipartUploadResult{
		XMLName:  struct{ Space, Local string }{Space: "", Local: "InitiateMultipartUploadResult"},
		UploadID: id,
		Key:      key,
		Bucket:   s.bucket.BucketName,
	}
	return &upload{
		upload: up,
		bucket: s.bucket,
	}, nil
}

func (u *upload) UploadedChunkIndex() []int {
	var arr []int
	lsRes, _ := u.bucket.ListUploadedParts(u.upload)
	for _, item := range lsRes.UploadedParts {
		arr = append(arr, item.PartNumber)
	}
	return arr
}

func (u *upload) ChunkCount() int {
	lsRes, _ := u.bucket.ListUploadedParts(u.upload)
	return len(lsRes.UploadedParts)
}

func (u *upload) UploadID() string {
	return u.upload.UploadID
}

func (u *upload) Append(r io.Reader, index int) error {
	all, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	_, err = u.bucket.UploadPart(u.upload, bytes.NewReader(all), int64(len(all)), index)
	return err
}

func (u *upload) AppendBytes(r []byte, index int) error {
	_, err := u.bucket.UploadPart(u.upload, bytes.NewReader(r), int64(len(r)), index)
	return err
}

func (u *upload) Abort() error {
	return u.bucket.AbortMultipartUpload(u.upload)
}

func (u *upload) Complete() error {
	lsRes, err := u.bucket.ListUploadedParts(u.upload)
	if err != nil {
		return err
	}

	var parts []oss.UploadPart
	for _, item := range lsRes.UploadedParts {
		parts = append(parts, oss.UploadPart{
			XMLName:    item.XMLName,
			PartNumber: item.PartNumber,
			ETag:       item.ETag,
		})
	}

	_, err = u.bucket.CompleteMultipartUpload(u.upload, parts)
	return err
}
