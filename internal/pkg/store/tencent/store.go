package tencent

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/tencentyun/cos-go-sdk-v5"

	store2 "github.com/limes-cloud/resource/internal/pkg/store"
)

type tencent struct {
	client *cos.Client
}

type upload struct {
	client *cos.Client
	upload *cos.InitiateMultipartUploadResult
}

func New(conf *store2.Config) (store2.Store, error) {
	if conf.Endpoint == "" || conf.Secret == "" || conf.Key == "" {
		return nil, errors.New("upload config error")
	}

	u, err := url.Parse(conf.Endpoint)
	if err != nil {
		return nil, err
	}

	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  conf.Secret,
			SecretKey: conf.Key,
		},
	})

	return &tencent{client: client}, nil
}

func (s *tencent) PutBytes(key string, in []byte) error {
	return s.Put(key, bytes.NewReader(in))
}

func (s *tencent) Put(key string, r io.Reader) error {
	response, err := s.client.Object.Put(context.Background(), key, r, nil)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return httpError(response)
	}

	return nil
}

func (s *tencent) PutFromLocal(key string, localPath string) error {
	response, err := s.client.Object.PutFromFile(context.Background(), key, localPath, nil)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return httpError(response)
	}

	return nil
}

func (s *tencent) Get(key string) (io.ReadCloser, error) {
	resp, err := s.client.Object.Get(context.Background(), key, nil)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (s *tencent) Delete(key string) error {
	_, err := s.client.Object.Delete(context.Background(), key)

	return err
}

func (s *tencent) Size(key string) (int64, error) {
	resp, err := s.client.Object.Head(context.Background(), key, nil)
	if err != nil {
		return 0, err
	}

	return resp.ContentLength, nil
}

func (s *tencent) Exists(key string) (bool, error) {
	return s.client.Object.IsExist(context.Background(), key)
}

func httpError(response *cos.Response) error {
	bt, err := io.ReadAll(response.Body)
	defer func() {
		err = response.Body.Close()
	}()
	if err != nil {
		return err
	}

	return errors.New(string(bt))
}

func (s *tencent) NewPutChunk(key string) (store2.PutChunk, error) {
	up, _, err := s.client.Object.InitiateMultipartUpload(context.Background(), key, nil)
	if err != nil {
		return nil, err
	}
	return &upload{
		upload: up,
		client: s.client,
	}, nil
}

func (s *tencent) NewPutChunkByUploadID(key, id string) (store2.PutChunk, error) {
	bucket, _, err := s.client.Bucket.Get(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	up := &cos.InitiateMultipartUploadResult{
		XMLName:  struct{ Space, Local string }{Space: "", Local: "InitiateMultipartUploadResult"},
		UploadID: id,
		Bucket:   bucket.Name,
		Key:      key,
	}
	return &upload{
		upload: up,
		client: s.client,
	}, nil
}

func (u *upload) UploadedChunkIndex() []int {
	var arr []int
	lsRes, _, _ := u.client.Object.ListParts(context.Background(), u.upload.Key, u.upload.UploadID, nil)
	for _, item := range lsRes.Parts {
		arr = append(arr, item.PartNumber)
	}
	return arr
}

func (u *upload) ChunkCount() int {
	lsRes, _, _ := u.client.Object.ListParts(context.Background(), u.upload.Key, u.upload.UploadID, nil)
	return len(lsRes.Parts)
}

func (u *upload) UploadID() string {
	return u.upload.UploadID
}

func (u *upload) Append(r io.Reader, index int) error {
	_, err := u.client.Object.UploadPart(context.Background(), u.upload.Key, u.upload.UploadID, index, r, nil)
	return err
}

func (u *upload) AppendBytes(r []byte, index int) error {
	_, err := u.client.Object.UploadPart(context.Background(), u.upload.Key, u.upload.UploadID, index, bytes.NewReader(r), nil)
	return err
}

func (u *upload) Abort() error {
	_, err := u.client.Object.AbortMultipartUpload(context.Background(), u.upload.Key, u.upload.UploadID)
	return err
}

func (u *upload) Complete() error {
	lsRes, _, err := u.client.Object.ListParts(context.Background(), u.upload.Key, u.upload.UploadID, nil)
	if err != nil {
		return err
	}

	opt := &cos.CompleteMultipartUploadOptions{}
	for _, item := range lsRes.Parts {
		opt.Parts = append(opt.Parts, cos.Object{
			Key:        item.Key,
			PartNumber: item.PartNumber,
			ETag:       item.ETag,
		})
	}

	_, _, err = u.client.Object.CompleteMultipartUpload(context.Background(), u.upload.Key, u.upload.UploadID, opt)
	return err
}
