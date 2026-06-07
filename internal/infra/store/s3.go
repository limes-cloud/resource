package store

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/limes-cloud/kratosx/library/lock"
	"github.com/redis/go-redis/v9"

	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/repository"
	"github.com/limes-cloud/resource/internal/types"
)

type S3 struct {
	conf      *core.Storage
	client    *awss3.Client
	bucket    string
	expire    time.Duration
	cdn       string
	antiTheft bool
	cache     *redis.Client
	ctx       context.Context
}

type s3Upload struct {
	client   *awss3.Client
	bucket   string
	key      string
	uploadID string
}

func newS3(ctx core.Context, conf *core.Storage) (*S3, error) {
	if conf.Endpoint == "" || conf.AK == "" || conf.Secret == "" {
		return nil, errors.New("store config error")
	}
	client := awss3.New(awss3.Options{
		BaseEndpoint: aws.String(conf.Endpoint),
		Region:       conf.Region,
		Credentials:  credentials.NewStaticCredentialsProvider(conf.AK, conf.Secret, ""),
		UsePathStyle: true,
	})
	return &S3{
		conf:      conf,
		client:    client,
		bucket:    conf.Bucket,
		expire:    conf.TemporaryExpire,
		cdn:       conf.ServerURL,
		antiTheft: conf.AntiTheft,
		cache:     ctx.Redis(),
		ctx:       context.Background(),
	}, nil
}

func (s *S3) Config() *core.Storage { return s.conf }

func (s *S3) ParserQuery(_ *types.ParserQuery) string { return "" }

func (s *S3) GenTemporaryURL(key string) (string, error) {
	if !s.antiTheft {
		return s.cdn + "/" + key, nil
	}
	var (
		err    error
		target string
		locker = lock.New(s.ctx, key+":lock")
	)
	ck := fmt.Sprintf("resource:%x", md5.Sum([]byte(key)))
	err = locker.AcquireFunc(
		func() error {
			target, err = s.cache.Get(context.Background(), ck).Result()
			return err
		},
		func() error {
			t := time.Now().Add(s.expire).Format("200601021504")
			st := s.conf.Secret + t + "/" + key
			target = fmt.Sprintf("%s/%s/%s/%s", s.cdn, t, fmt.Sprintf("%x", md5.Sum([]byte(st))), key)
			return s.cache.Set(context.Background(), ck, target, s.expire-10*time.Second).Err()
		},
	)
	return target, err
}

func (s *S3) VerifyTemporaryURL(_, _, _ string) error { return nil }

func (s *S3) PutBytes(key string, in []byte) error { return s.Put(key, bytes.NewReader(in)) }

func (s *S3) Put(key string, r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	_, err = s.client.PutObject(s.ctx, &awss3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	return err
}

func (s *S3) PutFromLocal(key string, localPath string) error {
	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return s.Put(key, f)
}

func (s *S3) Get(key string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(s.ctx, &awss3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

func (s *S3) Delete(key string) error {
	_, err := s.client.DeleteObject(s.ctx, &awss3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

func (s *S3) Size(key string) (int64, error) {
	out, err := s.client.HeadObject(s.ctx, &awss3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return 0, err
	}
	if out.ContentLength == nil {
		return 0, nil
	}
	return *out.ContentLength, nil
}

func (s *S3) Exists(key string) (bool, error) {
	_, err := s.client.HeadObject(s.ctx, &awss3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err == nil, nil
}

func (s *S3) NewPutChunk(key string) (repository.PutChunk, error) {
	out, err := s.client.CreateMultipartUpload(s.ctx, &awss3.CreateMultipartUploadInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return &s3Upload{client: s.client, bucket: s.bucket, key: key, uploadID: *out.UploadId}, nil
}

func (s *S3) NewPutChunkByUploadID(key, id string) (repository.PutChunk, error) {
	return &s3Upload{client: s.client, bucket: s.bucket, key: key, uploadID: id}, nil
}

func (u *s3Upload) UploadID() string { return u.uploadID }

func (u *s3Upload) UploadedChunkIndex() []uint32 {
	var arr []uint32
	out, err := u.client.ListParts(context.Background(), &awss3.ListPartsInput{
		Bucket:   aws.String(u.bucket),
		Key:      aws.String(u.key),
		UploadId: aws.String(u.uploadID),
	})
	if err != nil {
		return arr
	}
	for _, p := range out.Parts {
		if p.PartNumber != nil {
			arr = append(arr, uint32(*p.PartNumber))
		}
	}
	return arr
}

func (u *s3Upload) ChunkCount() int { return len(u.UploadedChunkIndex()) }

func (u *s3Upload) AppendBytes(in []byte, index int) error {
	partNum := int32(index)
	_, err := u.client.UploadPart(context.Background(), &awss3.UploadPartInput{
		Bucket:     aws.String(u.bucket),
		Key:        aws.String(u.key),
		UploadId:   aws.String(u.uploadID),
		PartNumber: &partNum,
		Body:       bytes.NewReader(in),
	})
	return err
}

func (u *s3Upload) Append(r io.Reader, index int) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return u.AppendBytes(data, index)
}

func (u *s3Upload) Abort() error {
	_, err := u.client.AbortMultipartUpload(context.Background(), &awss3.AbortMultipartUploadInput{
		Bucket:   aws.String(u.bucket),
		Key:      aws.String(u.key),
		UploadId: aws.String(u.uploadID),
	})
	return err
}

func (u *s3Upload) Complete() error {
	out, err := u.client.ListParts(context.Background(), &awss3.ListPartsInput{
		Bucket:   aws.String(u.bucket),
		Key:      aws.String(u.key),
		UploadId: aws.String(u.uploadID),
	})
	if err != nil {
		return err
	}
	var parts []s3types.CompletedPart
	for _, p := range out.Parts {
		parts = append(parts, s3types.CompletedPart{PartNumber: p.PartNumber, ETag: p.ETag})
	}
	_, err = u.client.CompleteMultipartUpload(context.Background(), &awss3.CompleteMultipartUploadInput{
		Bucket:          aws.String(u.bucket),
		Key:             aws.String(u.key),
		UploadId:        aws.String(u.uploadID),
		MultipartUpload: &s3types.CompletedMultipartUpload{Parts: parts},
	})
	return err
}
