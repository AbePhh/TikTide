package mocks

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryOSSClient 是测试用内存 OSS 客户端。
type MemoryOSSClient struct {
	mu         sync.RWMutex
	existing   map[string]struct{}
	lastSigned string
}

// NewMemoryOSSClient 创建新的测试用 OSS 客户端。
func NewMemoryOSSClient(existingObjectKeys ...string) *MemoryOSSClient {
	store := make(map[string]struct{}, len(existingObjectKeys))
	for _, key := range existingObjectKeys {
		store[key] = struct{}{}
	}
	return &MemoryOSSClient{existing: store}
}

// GeneratePutSignedURL 返回假的签名上传地址。
func (c *MemoryOSSClient) GeneratePutSignedURL(_ context.Context, objectKey string, _ time.Duration) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastSigned = objectKey
	return fmt.Sprintf("https://example.com/upload/%s", objectKey), nil
}

// ObjectExists 判断对象是否存在。
func (c *MemoryOSSClient) ObjectExists(_ context.Context, objectKey string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.existing[objectKey]
	return ok, nil
}

// ObjectURL 返回对象地址。
func (c *MemoryOSSClient) ObjectURL(objectKey string) string {
	return fmt.Sprintf("https://example.com/object/%s", objectKey)
}

// AddObject 向测试客户端添加对象。
func (c *MemoryOSSClient) AddObject(objectKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.existing[objectKey] = struct{}{}
}
