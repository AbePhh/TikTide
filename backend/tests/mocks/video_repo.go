package mocks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AbePhh/TikTide/backend/internal/video/model"
)

// MemoryVideoRepository 是测试用内存视频仓储。
type MemoryVideoRepository struct {
	mu           sync.RWMutex
	videos       map[int64]*model.Video
	objectKey    map[string]int64
	hashtags     map[int64]struct{}
	hashtagNames map[string]int64
	links        map[int64][]int64
}

// NewMemoryVideoRepository 创建新的内存视频仓储。
func NewMemoryVideoRepository(existingHashtagIDs ...int64) *MemoryVideoRepository {
	hashtags := make(map[int64]struct{}, len(existingHashtagIDs))
	hashtagNames := make(map[string]int64, len(existingHashtagIDs))
	for _, hashtagID := range existingHashtagIDs {
		hashtags[hashtagID] = struct{}{}
		hashtagNames[fmt.Sprintf("hashtag-%d", hashtagID)] = hashtagID
	}

	return &MemoryVideoRepository{
		videos:       make(map[int64]*model.Video),
		objectKey:    make(map[string]int64),
		hashtags:     hashtags,
		hashtagNames: hashtagNames,
		links:        make(map[int64][]int64),
	}
}

// CreateVideo 将视频写入内存仓储。
func (r *MemoryVideoRepository) CreateVideo(_ context.Context, video *model.Video, hashtagIDs []int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	videoCopy := *video
	now := time.Now()
	if videoCopy.CreatedAt.IsZero() {
		videoCopy.CreatedAt = now
	}
	if videoCopy.UpdatedAt.IsZero() {
		videoCopy.UpdatedAt = now
	}
	r.videos[video.ID] = &videoCopy
	r.objectKey[video.ObjectKey] = video.ID
	r.links[video.ID] = append([]int64(nil), hashtagIDs...)
	return nil
}

// CountHashtagsByIDs 统计存在的话题数量。
func (r *MemoryVideoRepository) CountHashtagsByIDs(_ context.Context, hashtagIDs []int64) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var count int64
	for _, hashtagID := range hashtagIDs {
		if _, ok := r.hashtags[hashtagID]; ok {
			count++
		}
	}
	return count, nil
}

// GetHashtagByID 根据话题 ID 查询测试话题。
func (r *MemoryVideoRepository) GetHashtagByID(_ context.Context, hashtagID int64) (*model.Hashtag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.hashtags[hashtagID]; !ok {
		return nil, model.ErrHashtagNotFound
	}

	return &model.Hashtag{
		ID:        hashtagID,
		Name:      r.getHashtagName(hashtagID),
		UseCount:  1,
		CreatedAt: time.Now(),
	}, nil
}

// CreateHashtag 按名称创建或返回已有测试话题。
func (r *MemoryVideoRepository) CreateHashtag(_ context.Context, name string) (*model.Hashtag, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if hashtagID, exists := r.hashtagNames[name]; exists {
		return &model.Hashtag{
			ID:        hashtagID,
			Name:      name,
			UseCount:  0,
			CreatedAt: time.Now(),
		}, nil
	}

	newID := int64(len(r.hashtags) + 1)
	for {
		if _, exists := r.hashtags[newID]; !exists {
			break
		}
		newID++
	}
	r.hashtags[newID] = struct{}{}
	r.hashtagNames[name] = newID

	return &model.Hashtag{
		ID:        newID,
		Name:      name,
		UseCount:  0,
		CreatedAt: time.Now(),
	}, nil
}

// ListVideosByHashtag 返回测试话题下视频列表。
func (r *MemoryVideoRepository) ListVideosByHashtag(_ context.Context, hashtagID int64, cursor *time.Time, limit int) ([]model.HashtagVideo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]model.HashtagVideo, 0)
	for videoID, hashtagIDs := range r.links {
		matched := false
		for _, id := range hashtagIDs {
			if id == hashtagID {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}

		video := r.videos[videoID]
		if video == nil {
			continue
		}
		if cursor != nil && !video.CreatedAt.Before(*cursor) {
			continue
		}
		items = append(items, model.HashtagVideo{
			VideoID:         video.ID,
			UserID:          video.UserID,
			Title:           video.Title,
			ObjectKey:       video.ObjectKey,
			SourceURL:       video.SourceURL,
			CoverURL:        video.CoverURL,
			Visibility:      video.Visibility,
			TranscodeStatus: video.TranscodeStatus,
			AuditStatus:     video.AuditStatus,
			CreatedAt:       video.CreatedAt,
		})
	}

	if len(items) > limit && limit > 0 {
		items = items[:limit]
	}
	return items, nil
}

func (r *MemoryVideoRepository) getHashtagName(hashtagID int64) string {
	for name, id := range r.hashtagNames {
		if id == hashtagID {
			return name
		}
	}
	return "hashtag-test"
}
