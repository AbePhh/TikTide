package service

import (
	"context"
	"strings"
	"time"

	messagemodel "github.com/AbePhh/TikTide/backend/internal/message/model"
	messageservice "github.com/AbePhh/TikTide/backend/internal/message/service"
	usermodel "github.com/AbePhh/TikTide/backend/internal/user/model"
	videomodel "github.com/AbePhh/TikTide/backend/internal/video/model"
	"github.com/AbePhh/TikTide/backend/pkg/errno"
	"github.com/AbePhh/TikTide/backend/pkg/utils"

	interactmodel "github.com/AbePhh/TikTide/backend/internal/interact/model"
)

type UserRepository interface {
	GetByID(ctx context.Context, userID int64) (*usermodel.User, error)
}

type VideoRepository interface {
	GetVideoByID(ctx context.Context, videoID int64) (*videomodel.Video, error)
}

type InteractService interface {
	ActionLikeVideo(ctx context.Context, userID int64, req ActionRequest) error
	ActionFavoriteVideo(ctx context.Context, userID int64, req ActionRequest) error
	ListFavorites(ctx context.Context, userID int64, req FavoriteListRequest) (*FavoriteListResult, error)
	PublishComment(ctx context.Context, userID int64, req PublishCommentRequest) (*CommentResult, error)
	DeleteComment(ctx context.Context, userID, commentID int64) error
	ListComments(ctx context.Context, userID int64, req CommentListRequest) (*CommentListResult, error)
	ActionLikeComment(ctx context.Context, userID int64, req CommentLikeRequest) error
}

type Service struct {
	repo       interactmodel.Repository
	userRepo   UserRepository
	videoRepo  VideoRepository
	messageSvc messageservice.MessageService
	ids        utils.IDGenerator
}

type ActionRequest struct {
	VideoID    int64
	ActionType int8
}

type FavoriteListRequest struct {
	Cursor *time.Time
	Limit  int
}

type FavoriteItem struct {
	VideoID         int64
	UserID          int64
	Title           string
	ObjectKey       string
	SourceURL       string
	CoverURL        string
	DurationMS      int32
	AllowComment    int8
	Visibility      int8
	TranscodeStatus int8
	AuditStatus     int8
	LikeCount       int64
	CommentCount    int64
	FavoriteCount   int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
	FavoritedAt     time.Time
}

type FavoriteListResult struct {
	Items      []FavoriteItem
	NextCursor *time.Time
}

type PublishCommentRequest struct {
	VideoID   int64
	Content   string
	ParentID  int64
	RootID    int64
	ToUserID  int64
}

type CommentResult struct {
	ID        int64
	VideoID   int64
	UserID    int64
	Content   string
	ParentID  int64
	RootID    int64
	ToUserID  int64
	LikeCount int64
	CreatedAt time.Time
}

type CommentListRequest struct {
	VideoID int64
	RootID  int64
	Cursor  *time.Time
	Limit   int
}

type CommentItem struct {
	ID        int64
	VideoID   int64
	UserID    int64
	Content   string
	ParentID  int64
	RootID    int64
	ToUserID  int64
	LikeCount int64
	IsDeleted bool
	CreatedAt time.Time
}

type CommentListResult struct {
	Items      []CommentItem
	NextCursor *time.Time
}

type CommentLikeRequest struct {
	CommentID  int64
	ActionType int8
}

func New(repo interactmodel.Repository, userRepo UserRepository, videoRepo VideoRepository, messageSvc messageservice.MessageService, ids utils.IDGenerator) *Service {
	return &Service{
		repo:       repo,
		userRepo:   userRepo,
		videoRepo:  videoRepo,
		messageSvc: messageSvc,
		ids:        ids,
	}
}

