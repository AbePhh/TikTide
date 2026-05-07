package oss

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/storage"

	"github.com/AbePhh/TikTide/backend/pkg/config"
)

type Client interface {
	GeneratePutSignedURL(ctx context.Context, objectKey, contentType string, expire time.Duration) (string, error)
	GenerateGetSignedURL(ctx context.Context, objectKey string, expire time.Duration) (string, error)
	ObjectExists(ctx context.Context, objectKey string) (bool, error)
	ObjectURL(objectKey string) string
	GetObject(ctx context.Context, objectKey string) (io.ReadCloser, error)
	PutObject(ctx context.Context, objectKey string, reader io.Reader) error
}

type QiniuClient struct {
	mac           *auth.Credentials
	bucketName    string
	bucketDomain  string
	formUploader  *storage.FormUploader
	bucketManager *storage.BucketManager
	accessKey     string
	secretKey     string
	region        string
}

func NewQiniuClient(cfg config.Config) (*QiniuClient, error) {
	if strings.TrimSpace(cfg.OSSBucket) == "" {
		return nil, fmt.Errorf("oss bucket is empty")
	}
	if strings.TrimSpace(cfg.OSSAccessKeyID) == "" || strings.TrimSpace(cfg.OSSAccessKeySecret) == "" {
		return nil, fmt.Errorf("oss access key is empty")
	}

	mac := auth.New(cfg.OSSAccessKeyID, cfg.OSSAccessKeySecret)
	storageConfig := &storage.Config{
		UseHTTPS: true,
		Zone:     newStorageZone(cfg.OSSRegion),
	}

	return &QiniuClient{
		mac:           mac,
		bucketName:    strings.TrimSpace(cfg.OSSBucket),
		bucketDomain:  normalizeBucketDomain(cfg.OSSEndpoint),
		formUploader:  storage.NewFormUploader(storageConfig),
		bucketManager: storage.NewBucketManager(mac, storageConfig),
		accessKey:     strings.TrimSpace(cfg.OSSAccessKeyID),
		secretKey:     strings.TrimSpace(cfg.OSSAccessKeySecret),
		region:        normalizeS3Region(cfg.OSSRegion),
	}, nil
}

func (c *QiniuClient) GeneratePutSignedURL(_ context.Context, objectKey, contentType string, expire time.Duration) (string, error) {
	policy := storage.PutPolicy{
		Scope:   c.bucketName + ":" + trimObjectKey(objectKey),
		Expires: uint64(expire.Seconds()),
	}
	if strings.TrimSpace(contentType) != "" {
		policy.MimeLimit = strings.TrimSpace(contentType)
	}
	return policy.UploadToken(c.mac), nil
}

func (c *QiniuClient) GenerateGetSignedURL(_ context.Context, objectKey string, expire time.Duration) (string, error) {
	signedURL, err := c.presignS3GetObjectURL(trimObjectKey(objectKey), expire)
	if err != nil {
		log.Printf("oss read sign failed: bucket=%s endpoint=%s region=%s object_key=%s err=%v", c.bucketName, c.bucketDomain, c.region, trimObjectKey(objectKey), err)
		return "", err
	}
	return signedURL, nil
}

func (c *QiniuClient) ObjectExists(_ context.Context, objectKey string) (bool, error) {
	_, err := c.bucketManager.Stat(c.bucketName, trimObjectKey(objectKey))
	if err == nil {
		return true, nil
	}
	if isQiniuNotFound(err) {
		return false, nil
	}
	return false, fmt.Errorf("check object existence: %w", err)
}

func (c *QiniuClient) ObjectURL(objectKey string) string {
	return strings.TrimRight(c.bucketDomain, "/") + "/" + trimObjectKey(objectKey)
}

func (c *QiniuClient) GetObject(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	signedURL, err := c.GenerateGetSignedURL(ctx, objectKey, 15*time.Minute)
	if err != nil {
		log.Printf("oss get object sign failed: bucket=%s endpoint=%s object_key=%s err=%v", c.bucketName, c.bucketDomain, trimObjectKey(objectKey), err)
		return nil, fmt.Errorf("generate read url: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, signedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create get request: %w", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Printf("oss get object http failed: bucket=%s endpoint=%s object_key=%s signed_url=%s err=%v", c.bucketName, c.bucketDomain, trimObjectKey(objectKey), signedURL, err)
		return nil, fmt.Errorf("get object: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		defer func() {
			_ = response.Body.Close()
		}()
		log.Printf("oss get object unexpected status: bucket=%s endpoint=%s object_key=%s signed_url=%s status=%d", c.bucketName, c.bucketDomain, trimObjectKey(objectKey), signedURL, response.StatusCode)
		return nil, fmt.Errorf("get object unexpected status: %d", response.StatusCode)
	}
	return response.Body, nil
}

func (c *QiniuClient) PutObject(ctx context.Context, objectKey string, reader io.Reader) error {
	uploadToken, err := c.GeneratePutSignedURL(ctx, objectKey, "", time.Hour)
	if err != nil {
		return fmt.Errorf("generate upload token: %w", err)
	}
	return c.formUploader.Put(ctx, nil, uploadToken, trimObjectKey(objectKey), reader, -1, &storage.PutExtra{})
}

func trimObjectKey(objectKey string) string {
	return strings.TrimPrefix(strings.TrimSpace(objectKey), "/")
}

func normalizeBucketDomain(endpoint string) string {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return strings.TrimRight(trimmed, "/")
	}
	return "https://" + strings.TrimRight(trimmed, "/")
}

