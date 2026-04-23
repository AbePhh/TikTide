package mocks

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/AbePhh/TikTide/backend/internal/user/model"
)

// MemoryUserRepository 是测试用内存用户仓储。
type MemoryUserRepository struct {
	mu       sync.RWMutex
	users    map[int64]*model.User
	username map[string]int64
	stats    map[int64]*model.UserStats
}

// NewMemoryUserRepository 创建新的内存仓储。
func NewMemoryUserRepository() *MemoryUserRepository {
	return &MemoryUserRepository{
		users:    make(map[int64]*model.User),
		username: make(map[string]int64),
		stats:    make(map[int64]*model.UserStats),
	}
}

func (r *MemoryUserRepository) Create(_ context.Context, user *model.User, stats *model.UserStats) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.username[user.Username]; exists {
		return errors.New("duplicate username")
	}

	now := time.Now()
	userCopy := *user
	userCopy.CreatedAt = now
	userCopy.UpdatedAt = now

	statsCopy := *stats
	r.users[user.ID] = &userCopy
	r.username[user.Username] = user.ID
	r.stats[user.ID] = &statsCopy
	return nil
}

func (r *MemoryUserRepository) GetByID(_ context.Context, userID int64) (*model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[userID]
	if !ok {
		return nil, model.ErrUserNotFound
	}
	copy := *user
	return &copy, nil
}

func (r *MemoryUserRepository) GetByUsername(_ context.Context, username string) (*model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userID, ok := r.username[username]
	if !ok {
		return nil, model.ErrUserNotFound
	}
	copy := *r.users[userID]
	return &copy, nil
}

func (r *MemoryUserRepository) GetStatsByID(_ context.Context, userID int64) (*model.UserStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats, ok := r.stats[userID]
	if !ok {
		return &model.UserStats{ID: userID}, nil
	}
	copy := *stats
	return &copy, nil
}

func (r *MemoryUserRepository) UpdateProfile(_ context.Context, userID int64, update model.ProfileUpdate) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, ok := r.users[userID]
	if !ok {
		return model.ErrUserNotFound
	}

	if update.Nickname != nil {
		user.Nickname = *update.Nickname
	}
	if update.AvatarURL != nil {
		user.AvatarURL = *update.AvatarURL
	}
	if update.Signature != nil {
		user.Signature = *update.Signature
	}
	if update.Gender != nil {
		user.Gender = *update.Gender
	}
	if update.BirthdayIsSet {
		user.Birthday = update.Birthday
	}
	user.UpdatedAt = time.Now()
	return nil
}

func (r *MemoryUserRepository) UpdatePassword(_ context.Context, userID int64, passwordHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, ok := r.users[userID]
	if !ok {
		return model.ErrUserNotFound
	}
	user.PasswordHash = passwordHash
	user.UpdatedAt = time.Now()
	return nil
}

// BanUser 在测试中将指定用户标记为封禁。
func (r *MemoryUserRepository) BanUser(userID int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if user, ok := r.users[userID]; ok {
		user.Status = model.UserStatusBanned
	}
}