func (s *Service) ActionLikeVideo(ctx context.Context, userID int64, req ActionRequest) error {
	if userID <= 0 || req.VideoID <= 0 || !isValidAction(req.ActionType) {
		return errno.ErrInvalidParam
	}
	video, err := s.ensureVisibleVideo(ctx, req.VideoID)
	if err != nil {
		return err
	}
	switch req.ActionType {
	case interactmodel.ActionLike:
		if err := s.repo.LikeVideo(ctx, userID, req.VideoID, video.UserID); err != nil {
			if err == interactmodel.ErrAlreadyLiked {
				return errno.ErrDuplicateLike
			}
			return errno.ErrInternalRPC
		}
		s.notifyLikeVideo(ctx, userID, video.UserID, req.VideoID)
	case interactmodel.ActionCancel:
		if err := s.repo.UnlikeVideo(ctx, userID, req.VideoID, video.UserID); err != nil {
			if err == interactmodel.ErrLikeNotFound {
				return nil
			}
			return errno.ErrInternalRPC
		}
	}
	return nil
}

func (s *Service) ActionFavoriteVideo(ctx context.Context, userID int64, req ActionRequest) error {
	if userID <= 0 || req.VideoID <= 0 || !isValidAction(req.ActionType) {
		return errno.ErrInvalidParam
	}
	if _, err := s.ensureVisibleVideo(ctx, req.VideoID); err != nil {
		return err
	}
	switch req.ActionType {
	case interactmodel.ActionLike:
		if err := s.repo.FavoriteVideo(ctx, userID, req.VideoID); err != nil {
			if err == interactmodel.ErrAlreadyFavorited {
				return errno.ErrDuplicateFavorite
			}
			return errno.ErrInternalRPC
		}
	case interactmodel.ActionCancel:
		if err := s.repo.UnfavoriteVideo(ctx, userID, req.VideoID); err != nil {
			if err == interactmodel.ErrFavoriteNotFound {
				return nil
			}
			return errno.ErrInternalRPC
		}
	}
	return nil
}

func (s *Service) ListFavorites(ctx context.Context, userID int64, req FavoriteListRequest) (*FavoriteListResult, error) {
	if userID <= 0 {
		return nil, errno.ErrInvalidParam
	}
	limit := normalizeLimit(req.Limit)
	items, err := s.repo.ListFavorites(ctx, userID, req.Cursor, limit)
	if err != nil {
		return nil, errno.ErrInternalRPC
	}
	result := make([]FavoriteItem, 0, len(items))
	for _, item := range items {
		result = append(result, FavoriteItem{
			VideoID:         item.VideoID,
			UserID:          item.UserID,
			Title:           item.Title,
			ObjectKey:       item.ObjectKey,
			SourceURL:       item.SourceURL,
			CoverURL:        item.CoverURL,
			DurationMS:      item.DurationMS,
			AllowComment:    item.AllowComment,
			Visibility:      item.Visibility,
			TranscodeStatus: item.TranscodeStatus,
			AuditStatus:     item.AuditStatus,
			LikeCount:       item.LikeCount,
			CommentCount:    item.CommentCount,
			FavoriteCount:   item.FavoriteCount,
			CreatedAt:       item.CreatedAt,
			UpdatedAt:       item.UpdatedAt,
			FavoritedAt:     item.FavoritedAt,
		})
	}
	var nextCursor *time.Time
	if len(items) == limit {
		last := items[len(items)-1].FavoritedAt
		nextCursor = &last
	}
	return &FavoriteListResult{Items: result, NextCursor: nextCursor}, nil
}

