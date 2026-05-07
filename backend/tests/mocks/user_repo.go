package mocks

import (
	"context"
	"errors"
	"sort"
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

func (r *MemoryUserRepository) ListUsersWithStats(_ context.Context, limit, offset int) ([]model.UserWithStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	userIDs := make([]int64, 0, len(r.users))
	for userID := range r.users {
		userIDs = append(userIDs, userID)
	}
	sort.Slice(userIDs, func(i, j int) bool { return userIDs[i] < userIDs[j] })

	if offset >= len(userIDs) {
		return []model.UserWithStats{}, nil
	}

	end := offset + limit
	if end > len(userIDs) {
		end = len(userIDs)
	}

	items := make([]model.UserWithStats, 0, end-offset)
	for _, userID := range userIDs[offset:end] {
		userCopy := *r.users[userID]
		statsCopy := model.UserStats{ID: userID}
		if stats, ok := r.stats[userID]; ok {
			statsCopy = *stats
		}
		items = append(items, model.UserWithStats{
			User:  userCopy,
			Stats: statsCopy,
		})
	}
	return items, nil
}

func (r *MemoryUserRepository) UpdateUsername(_ context.Context, userID int64, username string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, ok := r.users[userID]
	if !ok {
		return model.ErrUserNotFound
	}

	if existingUserID, exists := r.username[username]; exists && existingUserID != userID {
		return errors.New("duplicate username")
	}

	delete(r.username, user.Username)
	user.Username = username
	user.UpdatedAt = time.Now()
	r.username[username] = userID
	return nil
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

func (r *MemoryUserRepository) adjustStats(userID int64, followDelta, followerDelta int64) {
	stats, ok := r.stats[userID]
	if !ok {
		stats = &model.UserStats{ID: userID}
		r.stats[userID] = stats
	}

	stats.FollowCount += followDelta
	if stats.FollowCount < 0 {
		stats.FollowCount = 0
	}

	stats.FollowerCount += followerDelta
	if stats.FollowerCount < 0 {
		stats.FollowerCount = 0
	}
}

func (r *MemoryUserRepository) adjustTotalLikedCount(userID, delta int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	stats, ok := r.stats[userID]
	if !ok {
		stats = &model.UserStats{ID: userID}
		r.stats[userID] = stats
	}
	stats.TotalLikedCount += delta
	if stats.TotalLikedCount < 0 {
		stats.TotalLikedCount = 0
	}
}

func (r *MemoryUserRepository) adjustFavoriteCount(userID, delta int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	stats, ok := r.stats[userID]
	if !ok {
		stats = &model.UserStats{ID: userID}
		r.stats[userID] = stats
	}
	stats.FavoriteCount += delta
	if stats.FavoriteCount < 0 {
		stats.FavoriteCount = 0
	}
}

// BanUser 在测试中将指定用户标记为封禁。
func (r *MemoryUserRepository) BanUser(userID int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if user, ok := r.users[userID]; ok {
		user.Status = model.UserStatusBanned
	}
}
