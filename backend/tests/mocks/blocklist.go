package mocks

import (
	"context"
	"sync"
	"time"
)

// MemoryTokenBlacklist 是测试用内存 Token 黑名单。
type MemoryTokenBlacklist struct {
	mu     sync.RWMutex
	tokens map[string]time.Time
}

// NewMemoryTokenBlacklist 创建测试用 Token 黑名单。
func NewMemoryTokenBlacklist() *MemoryTokenBlacklist {
	return &MemoryTokenBlacklist{tokens: make(map[string]time.Time)}
}

// Add 将 Token 保存到黑名单直到过期。
func (b *MemoryTokenBlacklist) Add(_ context.Context, token string, expiresAt time.Time) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tokens[token] = expiresAt
	return nil
}

// Contains 判断 Token 当前是否在黑名单中。
func (b *MemoryTokenBlacklist) Contains(_ context.Context, token string) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	expiresAt, ok := b.tokens[token]
	if !ok {
		return false, nil
	}
	return time.Now().Before(expiresAt), nil
}