func (s *Service) PublishComment(ctx context.Context, userID int64, req PublishCommentRequest) (*CommentResult, error) {
	if userID <= 0 || req.VideoID <= 0 {
		return nil, errno.ErrInvalidParam
	}
	content := strings.TrimSpace(req.Content)
	if content == "" || len([]rune(content)) > 500 {
		return nil, errno.ErrInvalidComment
	}
	video, err := s.ensureVisibleVideo(ctx, req.VideoID)
	if err != nil {
		return nil, err
	}
	if video.AllowComment != 1 {
		return nil, errno.ErrCommentForbidden
	}

	parentID := req.ParentID
	rootID := req.RootID
	toUserID := req.ToUserID
	if parentID > 0 {
		parent, err := s.repo.GetCommentByID(ctx, parentID)
		if err != nil {
			if err == interactmodel.ErrCommentNotFound {
				return nil, errno.ErrCommentNotFound
			}
			return nil, errno.ErrInternalRPC
		}
		if parent.DeletedAt != nil {
			return nil, errno.ErrCommentDeleted
		}
		if parent.VideoID != req.VideoID {
			return nil, errno.ErrInvalidParam
		}
		if rootID == 0 {
			if parent.RootID > 0 {
				rootID = parent.RootID
			} else {
				rootID = parent.ID
			}
		}
		if toUserID == 0 {
			toUserID = parent.UserID
		}
	} else {
		rootID = 0
	}

	comment := &interactmodel.Comment{
		ID:       s.ids.NewID(),
		VideoID:  req.VideoID,
		UserID:   userID,
		Content:  content,
		ParentID: parentID,
		RootID:   rootID,
		ToUserID: toUserID,
	}
	if err := s.repo.CreateComment(ctx, comment); err != nil {
		return nil, errno.ErrInternalRPC
	}

	if parentID > 0 {
		s.notifyReplyComment(ctx, userID, toUserID, comment.ID, content)
	} else {
		s.notifyCommentVideo(ctx, userID, video.UserID, req.VideoID, content)
	}

	return &CommentResult{
		ID:        comment.ID,
		VideoID:   comment.VideoID,
		UserID:    comment.UserID,
		Content:   comment.Content,
		ParentID:  comment.ParentID,
		RootID:    comment.RootID,
		ToUserID:  comment.ToUserID,
		LikeCount: comment.LikeCount,
		CreatedAt: comment.CreatedAt,
	}, nil
}

func (s *Service) ListComments(ctx context.Context, userID int64, req CommentListRequest) (*CommentListResult, error) {
	if userID <= 0 || req.VideoID <= 0 {
		return nil, errno.ErrInvalidParam
	}
	if _, err := s.ensureVisibleVideo(ctx, req.VideoID); err != nil {
		return nil, err
	}
	limit := normalizeLimit(req.Limit)
	items, err := s.repo.ListComments(ctx, req.VideoID, req.RootID, req.Cursor, limit)
	if err != nil {
		return nil, errno.ErrInternalRPC
	}
	result := make([]CommentItem, 0, len(items))
	for _, item := range items {
		result = append(result, CommentItem{
			ID:        item.ID,
			VideoID:   item.VideoID,
			UserID:    item.UserID,
			Content:   item.Content,
			ParentID:  item.ParentID,
			RootID:    item.RootID,
			ToUserID:  item.ToUserID,
			LikeCount: item.LikeCount,
			IsDeleted: item.DeletedAt != nil,
			CreatedAt: item.CreatedAt,
		})
	}
	var nextCursor *time.Time
	if len(items) == limit {
		last := items[len(items)-1].CreatedAt
		nextCursor = &last
	}
	return &CommentListResult{Items: result, NextCursor: nextCursor}, nil
}

func (s *Service) DeleteComment(ctx context.Context, userID, commentID int64) error {
	if userID <= 0 || commentID <= 0 {
		return errno.ErrInvalidParam
	}
	comment, err := s.repo.GetCommentByID(ctx, commentID)
	if err != nil {
		if err == interactmodel.ErrCommentNotFound {
			return errno.ErrCommentNotFound
		}
		return errno.ErrInternalRPC
	}
	if comment.DeletedAt != nil {
		return nil
	}
	if comment.UserID != userID {
		return errno.ErrUnauthorized
	}
	if err := s.repo.DeleteComment(ctx, userID, commentID); err != nil {
		if err == interactmodel.ErrCommentNotFound {
			return errno.ErrCommentNotFound
		}
		return errno.ErrInternalRPC
	}
	return nil
}

