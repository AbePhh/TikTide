package model

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrUserNotFound 表示用户不存在。
	ErrUserNotFound = errors.New("user not found")
)

const (
	// UserStatusBanned 表示用户已封禁。
	UserStatusBanned int8 = 0
	// UserStatusActive 表示用户状态正常。
	UserStatusActive int8 = 1
)

// User 对应 t_user 表。
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Nickname     string
	AvatarURL    string
	Signature    string
	Gender       int8
	Birthday     *time.Time
	Status       int8
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// UserStats 对应 t_user_stats 表。
type UserStats struct {
	ID              int64
	FollowCount     int64
	FollowerCount   int64
	TotalLikedCount int64
	WorkCount       int64
	FavoriteCount   int64
}

// ProfileUpdate 描述允许更新的资料字段。
type ProfileUpdate struct {
	Nickname      *string
	AvatarURL     *string
	Signature     *string
	Gender        *int8
	Birthday      *time.Time
	BirthdayIsSet bool
}

// Repository 定义用户服务需要的持久化操作。
type Repository interface {
	Create(ctx context.Context, user *User, stats *UserStats) error
	GetByID(ctx context.Context, userID int64) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetStatsByID(ctx context.Context, userID int64) (*UserStats, error)
	UpdateProfile(ctx context.Context, userID int64, update ProfileUpdate) error
	UpdatePassword(ctx context.Context, userID int64, passwordHash string) error
}

func parseBirthday(raw string) (*time.Time, error) {
	value, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func parseDateTime(raw string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02 15:04:05", raw, time.Local)
}
