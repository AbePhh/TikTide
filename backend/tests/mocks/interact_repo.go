package mocks

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	interactmodel "github.com/AbePhh/TikTide/backend/internal/interact/model"
)

type MemoryInteractRepository struct {
	mu           sync.RWMutex
	videoRepo    *MemoryVideoRepository
	userRepo     *MemoryUserRepository
	videoLikes   map[string]time.Time
	favorites    map[string]time.Time
	commentLikes map[string]time.Time
	comments     map[int64]*interactmodel.Comment
}

func NewMemoryInteractRepository(videoRepo *MemoryVideoRepository, userRepo *MemoryUserRepository) *MemoryInteractRepository {
	return &MemoryInteractRepository{
		videoRepo:    videoRepo,
		userRepo:     userRepo,
		videoLikes:   make(map[string]time.Time),
		favorites:    make(map[string]time.Time),
		commentLikes: make(map[string]time.Time),
		comments:     make(map[int64]*interactmodel.Comment),
	}
}

func (r *MemoryInteractRepository) LikeVideo(_ context.Context, userID, videoID, authorUserID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := pairKey(userID, videoID)
	if _, exists := r.videoLikes[key]; exists {
		return interactmodel.ErrAlreadyLiked
	}
	r.videoLikes[key] = time.Now()
	if video := r.videoRepo.videos[videoID]; video != nil {
		video.LikeCount++
		video.UpdatedAt = time.Now()
	}
	if authorUserID > 0 && authorUserID != userID {
		r.userRepo.adjustTotalLikedCount(authorUserID, 1)
	}
	return nil
}

func (r *MemoryInteractRepository) UnlikeVideo(_ context.Context, userID, videoID, authorUserID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := pairKey(userID, videoID)
	if _, exists := r.videoLikes[key]; !exists {
		return interactmodel.ErrLikeNotFound
	}
	delete(r.videoLikes, key)
	if video := r.videoRepo.videos[videoID]; video != nil && video.LikeCount > 0 {
		video.LikeCount--
		video.UpdatedAt = time.Now()
	}
	if authorUserID > 0 && authorUserID != userID {
		r.userRepo.adjustTotalLikedCount(authorUserID, -1)
	}
	return nil
}

func (r *MemoryInteractRepository) FavoriteVideo(_ context.Context, userID, videoID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := pairKey(userID, videoID)
	if _, exists := r.favorites[key]; exists {
		return interactmodel.ErrAlreadyFavorited
	}
	r.favorites[key] = time.Now()
	if video := r.videoRepo.videos[videoID]; video != nil {
		video.FavoriteCount++
		video.UpdatedAt = time.Now()
	}
	r.userRepo.adjustFavoriteCount(userID, 1)
	return nil
}

func (r *MemoryInteractRepository) HasLikedVideo(_ context.Context, userID, videoID int64) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.videoLikes[pairKey(userID, videoID)]
	return exists, nil
}

func (r *MemoryInteractRepository) HasFavoritedVideo(_ context.Context, userID, videoID int64) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.favorites[pairKey(userID, videoID)]
	return exists, nil
}

func (r *MemoryInteractRepository) UnfavoriteVideo(_ context.Context, userID, videoID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := pairKey(userID, videoID)
	if _, exists := r.favorites[key]; !exists {
		return interactmodel.ErrFavoriteNotFound
	}
	delete(r.favorites, key)
	if video := r.videoRepo.videos[videoID]; video != nil && video.FavoriteCount > 0 {
		video.FavoriteCount--
		video.UpdatedAt = time.Now()
	}
	r.userRepo.adjustFavoriteCount(userID, -1)
	return nil
}

