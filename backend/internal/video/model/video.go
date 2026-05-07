package model

import (
	"context"
	"errors"
	"time"
)

var (
	ErrVideoNotFound         = errors.New("video not found")
	ErrInvalidTranscodeState = errors.New("invalid transcode state")
	ErrHashtagNotFound       = errors.New("hashtag not found")
	ErrDraftNotFound         = errors.New("draft not found")
)

const (
	TranscodePending    int8 = 0
	TranscodeProcessing int8 = 1
	TranscodeSuccess    int8 = 2
	TranscodeFailed     int8 = 3
)

const (
	VisibilityPrivate int8 = 0
	VisibilityPublic  int8 = 1
)

const (
	AuditPending  int8 = 0
	AuditPassed   int8 = 1
	AuditRejected int8 = 2
)

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

type VideoResource struct {
	ID         int64
	VideoID    int64
	Resolution string
	FileURL    string
	FileSize   int64
	Bitrate    int32
	CreatedAt  time.Time
}

type Draft struct {
	ID           int64
	UserID       int64
	ObjectKey    string
	CoverURL     string
	Title        string
	TagNames     string
	AllowComment int8
	Visibility   int8
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Hashtag struct {
	ID        int64
	Name      string
	UseCount  int64
	CreatedAt time.Time
}

type HashtagVideo struct {
	VideoID         int64
	UserID          int64
	Title           string
	ObjectKey       string
	SourceURL       string
	CoverURL        string
	Visibility      int8
	TranscodeStatus int8
	AuditStatus     int8
	CreatedAt       time.Time
}

type VideoTag struct {
	VideoID int64
	Name    string
}

type VideoSearchDocumentSource struct {
	Video    Video
	AuthorID int64
}

type Repository interface {
	CreateVideo(ctx context.Context, video *Video, hashtagIDs []int64) error
	GetVideoByID(ctx context.Context, videoID int64) (*Video, error)
	ListVideosByIDs(ctx context.Context, videoIDs []int64) ([]Video, error)
	ListVideosByUser(ctx context.Context, userID int64, cursor *time.Time, limit int, includeInvisible bool) ([]Video, error)
	ListRecommendVideos(ctx context.Context, limit int) ([]Video, error)
	ListVideosForSearch(ctx context.Context, limit, offset int) ([]Video, error)
	ListHashtagNamesByVideoIDs(ctx context.Context, videoIDs []int64) (map[int64][]string, error)
	ListVideoResources(ctx context.Context, videoID int64) ([]VideoResource, error)
	IncreasePlayCount(ctx context.Context, videoID int64) error
	MarkVideoTranscoding(ctx context.Context, videoID int64) error
	MarkVideoTranscodeSuccess(ctx context.Context, videoID int64, coverURL string, durationMS int32, resources []VideoResource) error
	MarkVideoTranscodeFailed(ctx context.Context, videoID int64, failReason string) error
	CountHashtagsByIDs(ctx context.Context, hashtagIDs []int64) (int64, error)
	GetHashtagByID(ctx context.Context, hashtagID int64) (*Hashtag, error)
	CreateHashtag(ctx context.Context, name string) (*Hashtag, error)
	ListHotHashtags(ctx context.Context, limit int) ([]Hashtag, error)
	ListHashtags(ctx context.Context, limit, offset int) ([]Hashtag, error)
	ListVideosByHashtag(ctx context.Context, hashtagID int64, cursor *time.Time, limit int) ([]HashtagVideo, error)
	CreateDraft(ctx context.Context, draft *Draft) error
	GetDraft(ctx context.Context, userID, draftID int64) (*Draft, error)
	UpdateDraft(ctx context.Context, draft *Draft) error
	ListDrafts(ctx context.Context, userID int64) ([]Draft, error)
	DeleteDraft(ctx context.Context, userID, draftID int64) error
}
