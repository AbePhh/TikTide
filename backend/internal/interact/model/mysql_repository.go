package model

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

type likeRecord struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement"`
	UserID    int64     `gorm:"column:user_id"`
	VideoID   int64     `gorm:"column:video_id"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (likeRecord) TableName() string { return "t_like" }

type favoriteRecord struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement"`
	UserID    int64     `gorm:"column:user_id"`
	VideoID   int64     `gorm:"column:video_id"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (favoriteRecord) TableName() string { return "t_favorite" }

type commentRecord struct {
	ID        int64      `gorm:"column:id;primaryKey"`
	VideoID   int64      `gorm:"column:video_id"`
	UserID    int64      `gorm:"column:user_id"`
	Content   string     `gorm:"column:content"`
	ParentID  int64      `gorm:"column:parent_id"`
	RootID    int64      `gorm:"column:root_id"`
	ToUserID  int64      `gorm:"column:to_user_id"`
	LikeCount int64      `gorm:"column:like_count"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime"`
	DeletedAt *time.Time `gorm:"column:deleted_at"`
}

func (commentRecord) TableName() string { return "t_comment" }

type commentLikeRecord struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement"`
	UserID    int64     `gorm:"column:user_id"`
	CommentID int64     `gorm:"column:comment_id"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (commentLikeRecord) TableName() string { return "t_comment_like" }

type MySQLRepository struct {
	db *gorm.DB
}

func NewMySQLRepository(db *gorm.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) LikeVideo(ctx context.Context, userID, videoID, authorUserID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		record := likeRecord{UserID: userID, VideoID: videoID}
		if err := tx.Table("t_like").Create(&record).Error; err != nil {
			if isDuplicateError(err) {
				return ErrAlreadyLiked
			}
			return fmt.Errorf("create like: %w", err)
		}
		if err := tx.Table("t_video").Where("id = ? AND deleted_at IS NULL", videoID).
			Updates(map[string]any{"like_count": gorm.Expr("like_count + 1")}).Error; err != nil {
			return fmt.Errorf("increase video like count: %w", err)
		}
		if authorUserID > 0 && authorUserID != userID {
			if err := tx.Table("t_user_stats").Where("id = ?", authorUserID).
				Updates(map[string]any{"total_liked_count": gorm.Expr("total_liked_count + 1")}).Error; err != nil {
				return fmt.Errorf("increase author liked count: %w", err)
			}
		}
		return nil
	})
}

func (r *MySQLRepository) UnlikeVideo(ctx context.Context, userID, videoID, authorUserID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Table("t_like").Where("user_id = ? AND video_id = ?", userID, videoID).Delete(&likeRecord{})
		if result.Error != nil {
			return fmt.Errorf("delete like: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return ErrLikeNotFound
		}
		if err := tx.Table("t_video").Where("id = ? AND deleted_at IS NULL", videoID).
			Updates(map[string]any{"like_count": gorm.Expr("CASE WHEN like_count > 0 THEN like_count - 1 ELSE 0 END")}).Error; err != nil {
			return fmt.Errorf("decrease video like count: %w", err)
		}
		if authorUserID > 0 && authorUserID != userID {
			if err := tx.Table("t_user_stats").Where("id = ?", authorUserID).
				Updates(map[string]any{"total_liked_count": gorm.Expr("CASE WHEN total_liked_count > 0 THEN total_liked_count - 1 ELSE 0 END")}).Error; err != nil {
				return fmt.Errorf("decrease author liked count: %w", err)
			}
		}
		return nil
	})
}

func (r *MySQLRepository) FavoriteVideo(ctx context.Context, userID, videoID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		record := favoriteRecord{UserID: userID, VideoID: videoID}
		if err := tx.Table("t_favorite").Create(&record).Error; err != nil {
			if isDuplicateError(err) {
				return ErrAlreadyFavorited
			}
			return fmt.Errorf("create favorite: %w", err)
		}
		if err := tx.Table("t_video").Where("id = ? AND deleted_at IS NULL", videoID).
			Updates(map[string]any{"favorite_count": gorm.Expr("favorite_count + 1")}).Error; err != nil {
			return fmt.Errorf("increase video favorite count: %w", err)
		}
		if err := tx.Table("t_user_stats").Where("id = ?", userID).
			Updates(map[string]any{"favorite_count": gorm.Expr("favorite_count + 1")}).Error; err != nil {
			return fmt.Errorf("increase user favorite count: %w", err)
		}
		return nil
	})
}

func (r *MySQLRepository) HasLikedVideo(ctx context.Context, userID, videoID int64) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Table("t_like").
		Where("user_id = ? AND video_id = ?", userID, videoID).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("count likes: %w", err)
	}
	return count > 0, nil
}

func (r *MySQLRepository) HasFavoritedVideo(ctx context.Context, userID, videoID int64) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Table("t_favorite").
		Where("user_id = ? AND video_id = ?", userID, videoID).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("count favorites: %w", err)
	}
	return count > 0, nil
}

func (r *MySQLRepository) UnfavoriteVideo(ctx context.Context, userID, videoID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Table("t_favorite").Where("user_id = ? AND video_id = ?", userID, videoID).Delete(&favoriteRecord{})
		if result.Error != nil {
			return fmt.Errorf("delete favorite: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return ErrFavoriteNotFound
		}
		if err := tx.Table("t_video").Where("id = ? AND deleted_at IS NULL", videoID).
			Updates(map[string]any{"favorite_count": gorm.Expr("CASE WHEN favorite_count > 0 THEN favorite_count - 1 ELSE 0 END")}).Error; err != nil {
			return fmt.Errorf("decrease video favorite count: %w", err)
		}
		if err := tx.Table("t_user_stats").Where("id = ?", userID).
			Updates(map[string]any{"favorite_count": gorm.Expr("CASE WHEN favorite_count > 0 THEN favorite_count - 1 ELSE 0 END")}).Error; err != nil {
			return fmt.Errorf("decrease user favorite count: %w", err)
		}
		return nil
	})
}

func (r *MySQLRepository) ListFavorites(ctx context.Context, userID int64, cursor *time.Time, limit int) ([]FavoriteVideo, error) {
	query := r.db.WithContext(ctx).
		Table("t_favorite AS f").
		Select(`v.id AS video_id, v.user_id, v.title, v.object_key, v.source_url, v.cover_url, v.duration_ms, v.allow_comment, v.visibility, v.transcode_status, v.audit_status, v.like_count, v.comment_count, v.favorite_count, v.created_at, v.updated_at, f.created_at AS favorited_at`).
		Joins("JOIN t_video AS v ON v.id = f.video_id").
		Where("f.user_id = ? AND v.deleted_at IS NULL", userID)
	if cursor != nil {
		query = query.Where("f.created_at < ?", *cursor)
	}
	var items []FavoriteVideo
	if err := query.Order("f.created_at DESC").Limit(limit).Scan(&items).Error; err != nil {
		return nil, fmt.Errorf("list favorites: %w", err)
	}
	return items, nil
}

func (r *MySQLRepository) ListUserVideoActions(ctx context.Context, userID int64, limit int) ([]UserVideoAction, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}

	type actionRow struct {
		VideoID    int64     `gorm:"column:video_id"`
		ActionType string    `gorm:"column:action_type"`
		Weight     float64   `gorm:"column:weight"`
		CreatedAt  time.Time `gorm:"column:created_at"`
	}

	rows := make([]actionRow, 0, limit*3)

	var likeRows []actionRow
	if err := r.db.WithContext(ctx).
		Table("t_like").
		Select("video_id, ? AS action_type, ? AS weight, created_at", "like", 1.0).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Scan(&likeRows).Error; err != nil {
		return nil, fmt.Errorf("list user like actions: %w", err)
	}
	rows = append(rows, likeRows...)

	var favoriteRows []actionRow
	if err := r.db.WithContext(ctx).
		Table("t_favorite").
		Select("video_id, ? AS action_type, ? AS weight, created_at", "favorite", 3.0).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Scan(&favoriteRows).Error; err != nil {
		return nil, fmt.Errorf("list user favorite actions: %w", err)
	}
	rows = append(rows, favoriteRows...)

	var commentRows []actionRow
	if err := r.db.WithContext(ctx).
		Table("t_comment").
		Select("video_id, ? AS action_type, ? AS weight, created_at", "comment", 2.0).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("created_at DESC").
		Limit(limit).
		Scan(&commentRows).Error; err != nil {
		return nil, fmt.Errorf("list user comment actions: %w", err)
	}
	rows = append(rows, commentRows...)

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].CreatedAt.After(rows[j].CreatedAt)
	})
	if len(rows) > limit {
		rows = rows[:limit]
	}

	items := make([]UserVideoAction, 0, len(rows))
	for _, row := range rows {
		items = append(items, UserVideoAction{
			VideoID:    row.VideoID,
			ActionType: row.ActionType,
			Weight:     row.Weight,
			CreatedAt:  row.CreatedAt,
		})
	}
	return items, nil
}

func (r *MySQLRepository) CreateComment(ctx context.Context, comment *Comment) error {
	record := commentRecord{
		ID:       comment.ID,
		VideoID:  comment.VideoID,
		UserID:   comment.UserID,
		Content:  comment.Content,
		ParentID: comment.ParentID,
		RootID:   comment.RootID,
		ToUserID: comment.ToUserID,
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("t_comment").Create(&record).Error; err != nil {
			return fmt.Errorf("create comment: %w", err)
		}
		if err := tx.Table("t_video").Where("id = ? AND deleted_at IS NULL", comment.VideoID).
			Updates(map[string]any{"comment_count": gorm.Expr("comment_count + 1")}).Error; err != nil {
			return fmt.Errorf("increase comment count: %w", err)
		}
		comment.CreatedAt = record.CreatedAt
		return nil
	})
}

func (r *MySQLRepository) DeleteComment(ctx context.Context, userID, commentID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record commentRecord
		if err := tx.Table("t_comment").
			Where("id = ? AND user_id = ?", commentID, userID).
			First(&record).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrCommentNotFound
			}
			return fmt.Errorf("query comment before delete: %w", err)
		}
		if record.DeletedAt != nil {
			return nil
		}

		now := time.Now()
		if err := tx.Table("t_comment").
			Where("id = ?", commentID).
			Update("deleted_at", now).Error; err != nil {
			return fmt.Errorf("soft delete comment: %w", err)
		}
		if err := tx.Table("t_video").Where("id = ? AND deleted_at IS NULL", record.VideoID).
			Updates(map[string]any{"comment_count": gorm.Expr("CASE WHEN comment_count > 0 THEN comment_count - 1 ELSE 0 END")}).Error; err != nil {
			return fmt.Errorf("decrease comment count: %w", err)
		}
		return nil
	})
}

func (r *MySQLRepository) ListComments(ctx context.Context, videoID, rootID int64, cursor *time.Time, limit int) ([]Comment, error) {
	query := r.db.WithContext(ctx).
		Table("t_comment").
		Where("video_id = ? AND root_id = ?", videoID, rootID)
	if rootID == 0 {
		query = query.Where("parent_id = 0 AND deleted_at IS NULL")
	} else {
		var root commentRecord
		if err := r.db.WithContext(ctx).
			Table("t_comment").
			Where("id = ? AND video_id = ?", rootID, videoID).
			First(&root).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return []Comment{}, nil
			}
			return nil, fmt.Errorf("get root comment: %w", err)
		}
		if root.DeletedAt != nil {
			return []Comment{}, nil
		}
	}
	if cursor != nil {
		query = query.Where("created_at < ?", *cursor)
	}
	var records []commentRecord
	if err := query.Order("created_at DESC").Limit(limit).Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	items := make([]Comment, 0, len(records))
	for _, record := range records {
		items = append(items, Comment{
			ID:        record.ID,
			VideoID:   record.VideoID,
			UserID:    record.UserID,
			Content:   record.Content,
			ParentID:  record.ParentID,
			RootID:    record.RootID,
			ToUserID:  record.ToUserID,
			LikeCount: record.LikeCount,
			CreatedAt: record.CreatedAt,
			DeletedAt: record.DeletedAt,
		})
	}
	return items, nil
}

func (r *MySQLRepository) GetCommentByID(ctx context.Context, commentID int64) (*Comment, error) {
	var record commentRecord
	if err := r.db.WithContext(ctx).Table("t_comment").Where("id = ?", commentID).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCommentNotFound
		}
		return nil, fmt.Errorf("get comment: %w", err)
	}
	return &Comment{
		ID:        record.ID,
		VideoID:   record.VideoID,
		UserID:    record.UserID,
		Content:   record.Content,
		ParentID:  record.ParentID,
		RootID:    record.RootID,
		ToUserID:  record.ToUserID,
		LikeCount: record.LikeCount,
		CreatedAt: record.CreatedAt,
		DeletedAt: record.DeletedAt,
	}, nil
}

func (r *MySQLRepository) LikeComment(ctx context.Context, userID, commentID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		record := commentLikeRecord{UserID: userID, CommentID: commentID}
		if err := tx.Table("t_comment_like").Create(&record).Error; err != nil {
			if isDuplicateError(err) {
				return ErrAlreadyLiked
			}
			return fmt.Errorf("create comment like: %w", err)
		}
		if err := tx.Table("t_comment").Where("id = ? AND deleted_at IS NULL", commentID).
			Updates(map[string]any{"like_count": gorm.Expr("like_count + 1")}).Error; err != nil {
			return fmt.Errorf("increase comment like count: %w", err)
		}
		return nil
	})
}

func (r *MySQLRepository) UnlikeComment(ctx context.Context, userID, commentID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Table("t_comment_like").Where("user_id = ? AND comment_id = ?", userID, commentID).Delete(&commentLikeRecord{})
		if result.Error != nil {
			return fmt.Errorf("delete comment like: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return ErrLikeNotFound
		}
		if err := tx.Table("t_comment").Where("id = ? AND deleted_at IS NULL", commentID).
			Updates(map[string]any{"like_count": gorm.Expr("CASE WHEN like_count > 0 THEN like_count - 1 ELSE 0 END")}).Error; err != nil {
			return fmt.Errorf("decrease comment like count: %w", err)
		}
		return nil
	})
}

func isDuplicateError(err error) bool {
	raw := strings.ToLower(err.Error())
	return strings.Contains(raw, "duplicate") || strings.Contains(raw, "unique")
}