func isQiniuNotFound(err error) bool {
	lowered := strings.ToLower(err.Error())
	return strings.Contains(lowered, "no such file or directory") ||
		strings.Contains(lowered, "status code = 612") ||
		strings.Contains(lowered, "file not exist") ||
		strings.Contains(lowered, "no such key")
}

func newStorageZone(rawRegion string) *storage.Zone {
	switch strings.ToLower(strings.TrimSpace(rawRegion)) {
	case "z0", "cn-east-1":
		return &storage.ZoneHuadong
	case "z1":
		return &storage.ZoneHuabei
	case "z2":
		return &storage.ZoneHuanan
	case "na0":
		return &storage.ZoneBeimei
	case "as0":
		return &storage.ZoneXinjiapo
	default:
		return &storage.ZoneHuadong
	}
}

func normalizeS3Region(rawRegion string) string {
	switch strings.ToLower(strings.TrimSpace(rawRegion)) {
	case "", "z0":
		return "cn-east-1"
	case "z1":
		return "cn-north-1"
	case "z2":
		return "cn-south-1"
	default:
		return strings.TrimSpace(rawRegion)
	}
}

func (c *QiniuClient) presignS3GetObjectURL(objectKey string, expire time.Duration) (string, error) {
	if strings.TrimSpace(objectKey) == "" {
		return "", fmt.Errorf("object key is empty")
	}

	endpointURL, err := url.Parse(c.bucketDomain)
	if err != nil {
		return "", fmt.Errorf("parse endpoint: %w", err)
	}
	if endpointURL.Scheme == "" || endpointURL.Host == "" {
		return "", fmt.Errorf("invalid endpoint: %s", c.bucketDomain)
	}

	expires := int64(expire.Seconds())
	if expires <= 0 {
		expires = 900
	}
	if expires > 604800 {
		expires = 604800
	}

	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	serviceName := "s3"
	credentialScope := dateStamp + "/" + c.region + "/" + serviceName + "/aws4_request"

	queryValues := map[string]string{
		"X-Amz-Algorithm":     "AWS4-HMAC-SHA256",
		"X-Amz-Credential":    c.accessKey + "/" + credentialScope,
		"X-Amz-Date":          amzDate,
		"X-Amz-Expires":       strconv.FormatInt(expires, 10),
		"X-Amz-SignedHeaders": "host",
	}

	canonicalURI := "/" + encodeS3ObjectKey(objectKey)
	canonicalQuery := canonicalizeQuery(queryValues)
	canonicalHeaders := "host:" + endpointURL.Host + "\n"
	signedHeaders := "host"
	payloadHash := "UNSIGNED-PAYLOAD"

	canonicalRequest := strings.Join([]string{
		http.MethodGet,
		canonicalURI,
		canonicalQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		sha256Hex(canonicalRequest),
	}, "\n")

	signingKey := buildAWS4SigningKey(c.secretKey, dateStamp, c.region, serviceName)
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))
	queryValues["X-Amz-Signature"] = signature

	return endpointURL.Scheme + "://" + endpointURL.Host + canonicalURI + "?" + canonicalizeQuery(queryValues), nil
}

func encodeS3ObjectKey(objectKey string) string {
	parts := strings.Split(objectKey, "/")
	escaped := make([]string, 0, len(parts))
	for _, part := range parts {
		escaped = append(escaped, url.PathEscape(part))
	}
	return strings.Join(escaped, "/")
}

func canonicalizeQuery(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, awsURLEncode(key)+"="+awsURLEncode(values[key]))
	}
	return strings.Join(pairs, "&")
}

func awsURLEncode(input string) string {
	escaped := url.QueryEscape(input)
	escaped = strings.ReplaceAll(escaped, "+", "%20")
	escaped = strings.ReplaceAll(escaped, "*", "%2A")
	escaped = strings.ReplaceAll(escaped, "%7E", "~")
	return escaped
}

func sha256Hex(input string) string {
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(data))
	return mac.Sum(nil)
}

func buildAWS4SigningKey(secretKey, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), dateStamp)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	return hmacSHA256(kService, "aws4_request")
}
