package mocks

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/AbePhh/TikTide/backend/internal/video/model"
)

type MemoryVideoRepository struct {
	mu           sync.RWMutex
	videos       map[int64]*model.Video
	resources    map[int64][]model.VideoResource
	objectKey    map[string]int64
	hashtags     map[int64]struct{}
	hashtagNames map[string]int64
	links        map[int64][]int64
	drafts       map[int64]*model.Draft
	nextDraftID  int64
	nextResID    int64
}

func NewMemoryVideoRepository(existingHashtagIDs ...int64) *MemoryVideoRepository {
	hashtags := make(map[int64]struct{}, len(existingHashtagIDs))
	hashtagNames := make(map[string]int64, len(existingHashtagIDs))
	for _, hashtagID := range existingHashtagIDs {
		hashtags[hashtagID] = struct{}{}
		hashtagNames[fmt.Sprintf("hashtag-%d", hashtagID)] = hashtagID
	}

	return &MemoryVideoRepository{
		videos:       make(map[int64]*model.Video),
		resources:    make(map[int64][]model.VideoResource),
		objectKey:    make(map[string]int64),
		hashtags:     hashtags,
		hashtagNames: hashtagNames,
		links:        make(map[int64][]int64),
		drafts:       make(map[int64]*model.Draft),
		nextDraftID:  1,
		nextResID:    1,
	}
}

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

func (r *MemoryVideoRepository) GetVideoByID(_ context.Context, videoID int64) (*model.Video, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	video, ok := r.videos[videoID]
	if !ok {
		return nil, model.ErrVideoNotFound
	}
	copy := *video
	return &copy, nil
}

func (r *MemoryVideoRepository) ListVideosByIDs(_ context.Context, videoIDs []int64) ([]model.Video, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]model.Video, 0, len(videoIDs))
	for _, videoID := range videoIDs {
		video, ok := r.videos[videoID]
		if !ok {
			continue
		}
		items = append(items, *video)
	}
	return items, nil
}

func (r *MemoryVideoRepository) ListVideosByUser(_ context.Context, userID int64, cursor *time.Time, limit int, includeInvisible bool) ([]model.Video, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]model.Video, 0)
	for _, video := range r.videos {
		if video.UserID != userID {
			continue
		}
		if cursor != nil && !video.CreatedAt.Before(*cursor) {
			continue
		}
		if !includeInvisible {
			if video.Visibility != model.VisibilityPublic || video.AuditStatus != model.AuditPassed || video.TranscodeStatus != model.TranscodeSuccess {
				continue
			}
		}
		items = append(items, *video)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID > items[j].ID
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r *MemoryVideoRepository) ListRecommendVideos(_ context.Context, limit int) ([]model.Video, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]model.Video, 0, len(r.videos))
	for _, video := range r.videos {
		if video.Visibility != model.VisibilityPublic || video.AuditStatus != model.AuditPassed || video.TranscodeStatus != model.TranscodeSuccess {
			continue
		}
		items = append(items, *video)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r *MemoryVideoRepository) ListVideosForSearch(_ context.Context, limit, offset int) ([]model.Video, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	videoIDs := make([]int64, 0, len(r.videos))
	for videoID := range r.videos {
		videoIDs = append(videoIDs, videoID)
	}
	sort.Slice(videoIDs, func(i, j int) bool { return videoIDs[i] < videoIDs[j] })

	if offset >= len(videoIDs) {
		return []model.Video{}, nil
	}

	end := offset + limit
	if end > len(videoIDs) {
		end = len(videoIDs)
	}

	items := make([]model.Video, 0, end-offset)
	for _, videoID := range videoIDs[offset:end] {
		items = append(items, *r.videos[videoID])
	}
	return items, nil
}

func (r *MemoryVideoRepository) ListHashtagNamesByVideoIDs(_ context.Context, videoIDs []int64) (map[int64][]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[int64][]string, len(videoIDs))
	for _, videoID := range videoIDs {
		for _, hashtagID := range r.links[videoID] {
			result[videoID] = append(result[videoID], r.getHashtagName(hashtagID))
		}
	}
	return result, nil
}

func (r *MemoryVideoRepository) ListVideoResources(_ context.Context, videoID int64) ([]model.VideoResource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.resources[videoID]
	result := make([]model.VideoResource, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	return result, nil
}

func (r *MemoryVideoRepository) IncreasePlayCount(_ context.Context, videoID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	video, ok := r.videos[videoID]
	if !ok {
		return model.ErrVideoNotFound
	}
	video.PlayCount++
	video.UpdatedAt = time.Now()
	return nil
}

func (r *MemoryVideoRepository) MarkVideoTranscoding(_ context.Context, videoID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	video, ok := r.videos[videoID]
	if !ok {
		return model.ErrVideoNotFound
	}
	if video.TranscodeStatus != model.TranscodePending {
		return model.ErrInvalidTranscodeState
	}
	video.TranscodeStatus = model.TranscodeProcessing
	video.TranscodeFailReason = ""
	video.UpdatedAt = time.Now()
	return nil
}

func (r *MemoryVideoRepository) MarkVideoTranscodeSuccess(_ context.Context, videoID int64, coverURL string, durationMS int32, resources []model.VideoResource) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	video, ok := r.videos[videoID]
	if !ok {
		return model.ErrVideoNotFound
	}
	if video.TranscodeStatus != model.TranscodeProcessing {
		return model.ErrInvalidTranscodeState
	}

	video.CoverURL = coverURL
	video.DurationMS = durationMS
	video.TranscodeStatus = model.TranscodeSuccess
	video.TranscodeFailReason = ""
	video.UpdatedAt = time.Now()

	stored := make([]model.VideoResource, 0, len(resources))
	for _, resource := range resources {
		copy := resource
		copy.ID = r.nextResID
		r.nextResID++
		copy.VideoID = videoID
		if copy.CreatedAt.IsZero() {
			copy.CreatedAt = time.Now()
		}
		stored = append(stored, copy)
	}
	r.resources[videoID] = stored
	return nil
}

