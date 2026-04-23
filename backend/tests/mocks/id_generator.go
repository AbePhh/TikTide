package mocks

import "sync/atomic"

// IncrementalIDGenerator 是测试用的确定性自增 ID 生成器。
type IncrementalIDGenerator struct {
	next atomic.Int64
}

// NewIncrementalIDGenerator 创建新的确定性 ID 生成器。
func NewIncrementalIDGenerator(start int64) *IncrementalIDGenerator {
	generator := &IncrementalIDGenerator{}
	generator.next.Store(start)
	return generator
}

// NewID 返回下一个自增 ID。
func (g *IncrementalIDGenerator) NewID() int64 {
	return g.next.Add(1)
}