func (r *MemoryInteractRepository) ListFavorites(_ context.Context, userID int64, cursor *time.Time, limit int) ([]interactmodel.FavoriteVideo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]interactmodel.FavoriteVideo, 0)
	for key, favoritedAt := range r.favorites {
		keyUserID, keyVideoID := parsePairKey(key)
		if keyUserID != userID {
			continue
		}
		if cursor != nil && !favoritedAt.Before(*cursor) {
			continue
		}
		video := r.videoRepo.videos[keyVideoID]
		if video == nil {
			continue
		}
		items = append(items, interactmodel.FavoriteVideo{
			VideoID:         video.ID,
			UserID:          video.UserID,
			Title:           video.Title,
			ObjectKey:       video.ObjectKey,
			SourceURL:       video.SourceURL,
			CoverURL:        video.CoverURL,
			DurationMS:      video.DurationMS,
			AllowComment:    video.AllowComment,
			Visibility:      video.Visibility,
			TranscodeStatus: video.TranscodeStatus,
			AuditStatus:     video.AuditStatus,
			LikeCount:       video.LikeCount,
			CommentCount:    video.CommentCount,
			FavoriteCount:   video.FavoriteCount,
			CreatedAt:       video.CreatedAt,
			UpdatedAt:       video.UpdatedAt,
			FavoritedAt:     favoritedAt,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].FavoritedAt.After(items[j].FavoritedAt) })
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r *MemoryInteractRepository) ListUserVideoActions(_ context.Context, userID int64, limit int) ([]interactmodel.UserVideoAction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]interactmodel.UserVideoAction, 0)
	for key, createdAt := range r.videoLikes {
		actionUserID, videoID := parsePairKey(key)
		if actionUserID != userID {
			continue
		}
		items = append(items, interactmodel.UserVideoAction{
			VideoID:    videoID,
			ActionType: "like",
			Weight:     1,
			CreatedAt:  createdAt,
		})
	}
	for key, createdAt := range r.favorites {
		actionUserID, videoID := parsePairKey(key)
		if actionUserID != userID {
			continue
		}
		items = append(items, interactmodel.UserVideoAction{
			VideoID:    videoID,
			ActionType: "favorite",
			Weight:     3,
			CreatedAt:  createdAt,
		})
	}
	for _, comment := range r.comments {
		if comment.UserID != userID || comment.DeletedAt != nil {
			continue
		}
		items = append(items, interactmodel.UserVideoAction{
			VideoID:    comment.VideoID,
			ActionType: "comment",
			Weight:     2,
			CreatedAt:  comment.CreatedAt,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r *MemoryInteractRepository) CreateComment(_ context.Context, comment *interactmodel.Comment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copyComment := *comment
	if copyComment.CreatedAt.IsZero() {
		copyComment.CreatedAt = time.Now()
	}
	r.comments[copyComment.ID] = &copyComment
	if video := r.videoRepo.videos[copyComment.VideoID]; video != nil {
		video.CommentCount++
		video.UpdatedAt = time.Now()
	}
	comment.CreatedAt = copyComment.CreatedAt
	return nil
}

func (r *MemoryInteractRepository) DeleteComment(_ context.Context, userID, commentID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	comment, ok := r.comments[commentID]
	if !ok || comment.UserID != userID {
		return interactmodel.ErrCommentNotFound
	}
	if comment.DeletedAt != nil {
		return nil
	}
	now := time.Now()
	comment.DeletedAt = &now
	if video := r.videoRepo.videos[comment.VideoID]; video != nil && video.CommentCount > 0 {
		video.CommentCount--
		video.UpdatedAt = now
	}
	return nil
}

func (r *MemoryInteractRepository) ListComments(_ context.Context, videoID, rootID int64, cursor *time.Time, limit int) ([]interactmodel.Comment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]interactmodel.Comment, 0)
	if rootID > 0 {
		root, ok := r.comments[rootID]
		if !ok || root.VideoID != videoID || root.DeletedAt != nil {
			return items, nil
		}
	}
	for _, comment := range r.comments {
		if comment.VideoID != videoID || comment.RootID != rootID {
			continue
		}
		if rootID == 0 && comment.ParentID != 0 {
			continue
		}
		if rootID == 0 && comment.DeletedAt != nil {
			continue
		}
		if cursor != nil && !comment.CreatedAt.Before(*cursor) {
			continue
		}
		items = append(items, *comment)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r *MemoryInteractRepository) GetCommentByID(_ context.Context, commentID int64) (*interactmodel.Comment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	comment, ok := r.comments[commentID]
	if !ok {
		return nil, interactmodel.ErrCommentNotFound
	}
	copyComment := *comment
	return &copyComment, nil
}

func (r *MemoryInteractRepository) LikeComment(_ context.Context, userID, commentID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := pairKey(userID, commentID)
	if _, exists := r.commentLikes[key]; exists {
		return interactmodel.ErrAlreadyLiked
	}
	r.commentLikes[key] = time.Now()
	if comment := r.comments[commentID]; comment != nil {
		comment.LikeCount++
	}
	return nil
}

func (r *MemoryInteractRepository) UnlikeComment(_ context.Context, userID, commentID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := pairKey(userID, commentID)
	if _, exists := r.commentLikes[key]; !exists {
		return interactmodel.ErrLikeNotFound
	}
	delete(r.commentLikes, key)
	if comment := r.comments[commentID]; comment != nil && comment.LikeCount > 0 {
		comment.LikeCount--
	}
	return nil
}

func pairKey(left, right int64) string {
	return fmt.Sprintf("%d:%d", left, right)
}

func parsePairKey(value string) (int64, int64) {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	left, _ := strconv.ParseInt(parts[0], 10, 64)
	right, _ := strconv.ParseInt(parts[1], 10, 64)
	return left, right
}
