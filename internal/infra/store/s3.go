package store

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	apierrors "github.com/limes-cloud/resource/api/errors"
	"io"
	"sort"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"github.com/limes-cloud/resource/internal/core"
	"github.com/limes-cloud/resource/internal/domain/entity"
	"github.com/limes-cloud/resource/internal/domain/repository"
	"github.com/limes-cloud/resource/internal/pkg/provider"
)

// cdnTypeATimeLayout CDN Type-A 的时间戳格式（分钟粒度）
const cdnTypeATimeLayout = "200601021504"

// ErrStoreConfigIncomplete store 行缺少构造客户端所需的字段
var ErrStoreConfigIncomplete = errors.New("store: 配置不完整")

// 连通性探测的分类错误（Check 返回，便于上层生成用户可读原因）
var (
	// ErrCheckAuthFailed 凭证或权限错误（403 系：密钥错 / 无权限 / 区域不匹配）
	ErrCheckAuthFailed = errors.New("store: 凭证或权限错误")
	// ErrCheckBucketNotFound 桶不存在（404）
	ErrCheckBucketNotFound = errors.New("store: 桶不存在")
)

// s3 S3 兼容厂商客户端（aws / bos / minio 等）。
type s3 struct {
	keyword string
	client  *awss3.Client
	bucket  string
	baseURL string // 公网访问地址前缀（读路径），已归一化去尾斜杠
	cdnAuth string // 读路径鉴权方式
	cdnKey  string // CDN Type-A 鉴权 key
}

// NewClient 由 store 行构造客户端。
func NewClient(ent *entity.Store) (repository.Client, error) {
	if ent.Endpoint == "" || ent.Bucket == "" || ent.AK == "" || ent.SK == "" || ent.Region == "" {
		return nil, ErrStoreConfigIncomplete
	}

	// region 归一化（容错 oss- / s3- 等端点前缀，如 oss-cn-beijing → cn-beijing；签名区域必须与 endpoint 一致）
	region := provider.NormalizeRegion(ent.Provider, ent.Region)

	// 寻址方式只由 (provider, region) 决定；provider 不在白名单或行为表未实现即报错，禁止静默默认
	style, err := provider.AddressingStyle(ent.Provider, region)
	if err != nil {
		return nil, err
	}

	client := awss3.New(awss3.Options{
		BaseEndpoint:               aws.String("https://" + ent.Endpoint),
		Region:                     region,
		Credentials:                credentials.NewStaticCredentialsProvider(ent.AK, ent.SK, ""),
		UsePathStyle:               style == provider.AddressingPath,
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	})

	inst := &s3{
		keyword: ent.Keyword,
		client:  client,
		bucket:  ent.Bucket,
		baseURL: provider.CanonicalBaseURL(ent.PublicBaseUrl),
		cdnAuth: ent.CdnAuthType,
	}
	if ent.CdnAuthKey != nil {
		inst.cdnKey = *ent.CdnAuthKey
	}
	return inst, nil
}

// Keyword 返回该客户端对应的 store.keyword
func (s *s3) Keyword() string { return s.keyword }

// Put 覆盖写入一份字节
func (s *s3) Put(ctx core.Context, in *repository.PutInput) error {
	_, err := s.client.PutObject(ctx.Ctx(), &awss3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(in.Key),
		Body:          in.Body,
		ContentLength: aws.Int64(in.Size),
		ContentType:   aws.String(in.ContentType),
	})
	return err
}

// Get 读取一份字节
func (s *s3) Get(ctx core.Context, key string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx.Ctx(), &awss3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isObjectNotFound(err) {
			return nil, apierrors.ObjectNotFoundError()
		}
		return nil, err
	}
	return out.Body, nil
}

// Head 取对象元信息；对象不存在时返回 ErrObjectNotFound
func (s *s3) Head(ctx core.Context, key string) (*repository.ObjectInfo, error) {
	out, err := s.client.HeadObject(ctx.Ctx(), &awss3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isObjectNotFound(err) {
			return nil, apierrors.ObjectNotFoundError()
		}
		return nil, err
	}
	return &repository.ObjectInfo{
		Size:        uint64(aws.ToInt64(out.ContentLength)),
		ContentType: aws.ToString(out.ContentType),
		ETag:        aws.ToString(out.ETag),
	}, nil
}

