package mocks

import (
	"context"
	"sync"

	"github.com/AbePhh/TikTide/backend/internal/video/model"
)

// MemoryVideoRepository 是测试用内存视频仓储。
type MemoryVideoRepository struct {
	mu        sync.RWMutex
	videos    map[int64]*model.Video
	objectKey map[string]int64
	hashtags  map[int64]struct{}
	links     map[int64][]int64
}

// NewMemoryVideoRepository 创建新的内存视频仓储。
func NewMemoryVideoRepository(existingHashtagIDs ...int64) *MemoryVideoRepository {
	hashtags := make(map[int64]struct{}, len(existingHashtagIDs))
	for _, hashtagID := range existingHashtagIDs {
		hashtags[hashtagID] = struct{}{}
	}

	return &MemoryVideoRepository{
		videos:    make(map[int64]*model.Video),
		objectKey: make(map[string]int64),
		hashtags:  hashtags,
		links:     make(map[int64][]int64),
	}
}

// CreateVideo 将视频写入内存仓储。
func (r *MemoryVideoRepository) CreateVideo(_ context.Context, video *model.Video, hashtagIDs []int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	videoCopy := *video
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
