package mocks

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"sync"
	"time"

	relationmodel "github.com/AbePhh/TikTide/backend/internal/relation/model"
)

// MemoryRelationRepository 是测试用的内存关注关系仓储。
type MemoryRelationRepository struct {
	mu        sync.RWMutex
	nextID    int64
	relations map[string]*relationmodel.Relation
	userRepo  *MemoryUserRepository
}

// NewMemoryRelationRepository 创建内存关注仓储。
func NewMemoryRelationRepository(userRepo *MemoryUserRepository) *MemoryRelationRepository {
	return &MemoryRelationRepository{
		nextID:    1,
		relations: make(map[string]*relationmodel.Relation),
		userRepo:  userRepo,
	}
}

func (r *MemoryRelationRepository) Create(_ context.Context, userID, followID int64) (*relationmodel.Relation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := relationKey(userID, followID)
	if _, exists := r.relations[key]; exists {
		return nil, errors.New("duplicate relation")
	}

	now := time.Now()
	relation := &relationmodel.Relation{
		ID:        r.nextID,
		UserID:    userID,
		FollowID:  followID,
		CreatedAt: now,
	}
	r.nextID++

	if reverse, exists := r.relations[relationKey(followID, userID)]; exists {
		relation.IsMutual = true
		reverse.IsMutual = true
	}

	r.relations[key] = relation
	r.userRepo.adjustStats(userID, 1, 0)
	r.userRepo.adjustStats(followID, 0, 1)

	copyRelation := *relation
	return &copyRelation, nil
}

func (r *MemoryRelationRepository) Delete(_ context.Context, userID, followID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := relationKey(userID, followID)
	relation, exists := r.relations[key]
	if !exists {
		return relationmodel.ErrRelationNotFound
	}

	delete(r.relations, key)
	r.userRepo.adjustStats(userID, -1, 0)
	r.userRepo.adjustStats(followID, 0, -1)

	if reverse, exists := r.relations[relationKey(followID, userID)]; exists {
		reverse.IsMutual = false
	}

	_ = relation
	return nil
}

func (r *MemoryRelationRepository) Get(_ context.Context, userID, followID int64) (*relationmodel.Relation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	relation, exists := r.relations[relationKey(userID, followID)]
	if !exists {
		return nil, relationmodel.ErrRelationNotFound
	}

	copyRelation := *relation
	return &copyRelation, nil
}

func (r *MemoryRelationRepository) ListFollowing(_ context.Context, userID, cursor int64, limit int) ([]relationmodel.Relation, error) {
	return r.list(userID, cursor, limit, true), nil
}

func (r *MemoryRelationRepository) ListFollowers(_ context.Context, userID, cursor int64, limit int) ([]relationmodel.Relation, error) {
	return r.list(userID, cursor, limit, false), nil
}

func (r *MemoryRelationRepository) ListFollowersAll(_ context.Context, userID int64) ([]relationmodel.Relation, error) {
	return r.list(userID, 0, 1<<30, false), nil
}

func (r *MemoryRelationRepository) ListFollowingAll(_ context.Context, userID int64) ([]relationmodel.Relation, error) {
	return r.list(userID, 0, 1<<30, true), nil
}

func (r *MemoryRelationRepository) list(userID, cursor int64, limit int, following bool) []relationmodel.Relation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]relationmodel.Relation, 0)
	for _, relation := range r.relations {
		if following && relation.UserID != userID {
			continue
		}
		if !following && relation.FollowID != userID {
			continue
		}
		if cursor > 0 && relation.ID >= cursor {
			continue
		}

		items = append(items, *relation)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID > items[j].ID
	})

	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

func relationKey(userID, followID int64) string {
	return stringKey(userID) + ":" + stringKey(followID)
}

func stringKey(value int64) string {
	return strconv.FormatInt(value, 10)
}
