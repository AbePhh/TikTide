package model

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// MySQLRepository 基于 GORM 持久化用户数据。
type MySQLRepository struct {
	db *gorm.DB
}

type userRecord struct {
	ID           int64      `gorm:"column:id;primaryKey"`
	Username     string     `gorm:"column:username"`
	PasswordHash string     `gorm:"column:password_hash"`
	Nickname     string     `gorm:"column:nickname"`
	AvatarURL    string     `gorm:"column:avatar_url"`
	Signature    string     `gorm:"column:signature"`
	Gender       int8       `gorm:"column:gender"`
	Birthday     *time.Time `gorm:"column:birthday"`
	Status       int8       `gorm:"column:status"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (userRecord) TableName() string { return "t_user" }

type userStatsRecord struct {
	ID              int64 `gorm:"column:id;primaryKey"`
	FollowCount     int64 `gorm:"column:follow_count"`
	FollowerCount   int64 `gorm:"column:follower_count"`
	TotalLikedCount int64 `gorm:"column:total_liked_count"`
	WorkCount       int64 `gorm:"column:work_count"`
	FavoriteCount   int64 `gorm:"column:favorite_count"`
}

func (userStatsRecord) TableName() string { return "t_user_stats" }

// NewMySQLRepository 创建 GORM 用户仓储。
func NewMySQLRepository(db *gorm.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

// Create 在一个事务中同时写入用户表和统计表。
func (r *MySQLRepository) Create(ctx context.Context, user *User, stats *UserStats) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		record := toUserRecord(user)
		if err := tx.Table("t_user").Create(&record).Error; err != nil {
			return fmt.Errorf("insert user: %w", err)
		}

		statsRecord := userStatsRecord{
			ID:              stats.ID,
			FollowCount:     stats.FollowCount,
			FollowerCount:   stats.FollowerCount,
			TotalLikedCount: stats.TotalLikedCount,
			WorkCount:       stats.WorkCount,
			FavoriteCount:   stats.FavoriteCount,
		}
		if err := tx.Table("t_user_stats").Create(&statsRecord).Error; err != nil {
			return fmt.Errorf("insert user stats: %w", err)
		}
		return nil
	})
}

// GetByID 根据用户 ID 查询用户。
func (r *MySQLRepository) GetByID(ctx context.Context, userID int64) (*User, error) {
	var record userRecord
	err := r.db.WithContext(ctx).
		Table("t_user").
		Where("id = ? AND deleted_at IS NULL", userID).
		First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("query user by id: %w", err)
	}

	return fromUserRecord(record)
}

// GetByUsername 根据用户名查询用户。
func (r *MySQLRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	var record userRecord
	err := r.db.WithContext(ctx).
		Table("t_user").
		Where("username = ? AND deleted_at IS NULL", username).
		First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("query user by username: %w", err)
	}

	return fromUserRecord(record)
}

// GetStatsByID 查询用户统计信息。
func (r *MySQLRepository) GetStatsByID(ctx context.Context, userID int64) (*UserStats, error) {
	var record userStatsRecord
	err := r.db.WithContext(ctx).
		Table("t_user_stats").
		Where("id = ?", userID).
		First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &UserStats{ID: userID}, nil
		}
		return nil, fmt.Errorf("query user stats: %w", err)
	}

	return &UserStats{
		ID:              record.ID,
		FollowCount:     record.FollowCount,
		FollowerCount:   record.FollowerCount,
		TotalLikedCount: record.TotalLikedCount,
		WorkCount:       record.WorkCount,
		FavoriteCount:   record.FavoriteCount,
	}, nil
}

func (r *MySQLRepository) ListUsersWithStats(ctx context.Context, limit, offset int) ([]UserWithStats, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	type row struct {
		ID              int64      `gorm:"column:id"`
		Username        string     `gorm:"column:username"`
		PasswordHash    string     `gorm:"column:password_hash"`
		Nickname        string     `gorm:"column:nickname"`
		AvatarURL       string     `gorm:"column:avatar_url"`
		Signature       string     `gorm:"column:signature"`
		Gender          int8       `gorm:"column:gender"`
		Birthday        *time.Time `gorm:"column:birthday"`
		Status          int8       `gorm:"column:status"`
		CreatedAt       time.Time  `gorm:"column:created_at"`
		UpdatedAt       time.Time  `gorm:"column:updated_at"`
		FollowCount     int64      `gorm:"column:follow_count"`
		FollowerCount   int64      `gorm:"column:follower_count"`
		TotalLikedCount int64      `gorm:"column:total_liked_count"`
		WorkCount       int64      `gorm:"column:work_count"`
		FavoriteCount   int64      `gorm:"column:favorite_count"`
	}

	rows := make([]row, 0, limit)
	if err := r.db.WithContext(ctx).
		Table("t_user AS u").
		Select(
			"u.id, u.username, u.password_hash, u.nickname, u.avatar_url, u.signature, u.gender, u.birthday, u.status, u.created_at, u.updated_at, " +
				"COALESCE(s.follow_count, 0) AS follow_count, COALESCE(s.follower_count, 0) AS follower_count, " +
				"COALESCE(s.total_liked_count, 0) AS total_liked_count, COALESCE(s.work_count, 0) AS work_count, COALESCE(s.favorite_count, 0) AS favorite_count",
		).
		Joins("LEFT JOIN t_user_stats AS s ON s.id = u.id").
		Where("u.deleted_at IS NULL").
		Order("u.id ASC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list users with stats: %w", err)
	}

	items := make([]UserWithStats, 0, len(rows))
	for _, item := range rows {
		items = append(items, UserWithStats{
			User: User{
				ID:           item.ID,
				Username:     item.Username,
				PasswordHash: item.PasswordHash,
				Nickname:     item.Nickname,
				AvatarURL:    item.AvatarURL,
				Signature:    item.Signature,
				Gender:       item.Gender,
				Birthday:     item.Birthday,
				Status:       item.Status,
				CreatedAt:    item.CreatedAt,
				UpdatedAt:    item.UpdatedAt,
			},
			Stats: UserStats{
				ID:              item.ID,
				FollowCount:     item.FollowCount,
				FollowerCount:   item.FollowerCount,
				TotalLikedCount: item.TotalLikedCount,
				WorkCount:       item.WorkCount,
				FavoriteCount:   item.FavoriteCount,
			},
		})
	}
	return items, nil
}

func (r *MySQLRepository) UpdateUsername(ctx context.Context, userID int64, username string) error {
	result := r.db.WithContext(ctx).
		Table("t_user").
		Where("id = ? AND deleted_at IS NULL", userID).
		Update("username", username)
	if result.Error != nil {
		return fmt.Errorf("update username: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

// UpdateProfile 更新用户可编辑资料字段。
func (r *MySQLRepository) UpdateProfile(ctx context.Context, userID int64, update ProfileUpdate) error {
	updates := make(map[string]any)
	if update.Nickname != nil {
		updates["nickname"] = *update.Nickname
	}
	if update.AvatarURL != nil {
		updates["avatar_url"] = *update.AvatarURL
	}
	if update.Signature != nil {
		updates["signature"] = *update.Signature
	}
	if update.Gender != nil {
		updates["gender"] = *update.Gender
	}
	if update.BirthdayIsSet {
		if update.Birthday == nil {
			updates["birthday"] = nil
		} else {
			updates["birthday"] = *update.Birthday
		}
	}

	if len(updates) == 0 {
		return nil
	}

	result := r.db.WithContext(ctx).
		Table("t_user").
		Where("id = ? AND deleted_at IS NULL", userID).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update profile: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdatePassword 更新用户密码哈希。
func (r *MySQLRepository) UpdatePassword(ctx context.Context, userID int64, passwordHash string) error {
	result := r.db.WithContext(ctx).
		Table("t_user").
		Where("id = ? AND deleted_at IS NULL", userID).
		Update("password_hash", passwordHash)
	if result.Error != nil {
		return fmt.Errorf("update password: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

func toUserRecord(user *User) userRecord {
	return userRecord{
		ID:           user.ID,
		Username:     user.Username,
		PasswordHash: user.PasswordHash,
		Nickname:     user.Nickname,
		AvatarURL:    user.AvatarURL,
		Signature:    user.Signature,
		Gender:       user.Gender,
		Birthday:     user.Birthday,
		Status:       user.Status,
	}
}

func fromUserRecord(record userRecord) (*User, error) {
	user := &User{
		ID:           record.ID,
		Username:     record.Username,
		PasswordHash: record.PasswordHash,
		Nickname:     record.Nickname,
		AvatarURL:    record.AvatarURL,
		Signature:    record.Signature,
		Gender:       record.Gender,
		Status:       record.Status,
	}

	user.Birthday = record.Birthday
	user.CreatedAt = record.CreatedAt
	user.UpdatedAt = record.UpdatedAt

	return user, nil
}