func (s *Service) ActionLikeComment(ctx context.Context, userID int64, req CommentLikeRequest) error {
	if userID <= 0 || req.CommentID <= 0 || !isValidAction(req.ActionType) {
		return errno.ErrInvalidParam
	}
	comment, err := s.repo.GetCommentByID(ctx, req.CommentID)
	if err != nil {
		if err == interactmodel.ErrCommentNotFound {
			return errno.ErrCommentNotFound
		}
		return errno.ErrInternalRPC
	}
	if comment.DeletedAt != nil {
		return errno.ErrCommentDeleted
	}
	switch req.ActionType {
	case interactmodel.ActionLike:
		if err := s.repo.LikeComment(ctx, userID, req.CommentID); err != nil {
			if err == interactmodel.ErrAlreadyLiked {
				return errno.ErrDuplicateLike
			}
			return errno.ErrInternalRPC
		}
	case interactmodel.ActionCancel:
		if err := s.repo.UnlikeComment(ctx, userID, req.CommentID); err != nil {
			if err == interactmodel.ErrLikeNotFound {
				return nil
			}
			return errno.ErrInternalRPC
		}
	}
	return nil
}

func (s *Service) ensureVisibleVideo(ctx context.Context, videoID int64) (*videomodel.Video, error) {
	video, err := s.videoRepo.GetVideoByID(ctx, videoID)
	if err != nil {
		if err == videomodel.ErrVideoNotFound {
			return nil, errno.ErrResourceNotFound
		}
		return nil, errno.ErrInternalRPC
	}
	if video.Visibility != videomodel.VisibilityPublic || video.AuditStatus != videomodel.AuditPassed {
		return nil, errno.ErrVideoInvisible
	}
	if video.TranscodeStatus == videomodel.TranscodeFailed {
		return nil, errno.ErrVideoTranscodeFailed
	}
	if video.TranscodeStatus != videomodel.TranscodeSuccess {
		return nil, errno.ErrVideoTranscoding
	}
	return video, nil
}

func (s *Service) notifyLikeVideo(ctx context.Context, senderID, receiverID, videoID int64) {
	if s.messageSvc == nil || receiverID <= 0 || receiverID == senderID {
		return
	}
	_ = s.messageSvc.CreateMessage(ctx, messageservice.CreateMessageRequest{
		ReceiverID: receiverID,
		SenderID:   senderID,
		Type:       messagemodel.MessageTypeLikeVideo,
		RelatedID:  videoID,
		Content:    "你的视频收到了一个赞",
	})
}

func (s *Service) notifyCommentVideo(ctx context.Context, senderID, receiverID, videoID int64, content string) {
	if s.messageSvc == nil || receiverID <= 0 || receiverID == senderID {
		return
	}
	_ = s.messageSvc.CreateMessage(ctx, messageservice.CreateMessageRequest{
		ReceiverID: receiverID,
		SenderID:   senderID,
		Type:       messagemodel.MessageTypeCommentVideo,
		RelatedID:  videoID,
		Content:    "你的视频收到新评论: " + truncateContent(content),
	})
}

func (s *Service) notifyReplyComment(ctx context.Context, senderID, receiverID, commentID int64, content string) {
	if s.messageSvc == nil || receiverID <= 0 || receiverID == senderID {
		return
	}
	_ = s.messageSvc.CreateMessage(ctx, messageservice.CreateMessageRequest{
		ReceiverID: receiverID,
		SenderID:   senderID,
		Type:       messagemodel.MessageTypeReplyComment,
		RelatedID:  commentID,
		Content:    "你收到一条评论回复: " + truncateContent(content),
	})
}

func truncateContent(content string) string {
	runes := []rune(strings.TrimSpace(content))
	if len(runes) <= 50 {
		return string(runes)
	}
	return string(runes[:50])
}

func isValidAction(actionType int8) bool {
	return actionType == interactmodel.ActionLike || actionType == interactmodel.ActionCancel
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 50 {
		return 50
	}
	return limit
}
