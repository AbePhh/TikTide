package model

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// MySQLRepository 基于 GORM 实现关注关系读写。
type MySQLRepository struct {
	db *gorm.DB
}

type relationRecord struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement"`
	UserID    int64     `gorm:"column:user_id"`
	FollowID  int64     `gorm:"column:follow_id"`
	IsMutual  bool      `gorm:"column:is_mutual"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (relationRecord) TableName() string { return "t_relation" }

// NewMySQLRepository 创建关注关系仓储。
func NewMySQLRepository(db *gorm.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

// Create 创建关注关系，并同步维护关注统计与互关状态。
func (r *MySQLRepository) Create(ctx context.Context, userID, followID int64) (*Relation, error) {
	record := relationRecord{
		UserID:   userID,
		FollowID: followID,
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("t_relation").Create(&record).Error; err != nil {
			return fmt.Errorf("insert relation: %w", err)
		}

		if err := tx.Table("t_user_stats").
			Where("id = ?", userID).
			Update("follow_count", gorm.Expr("follow_count + ?", 1)).Error; err != nil {
			return fmt.Errorf("increase follow count: %w", err)
		}

		if err := tx.Table("t_user_stats").
			Where("id = ?", followID).
			Update("follower_count", gorm.Expr("follower_count + ?", 1)).Error; err != nil {
			return fmt.Errorf("increase follower count: %w", err)
		}

		var reverse relationRecord
		err := tx.Table("t_relation").
			Where("user_id = ? AND follow_id = ?", followID, userID).
			First(&reverse).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return fmt.Errorf("query reverse relation: %w", err)
		}

		if err := tx.Table("t_relation").
			Where("id IN ?", []int64{record.ID, reverse.ID}).
			Update("is_mutual", true).Error; err != nil {
			return fmt.Errorf("update mutual flag: %w", err)
		}

		record.IsMutual = true
		return nil
	})
	if err != nil {
		return nil, err
	}

	return fromRelationRecord(record), nil
}

// Delete 删除关注关系，并同步维护关注统计与互关状态。
func (r *MySQLRepository) Delete(ctx context.Context, userID, followID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record relationRecord
		if err := tx.Table("t_relation").
			Where("user_id = ? AND follow_id = ?", userID, followID).
			First(&record).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRelationNotFound
			}
			return fmt.Errorf("query relation before delete: %w", err)
		}

		if err := tx.Table("t_relation").Delete(&record).Error; err != nil {
			return fmt.Errorf("delete relation: %w", err)
		}

		if err := tx.Table("t_user_stats").
			Where("id = ?", userID).
			Update("follow_count", gorm.Expr("GREATEST(follow_count - ?, 0)", 1)).Error; err != nil {
			return fmt.Errorf("decrease follow count: %w", err)
		}

		if err := tx.Table("t_user_stats").
			Where("id = ?", followID).
			Update("follower_count", gorm.Expr("GREATEST(follower_count - ?, 0)", 1)).Error; err != nil {
			return fmt.Errorf("decrease follower count: %w", err)
		}

		if err := tx.Table("t_relation").
			Where("user_id = ? AND follow_id = ?", followID, userID).
			Update("is_mutual", false).Error; err != nil {
			return fmt.Errorf("clear reverse mutual flag: %w", err)
		}

		return nil
	})
}

// Get 查询一条关注关系。
func (r *MySQLRepository) Get(ctx context.Context, userID, followID int64) (*Relation, error) {
	var record relationRecord
	err := r.db.WithContext(ctx).
		Table("t_relation").
		Where("user_id = ? AND follow_id = ?", userID, followID).
		First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRelationNotFound
		}
		return nil, fmt.Errorf("query relation: %w", err)
	}

	return fromRelationRecord(record), nil
}

// ListFollowing 返回指定用户的关注列表。
func (r *MySQLRepository) ListFollowing(ctx context.Context, userID, cursor int64, limit int) ([]Relation, error) {
	query := r.db.WithContext(ctx).
		Table("t_relation").
		Where("user_id = ?", userID)
	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}

	var records []relationRecord
	if err := query.Order("id DESC").Limit(limit).Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list following: %w", err)
	}

	return fromRelationRecords(records), nil
}

// ListFollowers 返回指定用户的粉丝列表。
func (r *MySQLRepository) ListFollowers(ctx context.Context, userID, cursor int64, limit int) ([]Relation, error) {
	query := r.db.WithContext(ctx).
		Table("t_relation").
		Where("follow_id = ?", userID)
	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}

	var records []relationRecord
	if err := query.Order("id DESC").Limit(limit).Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list followers: %w", err)
	}

	return fromRelationRecords(records), nil
}

// ListFollowersAll 返回指定用户全部粉丝关系，用于关注流分发。
func (r *MySQLRepository) ListFollowersAll(ctx context.Context, userID int64) ([]Relation, error) {
	var records []relationRecord
	if err := r.db.WithContext(ctx).
		Table("t_relation").
		Where("follow_id = ?", userID).
		Order("id DESC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list all followers: %w", err)
	}

	return fromRelationRecords(records), nil
}

// ListFollowingAll 返回指定用户全部关注关系，用于关注流合并拉取。
func (r *MySQLRepository) ListFollowingAll(ctx context.Context, userID int64) ([]Relation, error) {
	var records []relationRecord
	if err := r.db.WithContext(ctx).
		Table("t_relation").
		Where("user_id = ?", userID).
		Order("id DESC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list all following: %w", err)
	}

	return fromRelationRecords(records), nil
}

func fromRelationRecord(record relationRecord) *Relation {
	return &Relation{
		ID:        record.ID,
		UserID:    record.UserID,
		FollowID:  record.FollowID,
		IsMutual:  record.IsMutual,
		CreatedAt: record.CreatedAt,
	}
}

func fromRelationRecords(records []relationRecord) []Relation {
	items := make([]Relation, 0, len(records))
	for _, record := range records {
		items = append(items, *fromRelationRecord(record))
	}
	return items
}
