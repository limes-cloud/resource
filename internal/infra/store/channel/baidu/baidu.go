package baidu

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/baidubce/bce-sdk-go/bce"
	"github.com/baidubce/bce-sdk-go/services/bos"
	"github.com/baidubce/bce-sdk-go/services/bos/api"
	"github.com/go-redis/redis/v8"
	"github.com/limes-cloud/kratosx/pkg/lock"
	"github.com/tencentyun/cos-go-sdk-v5"

	"github.com/limes-cloud/resource/internal/infra/store/config"
	"github.com/limes-cloud/resource/internal/infra/store/types"
)

type Baidu struct {
	bucket    string
	keyword   string
	client    *bos.Client
	expire    time.Duration
	cache     *redis.Client
	cdn       string
	antiTheft bool
}

type upload struct {
	key    string
	bucket string
	upload *api.InitiateMultipartUploadResult
	client *bos.Client
}

func New(conf *config.Config) (*Baidu, error) {
	if conf.Endpoint == "" || conf.Secret == "" || conf.Id == "" {
		return nil, errors.New("store config error")
	}

	client, err := bos.NewClientWithConfig(&bos.BosClientConfiguration{
		Ak:               conf.Id,
		Sk:               conf.Secret,
		Endpoint:         conf.ServerURL,
		RedirectDisabled: false,
	})
	if err != nil {
		return nil, err
	}

	return &Baidu{
		keyword:   conf.Keyword,
		bucket:    conf.Bucket,
		client:    client,
		expire:    conf.TemporaryExpire,
		cache:     conf.Cache,
		cdn:       conf.Endpoint,
		antiTheft: conf.AntiTheft,
	}, nil
}

func (s *Baidu) GetKeyword() string {
	return s.keyword
}

func (s *Baidu) GenTemporaryURL(key string) (string, error) {
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
			st := s.client.Config.Credentials.SecretAccessKey + t + "/" + key
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

func (s *Baidu) VerifyTemporaryURL(key string, expire string, sign string) error {
	return nil
}

func (s *Baidu) PutBytes(key string, in []byte) error {
	return s.Put(key, bytes.NewReader(in))
}

func (s *Baidu) Put(key string, r io.Reader) error {
	response, err := s.client.PutObjectFromStream(s.bucket, key, r, nil)
	if err != nil {
		return err
	}

	// todo
	fmt.Println(response)

	return nil
}

func (s *Baidu) PutFromLocal(key string, localPath string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}

	return s.Put(key, file)
}

func (s *Baidu) Get(key string) (io.ReadCloser, error) {
	resp, err := s.client.GetObject(s.bucket, key, nil)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (s *Baidu) Delete(key string) error {
	return s.client.DeleteObject(s.bucket, key)
}

func (s *Baidu) Size(key string) (int64, error) {
	resp, err := s.client.BasicGetObject(s.bucket, key)
	if err != nil {
		return 0, err
	}

	return resp.ContentLength, nil
}

func (s *Baidu) Exists(key string) (bool, error) {
	_, err := s.client.GetObjectMeta(s.bucket, key)
	if realErr, ok := err.(*bce.BceServiceError); ok {
		if realErr.StatusCode == 404 {
			return false, nil
		}
	}
	return true, nil
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

func (s *Baidu) NewPutChunk(key string) (types.PutChunk, error) {
	up, err := s.client.BasicInitiateMultipartUpload(s.bucket, key)
	if err != nil {
		return nil, err
	}
	return &upload{
		key:    key,
		upload: up,
		client: s.client,
		bucket: s.bucket,
	}, nil
}

func (s *Baidu) NewPutChunkByUploadID(key, id string) (types.PutChunk, error) {
	up := &api.InitiateMultipartUploadResult{
		UploadId: id,
		Bucket:   s.bucket,
		Key:      key,
	}
	return &upload{
		key:    key,
		upload: up,
		client: s.client,
		bucket: s.bucket,
	}, nil
}

func (u *upload) UploadedChunkIndex() []uint32 {
	var arr []uint32
	lsRes, _ := u.client.ListParts(u.bucket, u.upload.Key, u.UploadID(), nil)
	for _, item := range lsRes.Parts {
		arr = append(arr, uint32(item.PartNumber))
	}
	return arr
}

func (u *upload) ChunkCount() int {
	lsRes, _ := u.client.ListParts(u.bucket, u.upload.Key, u.UploadID(), nil)
	return len(lsRes.Parts)
}

func (u *upload) UploadID() string {
	return u.upload.UploadId
}

func (u *upload) Append(r io.Reader, index int) error {
	body, _ := bce.NewBodyFromSizedReader(r, -1)
	_, err := u.client.BasicUploadPart(u.bucket, u.upload.Key, u.upload.UploadId, index, body)
	return err
}

func (u *upload) AppendBytes(r []byte, index int) error {
	_, err := u.client.UploadPartFromBytes(u.bucket, u.upload.Key, u.upload.UploadId, index, r, nil)
	return err
}

func (u *upload) Abort() error {
	return u.client.AbortMultipartUpload(u.bucket, u.upload.Key, u.upload.UploadId)
}

func (u *upload) Complete() error {
	lsRes, err := u.client.ListParts(u.bucket, u.upload.Key, u.UploadID(), nil)
	if err != nil {
		return err
	}

	opt := &api.CompleteMultipartUploadArgs{}
	for _, item := range lsRes.Parts {
		opt.Parts = append(opt.Parts, api.UploadInfoType{
			PartNumber: item.PartNumber,
			ETag:       item.ETag,
		})
	}

	_, err = u.client.CompleteMultipartUploadFromStruct(u.bucket, u.upload.Key, u.UploadID(), opt)
	return err
}
