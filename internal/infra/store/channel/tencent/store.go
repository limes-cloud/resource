package tencent

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/limes-cloud/kratosx/pkg/lock"
	"github.com/tencentyun/cos-go-sdk-v5"

	"github.com/limes-cloud/resource/internal/infra/store/config"
	"github.com/limes-cloud/resource/internal/infra/store/types"
)

type Tencent struct {
	keyword   string
	client    *cos.Client
	expire    time.Duration
	cache     *redis.Client
	cdn       string
	antiTheft bool
}

type upload struct {
	client *cos.Client
	upload *cos.InitiateMultipartUploadResult
}

func New(conf *config.Config) (*Tencent, error) {
	if conf.Endpoint == "" || conf.Secret == "" || conf.Id == "" {
		return nil, errors.New("store config error")
	}

	u, err := url.Parse(conf.Endpoint)
	if err != nil {
		return nil, err
	}

	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  conf.Id,
			SecretKey: conf.Secret,
		},
	})

	return &Tencent{
		keyword:   conf.Keyword,
		client:    client,
		expire:    conf.TemporaryExpire,
		cache:     conf.Cache,
		cdn:       conf.Endpoint,
		antiTheft: conf.AntiTheft,
	}, nil
}

func (s *Tencent) GetKeyword() string {
	return s.keyword
}

func (s *Tencent) GenTemporaryURL(key string) (string, error) {
	if !s.antiTheft {
		return s.cdn + "/" + key, nil
	}
	var (
		err    error
		target string
		locker = lock.New(s.cache, key+":lock")
	)
	ck := fmt.Sprintf("resource:%x", md5.Sum([]byte(key)))
	err = locker.AcquireFunc(context.Background(),
		func() error {
			target, err = s.cache.Get(context.Background(), ck).Result()
			return err
		},
		func() error {
			t := time.Now().Add(s.expire).Format("200601021504")
			st := s.client.GetCredential().GetSecretKey() + t + "/" + key
			target = fmt.Sprintf("%s/%s/%s/%s",
				s.cdn,
				t,
				fmt.Sprintf("%x", md5.Sum([]byte(st))),
				key,
			)
			return s.cache.Set(context.Background(), ck, target, s.expire-10*time.Second).Err()
		},
	)
	if err != nil {
		return "", err
	}
	return target, nil
}

func (s *Tencent) VerifyTemporaryURL(key string, expire string, sign string) error {
	return nil
}

func (s *Tencent) PutBytes(key string, in []byte) error {
	return s.Put(key, bytes.NewReader(in))
}

func (s *Tencent) Put(key string, r io.Reader) error {
	response, err := s.client.Object.Put(context.Background(), key, r, nil)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return httpError(response)
	}

	return nil
}

func (s *Tencent) PutFromLocal(key string, localPath string) error {
	response, err := s.client.Object.PutFromFile(context.Background(), key, localPath, nil)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return httpError(response)
	}

	return nil
}

func (s *Tencent) Get(key string) (io.ReadCloser, error) {
	resp, err := s.client.Object.Get(context.Background(), key, nil)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (s *Tencent) Delete(key string) error {
	_, err := s.client.Object.Delete(context.Background(), key)

	return err
}

func (s *Tencent) Size(key string) (int64, error) {
	resp, err := s.client.Object.Head(context.Background(), key, nil)
	if err != nil {
		return 0, err
	}

	return resp.ContentLength, nil
}

func (s *Tencent) Exists(key string) (bool, error) {
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

func (s *Tencent) NewPutChunk(key string) (types.PutChunk, error) {
	up, _, err := s.client.Object.InitiateMultipartUpload(context.Background(), key, nil)
	if err != nil {
		return nil, err
	}
	return &upload{
		upload: up,
		client: s.client,
	}, nil
}

func (s *Tencent) NewPutChunkByUploadID(key, id string) (types.PutChunk, error) {
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

func (u *upload) UploadedChunkIndex() []uint32 {
	var arr []uint32
	lsRes, _, _ := u.client.Object.ListParts(context.Background(), u.upload.Key, u.upload.UploadID, nil)
	for _, item := range lsRes.Parts {
		arr = append(arr, uint32(item.PartNumber))
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
