package model

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrVideoNotFound 表示视频不存在。
	ErrVideoNotFound = errors.New("video not found")
)

const (
	// TranscodePending 表示待转码。
	TranscodePending int8 = 0
	// TranscodeProcessing 表示转码中。
	TranscodeProcessing int8 = 1
	// TranscodeSuccess 表示转码成功。
	TranscodeSuccess int8 = 2
	// TranscodeFailed 表示转码失败。
	TranscodeFailed int8 = 3
)

const (
	// VisibilityPrivate 表示仅自己可见。
	VisibilityPrivate int8 = 0
	// VisibilityPublic 表示公开可见。
	VisibilityPublic int8 = 1
)

const (
	// AuditPending 表示待审核。
	AuditPending int8 = 0
	// AuditPassed 表示审核通过。
	AuditPassed int8 = 1
	// AuditRejected 表示审核驳回。
	AuditRejected int8 = 2
)

// Video 对应 t_video 表。
type Video struct {
	ID                  int64
	UserID              int64
	ObjectKey           string
	SourceURL           string
	Title               string
	CoverURL            string
	DurationMS          int32
	AllowComment        int8
	Visibility          int8
	TranscodeStatus     int8
	AuditStatus         int8
	TranscodeFailReason string
	AuditRemark         string
	PlayCount           int64
	LikeCount           int64
	CommentCount        int64
	FavoriteCount       int64
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// Hashtag 对应 t_hashtag 表。
type Hashtag struct {
	ID        int64
	Name      string
	UseCount  int64
	CreatedAt time.Time
}

// Repository 定义视频模块所需的持久化操作。
type Repository interface {
	CreateVideo(ctx context.Context, video *Video, hashtagIDs []int64) error
	CountHashtagsByIDs(ctx context.Context, hashtagIDs []int64) (int64, error)
}
