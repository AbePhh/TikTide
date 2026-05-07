package model

import (
	"context"
	"errors"
	"time"
)

var (
	ErrAlreadyLiked     = errors.New("already liked")
	ErrLikeNotFound     = errors.New("like not found")
	ErrAlreadyFavorited = errors.New("already favorited")
	ErrFavoriteNotFound = errors.New("favorite not found")
	ErrCommentNotFound  = errors.New("comment not found")
)

const (
	ActionLike   int8 = 1
	ActionCancel int8 = 2
)

type VideoLike struct {
	ID        int64
	UserID    int64
	VideoID   int64
	CreatedAt time.Time
}

type Favorite struct {
	ID        int64
	UserID    int64
	VideoID   int64
	CreatedAt time.Time
}

type Comment struct {
	ID        int64
	VideoID   int64
	UserID    int64
	Content   string
	ParentID  int64
	RootID    int64
	ToUserID  int64
	LikeCount int64
	CreatedAt time.Time
	DeletedAt *time.Time
}

type CommentLike struct {
	ID        int64
	UserID    int64
	CommentID int64
	CreatedAt time.Time
}

type FavoriteVideo struct {
	VideoID         int64
	UserID          int64
	Title           string
	ObjectKey       string
	SourceURL       string
	CoverURL        string
	DurationMS      int32
	AllowComment    int8
	Visibility      int8
	TranscodeStatus int8
	AuditStatus     int8
	LikeCount       int64
	CommentCount    int64
	FavoriteCount   int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
	FavoritedAt     time.Time
}

type UserVideoAction struct {
	VideoID    int64
	ActionType string
	Weight     float64
	CreatedAt  time.Time
}

type Repository interface {
	LikeVideo(ctx context.Context, userID, videoID, authorUserID int64) error
	UnlikeVideo(ctx context.Context, userID, videoID, authorUserID int64) error
	HasLikedVideo(ctx context.Context, userID, videoID int64) (bool, error)
	FavoriteVideo(ctx context.Context, userID, videoID int64) error
	UnfavoriteVideo(ctx context.Context, userID, videoID int64) error
	HasFavoritedVideo(ctx context.Context, userID, videoID int64) (bool, error)
	ListFavorites(ctx context.Context, userID int64, cursor *time.Time, limit int) ([]FavoriteVideo, error)
	ListUserVideoActions(ctx context.Context, userID int64, limit int) ([]UserVideoAction, error)
	CreateComment(ctx context.Context, comment *Comment) error
	DeleteComment(ctx context.Context, userID, commentID int64) error
	ListComments(ctx context.Context, videoID, rootID int64, cursor *time.Time, limit int) ([]Comment, error)
	GetCommentByID(ctx context.Context, commentID int64) (*Comment, error)
	LikeComment(ctx context.Context, userID, commentID int64) error
	UnlikeComment(ctx context.Context, userID, commentID int64) error
}
