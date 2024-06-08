package aliyun

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/go-redis/redis/v8"
	"github.com/limes-cloud/kratosx/pkg/lock"

	"github.com/limes-cloud/resource/internal/pkg/store"
)

type aliyun struct {
	bucket *oss.Bucket
	expire time.Duration
	cache  *redis.Client
	cdn    string
}

type upload struct {
	bucket *oss.Bucket
	upload oss.InitiateMultipartUploadResult
}

func New(conf *store.Config) (store.Store, error) {
	if conf.Endpoint == "" || conf.Id == "" || conf.Secret == "" {
		return nil, errors.New("store config error")
	}

	client, err := oss.New(conf.Endpoint, conf.Id, conf.Secret)
	if err != nil {
		return nil, err
	}

	bucket, err := client.Bucket(conf.Bucket)
	if err != nil {
		return nil, err
	}

	return &aliyun{
		bucket: bucket,
		expire: conf.TemporaryExpire,
		cache:  conf.Cache,
		cdn:    conf.ServerURL,
	}, nil
}

func (s *aliyun) GenTemporaryURL(key string) (string, error) {
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
			st := s.bucket.Client.Config.AccessKeySecret + t + "/" + key
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

func (s *aliyun) VerifyTemporaryURL(key string, expire string, sign string) error {
	return nil
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

func (u *upload) UploadedChunkIndex() []uint32 {
	var arr []uint32
	lsRes, _ := u.bucket.ListUploadedParts(u.upload)
	for _, item := range lsRes.UploadedParts {
		arr = append(arr, uint32(item.PartNumber))
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
