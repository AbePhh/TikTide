package oss

import (
	"context"
	"fmt"
	"strings"
	"time"

	alioss "github.com/aliyun/aliyun-oss-go-sdk/oss"

	"github.com/AbePhh/TikTide/backend/pkg/config"
)

// Client 定义 OSS 直传和对象探测能力。
type Client interface {
	GeneratePutSignedURL(ctx context.Context, objectKey string, expire time.Duration) (string, error)
	ObjectExists(ctx context.Context, objectKey string) (bool, error)
	ObjectURL(objectKey string) string
}

// AliyunClient 基于阿里云 OSS SDK 实现。
type AliyunClient struct {
	bucket     *alioss.Bucket
	endpoint   string
	bucketName string
}

// NewAliyunClient 创建阿里云 OSS 客户端。
func NewAliyunClient(cfg config.Config) (*AliyunClient, error) {
	if strings.TrimSpace(cfg.OSSEndpoint) == "" || strings.TrimSpace(cfg.OSSBucket) == "" {
		return nil, fmt.Errorf("oss endpoint or bucket is empty")
	}
	if strings.TrimSpace(cfg.OSSAccessKeyID) == "" || strings.TrimSpace(cfg.OSSAccessKeySecret) == "" {
		return nil, fmt.Errorf("oss access key is empty")
	}

	endpoint := cfg.OSSEndpoint
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}

	client, err := alioss.New(endpoint, cfg.OSSAccessKeyID, cfg.OSSAccessKeySecret, alioss.Region(cfg.OSSRegion))
	if err != nil {
		return nil, fmt.Errorf("create aliyun oss client: %w", err)
	}

	bucket, err := client.Bucket(cfg.OSSBucket)
	if err != nil {
		return nil, fmt.Errorf("get bucket: %w", err)
	}

	return &AliyunClient{
		bucket:     bucket,
		endpoint:   strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://"),
		bucketName: cfg.OSSBucket,
	}, nil
}

// GeneratePutSignedURL 生成 PUT 上传签名地址。
func (c *AliyunClient) GeneratePutSignedURL(_ context.Context, objectKey string, expire time.Duration) (string, error) {
	url, err := c.bucket.SignURL(objectKey, alioss.HTTPPut, int64(expire.Seconds()))
	if err != nil {
		return "", fmt.Errorf("sign upload url: %w", err)
	}
	return url, nil
}

// ObjectExists 判断对象是否存在。
func (c *AliyunClient) ObjectExists(_ context.Context, objectKey string) (bool, error) {
	exists, err := c.bucket.IsObjectExist(objectKey)
	if err != nil {
		return false, fmt.Errorf("check object existence: %w", err)
	}
	return exists, nil
}

// ObjectURL 返回对象公开地址格式。
func (c *AliyunClient) ObjectURL(objectKey string) string {
	return fmt.Sprintf("https://%s.%s/%s", c.bucketName, c.endpoint, strings.TrimPrefix(objectKey, "/"))
}
