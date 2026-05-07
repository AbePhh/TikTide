package model

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrRelationNotFound 表示关注关系不存在。
	ErrRelationNotFound = errors.New("relation not found")
)

// Relation 表示一条关注关系记录。
type Relation struct {
	ID        int64
	UserID    int64
	FollowID  int64
	IsMutual  bool
	CreatedAt time.Time
}

// Repository 定义关注关系模块依赖的持久化能力。
type Repository interface {
	Create(ctx context.Context, userID, followID int64) (*Relation, error)
	Delete(ctx context.Context, userID, followID int64) error
	Get(ctx context.Context, userID, followID int64) (*Relation, error)
	ListFollowing(ctx context.Context, userID, cursor int64, limit int) ([]Relation, error)
	ListFollowers(ctx context.Context, userID, cursor int64, limit int) ([]Relation, error)
	ListFollowersAll(ctx context.Context, userID int64) ([]Relation, error)
	ListFollowingAll(ctx context.Context, userID int64) ([]Relation, error)
}