// Check 连通性自检：HeadBucket 探测（200 = 认证/桶/网络全通；只读探测，不写任何数据）。
// 错误按类别映射为 ErrCheckAuthFailed / ErrCheckBucketNotFound，网络等其它错误原样返回。
func (s *s3) Check(ctx core.Context) error {
	_, err := s.client.HeadBucket(ctx.Ctx(), &awss3.HeadBucketInput{Bucket: aws.String(s.bucket)})
	if err == nil {
		return nil
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NotFound", "NoSuchBucket":
			return ErrCheckBucketNotFound
		case "Forbidden", "AccessDenied", "InvalidAccessKeyId", "SignatureDoesNotMatch", "InvalidToken":
			return ErrCheckAuthFailed
		}
	}
	return err
}

// Delete 删除对象；对象不存在也返回 nil（S3 系 DELETE 幂等）
func (s *s3) Delete(ctx core.Context, key string) error {
	_, err := s.client.DeleteObject(ctx.Ctx(), &awss3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil && isObjectNotFound(err) {
		return nil
	}
	return err
}

// DeleteBatch 批量删除对象（S3 DeleteObjects，按 1000 一批）；对象不存在不报错（S3 幂等）。
// 返回体里的 per-key 失败会汇总成 error 交上层记录（行已先删，无法重试，仅告警可见）。
func (s *s3) DeleteBatch(ctx core.Context, keys []string) error {
	const maxPerCall = 1000
	var failed int
	var firstKey, firstMsg string
	for start := 0; start < len(keys); start += maxPerCall {
		end := start + maxPerCall
		if end > len(keys) {
			end = len(keys)
		}
		ids := make([]s3types.ObjectIdentifier, 0, end-start)
		for _, k := range keys[start:end] {
			ids = append(ids, s3types.ObjectIdentifier{Key: aws.String(k)})
		}
		out, err := s.client.DeleteObjects(ctx.Ctx(), &awss3.DeleteObjectsInput{
			Bucket: aws.String(s.bucket),
			Delete: &s3types.Delete{Objects: ids, Quiet: aws.Bool(true)},
		})
		if err != nil {
			return err
		}
		for _, de := range out.Errors {
			failed++
			if firstKey == "" {
				firstKey = aws.ToString(de.Key)
				firstMsg = aws.ToString(de.Message)
			}
		}
	}
	if failed > 0 {
		return fmt.Errorf("批量删桶 %d 个失败，首个 key=%s msg=%s", failed, firstKey, firstMsg)
	}
	return nil
}

// NewMultipart 发起分片上传并返回厂商 uploadId（对象元数据只能在此刻写死）。
func (s *s3) NewMultipart(ctx core.Context, key, contentType string) (string, error) {
	out, err := s.client.CreateMultipartUpload(ctx.Ctx(), &awss3.CreateMultipartUploadInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(out.UploadId), nil
}

// ListParts 列出已传分片（全量分页、按片号升序）。
func (s *s3) ListParts(ctx core.Context, key, uploadID string) ([]repository.UploadedPart, error) {
	var (
		parts  []repository.UploadedPart
		marker *string
	)
	for {
		out, err := s.client.ListParts(ctx.Ctx(), &awss3.ListPartsInput{
			Bucket:           aws.String(s.bucket),
			Key:              aws.String(key),
			UploadId:         aws.String(uploadID),
			PartNumberMarker: marker,
		})
		if err != nil {
			if isNoSuchUpload(err) {
				return nil, apierrors.UploadSessionNotFoundError()
			}
			return nil, err
		}
		for _, p := range out.Parts {
			parts = append(parts, repository.UploadedPart{
				Number: uint32(aws.ToInt32(p.PartNumber)),
				Size:   uint64(aws.ToInt64(p.Size)),
				ETag:   aws.ToString(p.ETag),
			})
		}
		if !aws.ToBool(out.IsTruncated) || out.NextPartNumberMarker == nil {
			break
		}
		marker = out.NextPartNumberMarker
	}

	sort.Slice(parts, func(i, j int) bool { return parts[i].Number < parts[j].Number })
	return parts, nil
}

// Complete 合并分片；会话已在厂商侧结束时返回 ErrNoSuchUpload
func (s *s3) Complete(ctx core.Context, key, uploadID string, parts []repository.CompletedPart) error {
	completed := make([]s3types.CompletedPart, 0, len(parts))
	for _, p := range parts {
		completed = append(completed, s3types.CompletedPart{
			PartNumber: aws.Int32(int32(p.Number)),
			ETag:       aws.String(p.ETag),
		})
	}

	_, err := s.client.CompleteMultipartUpload(ctx.Ctx(), &awss3.CompleteMultipartUploadInput{
		Bucket:          aws.String(s.bucket),
		Key:             aws.String(key),
		UploadId:        aws.String(uploadID),
		MultipartUpload: &s3types.CompletedMultipartUpload{Parts: completed},
	})
	if err != nil {
		if isNoSuchUpload(err) {
			return apierrors.UploadSessionNotFoundError()
		}
		return err
	}
	return nil
}

// Abort 释放分片；会话已在厂商侧结束时返回 ErrNoSuchUpload
func (s *s3) Abort(ctx core.Context, key, uploadID string) error {
	_, err := s.client.AbortMultipartUpload(ctx.Ctx(), &awss3.AbortMultipartUploadInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	if err != nil {
		if isNoSuchUpload(err) {
			return apierrors.UploadSessionNotFoundError()
		}
		return err
	}
	return nil
}

// PresignPart 生成分片上传的预签名地址。
func (s *s3) PresignPart(ctx core.Context, in *repository.PresignPartInput) (string, error) {
	presigner := awss3.NewPresignClient(s.client)
	req, err := presigner.PresignUploadPart(ctx.Ctx(), &awss3.UploadPartInput{
		Bucket:     aws.String(s.bucket),
		Key:        aws.String(in.Key),
		UploadId:   aws.String(in.UploadID),
		PartNumber: aws.Int32(int32(in.PartNumber)),
	}, func(o *awss3.PresignOptions) { o.Expires = in.Ttl })
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

// GenTemporaryURL 生成读侧临时地址，按 store.cdn_auth_type 分档。
func (s *s3) GenTemporaryURL(ctx core.Context, key string, ttl time.Duration) (string, error) {
	switch s.cdnAuth {
	case entity.CdnAuthS3Presign:
		// 用厂商 endpoint 预签名：SigV4 的 host 参与签名，换成 CDN 域名会整域 403
		presigner := awss3.NewPresignClient(s.client)
		req, err := presigner.PresignGetObject(ctx.Ctx(), &awss3.GetObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key),
		}, func(o *awss3.PresignOptions) { o.Expires = ttl })
		if err != nil {
			return "", err
		}
		return req.URL, nil

	case entity.CdnAuthTypeA:
		// CDN Type-A：{base}/{过期时间}/{md5(key+过期时间+"/"+对象名)}/{对象名}
		expire := time.Now().Add(ttl).Format(cdnTypeATimeLayout)
		sign := md5Hex(s.cdnKey + expire + "/" + key)
		return s.baseURL + "/" + expire + "/" + sign + "/" + key, nil

	case entity.CdnAuthTypeB:
		// CDN Type-B（query 鉴权）：{base}/{对象名}?auth_key={ts}-0-0-{md5("/对象名-ts-0-0-key")}
		ts := strconv.FormatInt(time.Now().Add(ttl).Unix(), 10)
		sign := md5Hex("/" + key + "-" + ts + "-0-0-" + s.cdnKey)
		return s.baseURL + "/" + key + "?auth_key=" + ts + "-0-0-" + sign, nil

	case entity.CdnAuthTypeC:
		// CDN Type-C：{base}/{md5("key+/对象名+ts")}/{hex(ts)}/{对象名}
		now := time.Now().Add(ttl).Unix()
		sign := md5Hex(s.cdnKey + "/" + key + strconv.FormatInt(now, 10))
		return s.baseURL + "/" + sign + "/" + strconv.FormatInt(now, 16) + "/" + key, nil

	default:
		// none：公有读，直接拼 URL
		return s.baseURL + "/" + key, nil
	}
}

// md5Hex 计算字符串的 MD5 小写 hex
func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

// isNoSuchUpload 判断厂商错误是否为"会话已在厂商侧结束"
func isNoSuchUpload(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "NoSuchUpload"
	}
	return false
}

// isObjectNotFound 判断厂商错误是否为"对象 / 桶不存在"
func isObjectNotFound(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NoSuchKey", "NoSuchBucket", "NotFound":
			return true
		}
	}
	return false
}

// 编译期断言：*s3 实现 repository.Client
var _ repository.Client = (*s3)(nil)
