package model

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type videoRecord struct {
	ID                  int64     `gorm:"column:id;primaryKey"`
	UserID              int64     `gorm:"column:user_id"`
	ObjectKey           string    `gorm:"column:object_key"`
	SourceURL           string    `gorm:"column:source_url"`
	Title               string    `gorm:"column:title"`
	CoverURL            string    `gorm:"column:cover_url"`
	DurationMS          int32     `gorm:"column:duration_ms"`
	AllowComment        int8      `gorm:"column:allow_comment"`
	Visibility          int8      `gorm:"column:visibility"`
	TranscodeStatus     int8      `gorm:"column:transcode_status"`
	AuditStatus         int8      `gorm:"column:audit_status"`
	TranscodeFailReason string    `gorm:"column:transcode_fail_reason"`
	AuditRemark         string    `gorm:"column:audit_remark"`
	PlayCount           int64     `gorm:"column:play_count"`
	LikeCount           int64     `gorm:"column:like_count"`
	CommentCount        int64     `gorm:"column:comment_count"`
	FavoriteCount       int64     `gorm:"column:favorite_count"`
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt           time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (videoRecord) TableName() string { return "t_video" }

type videoHashtagRecord struct {
	ID        int64 `gorm:"column:id;primaryKey;autoIncrement"`
	VideoID   int64 `gorm:"column:video_id"`
	HashtagID int64 `gorm:"column:hashtag_id"`
}

func (videoHashtagRecord) TableName() string { return "t_video_hashtag" }

type hashtagRecord struct {
	ID int64 `gorm:"column:id;primaryKey"`
}

func (hashtagRecord) TableName() string { return "t_hashtag" }

// MySQLRepository 基于 GORM 持久化视频数据。
type MySQLRepository struct {
	db *gorm.DB
}

// NewMySQLRepository 创建视频仓储。
func NewMySQLRepository(db *gorm.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

// CreateVideo 创建视频和话题关联关系。
func (r *MySQLRepository) CreateVideo(ctx context.Context, video *Video, hashtagIDs []int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		record := videoRecord{
			ID:                  video.ID,
			UserID:              video.UserID,
			ObjectKey:           video.ObjectKey,
			SourceURL:           video.SourceURL,
			Title:               video.Title,
			CoverURL:            video.CoverURL,
			DurationMS:          video.DurationMS,
			AllowComment:        video.AllowComment,
			Visibility:          video.Visibility,
			TranscodeStatus:     video.TranscodeStatus,
			AuditStatus:         video.AuditStatus,
			TranscodeFailReason: video.TranscodeFailReason,
			AuditRemark:         video.AuditRemark,
			PlayCount:           video.PlayCount,
			LikeCount:           video.LikeCount,
			CommentCount:        video.CommentCount,
			FavoriteCount:       video.FavoriteCount,
		}
		if err := tx.Table("t_video").Create(&record).Error; err != nil {
			return fmt.Errorf("create video: %w", err)
		}

		if len(hashtagIDs) == 0 {
			return nil
		}

		links := make([]videoHashtagRecord, 0, len(hashtagIDs))
		for _, hashtagID := range hashtagIDs {
			links = append(links, videoHashtagRecord{
				VideoID:   video.ID,
				HashtagID: hashtagID,
			})
		}

		if err := tx.Table("t_video_hashtag").Create(&links).Error; err != nil {
			return fmt.Errorf("create video hashtag links: %w", err)
		}

		if err := tx.Table("t_hashtag").
			Where("id IN ?", hashtagIDs).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Update("use_count", gorm.Expr("use_count + ?", 1)).Error; err != nil {
			return fmt.Errorf("increment hashtag use count: %w", err)
		}

		return nil
	})
}

// CountHashtagsByIDs 统计存在的话题数量。
func (r *MySQLRepository) CountHashtagsByIDs(ctx context.Context, hashtagIDs []int64) (int64, error) {
	if len(hashtagIDs) == 0 {
		return 0, nil
	}

	var count int64
	if err := r.db.WithContext(ctx).
		Table("t_hashtag").
		Where("id IN ?", hashtagIDs).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count hashtags: %w", err)
	}
	return count, nil
}