func (r *MemoryVideoRepository) MarkVideoTranscodeFailed(_ context.Context, videoID int64, failReason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	video, ok := r.videos[videoID]
	if !ok {
		return model.ErrVideoNotFound
	}
	if video.TranscodeStatus != model.TranscodeProcessing {
		return model.ErrInvalidTranscodeState
	}
	video.TranscodeStatus = model.TranscodeFailed
	video.TranscodeFailReason = failReason
	video.UpdatedAt = time.Now()
	return nil
}

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

func (r *MemoryVideoRepository) ListHotHashtags(_ context.Context, limit int) ([]model.Hashtag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 10
	}

	items := make([]model.Hashtag, 0, len(r.hashtags))
	for hashtagID := range r.hashtags {
		items = append(items, model.Hashtag{
			ID:        hashtagID,
			Name:      r.getHashtagName(hashtagID),
			UseCount:  1,
			CreatedAt: time.Now(),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].UseCount == items[j].UseCount {
			return items[i].ID > items[j].ID
		}
		return items[i].UseCount > items[j].UseCount
	})

	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r *MemoryVideoRepository) ListHashtags(_ context.Context, limit, offset int) ([]model.Hashtag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	hashtagIDs := make([]int64, 0, len(r.hashtags))
	for hashtagID := range r.hashtags {
		hashtagIDs = append(hashtagIDs, hashtagID)
	}
	sort.Slice(hashtagIDs, func(i, j int) bool { return hashtagIDs[i] < hashtagIDs[j] })

	if offset >= len(hashtagIDs) {
		return []model.Hashtag{}, nil
	}

	end := offset + limit
	if end > len(hashtagIDs) {
		end = len(hashtagIDs)
	}

	items := make([]model.Hashtag, 0, end-offset)
	for _, hashtagID := range hashtagIDs[offset:end] {
		items = append(items, model.Hashtag{
			ID:        hashtagID,
			Name:      r.getHashtagName(hashtagID),
			UseCount:  1,
			CreatedAt: time.Now(),
		})
	}
	return items, nil
}

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
		if video.Visibility != model.VisibilityPublic || video.AuditStatus != model.AuditPassed || video.TranscodeStatus != model.TranscodeSuccess {
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

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if len(items) > limit && limit > 0 {
		items = items[:limit]
	}
	return items, nil
}

func (r *MemoryVideoRepository) CreateDraft(_ context.Context, draft *model.Draft) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	copyDraft := *draft
	copyDraft.ID = r.nextDraftID
	r.nextDraftID++
	now := time.Now()
	copyDraft.CreatedAt = now
	copyDraft.UpdatedAt = now
	r.drafts[copyDraft.ID] = &copyDraft

	draft.ID = copyDraft.ID
	draft.CreatedAt = copyDraft.CreatedAt
	draft.UpdatedAt = copyDraft.UpdatedAt
	return nil
}

func (r *MemoryVideoRepository) GetDraft(_ context.Context, userID, draftID int64) (*model.Draft, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	draft, ok := r.drafts[draftID]
	if !ok || draft.UserID != userID {
		return nil, model.ErrDraftNotFound
	}

	copyDraft := *draft
	return &copyDraft, nil
}

func (r *MemoryVideoRepository) UpdateDraft(_ context.Context, draft *model.Draft) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	current, ok := r.drafts[draft.ID]
	if !ok || current.UserID != draft.UserID {
		return model.ErrDraftNotFound
	}

	current.ObjectKey = draft.ObjectKey
	current.CoverURL = draft.CoverURL
	current.Title = draft.Title
	current.TagNames = draft.TagNames
	current.AllowComment = draft.AllowComment
	current.Visibility = draft.Visibility
	current.UpdatedAt = time.Now()

	*draft = *current
	return nil
}

func (r *MemoryVideoRepository) ListDrafts(_ context.Context, userID int64) ([]model.Draft, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]model.Draft, 0)
	for _, draft := range r.drafts {
		if draft.UserID == userID {
			items = append(items, *draft)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return items, nil
}

func (r *MemoryVideoRepository) DeleteDraft(_ context.Context, userID, draftID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	draft, ok := r.drafts[draftID]
	if !ok || draft.UserID != userID {
		return model.ErrDraftNotFound
	}
	delete(r.drafts, draftID)
	return nil
}

func (r *MemoryVideoRepository) getHashtagName(hashtagID int64) string {
	for name, id := range r.hashtagNames {
		if id == hashtagID {
			return name
		}
	}
	return "hashtag-test"
}
