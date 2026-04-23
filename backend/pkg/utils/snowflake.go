package utils

import "github.com/bwmarrin/snowflake"

// IDGenerator 定义统一的 int64 ID 生成能力。
type IDGenerator interface {
	NewID() int64
}

// SnowflakeGenerator 对雪花算法节点做了一层封装。
type SnowflakeGenerator struct {
	node *snowflake.Node
}

// NewSnowflakeGenerator 创建雪花算法 ID 生成器。
func NewSnowflakeGenerator(nodeID int64) (*SnowflakeGenerator, error) {
	node, err := snowflake.NewNode(nodeID)
	if err != nil {
		return nil, err
	}
	return &SnowflakeGenerator{node: node}, nil
}

// NewID 生成一个新的业务 ID。
func (g *SnowflakeGenerator) NewID() int64 {
	return g.node.Generate().Int64()
}
