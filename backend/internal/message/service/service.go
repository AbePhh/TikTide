package service

import (
	"context"
	"strconv"

	"github.com/redis/go-redis/v9"

	"github.com/AbePhh/TikTide/backend/internal/message/model"
	"github.com/AbePhh/TikTide/backend/pkg/errno"
	"github.com/AbePhh/TikTide/backend/pkg/rediskey"
)

type MessageService interface {
	CreateMessage(ctx context.Context, req CreateMessageRequest) error
	CreateVideoProcessResult(ctx context.Context, receiverID, relatedID int64, content string) error
	GetUnreadCount(ctx context.Context, userID int64) (map[string]int64, error)
	ListMessages(ctx context.Context, userID int64, req ListRequest) (*ListResult, error)
	MarkRead(ctx context.Context, userID int64, req MarkReadRequest) error
}

type Service struct {
	repo  model.Repository
	redis *redis.Client
}

type ListRequest struct {
	Type   *int8
	Cursor int64
	Limit  int
}

type MessageItem struct {
	ID         int64
	ReceiverID int64
	SenderID   int64
	Type       int8
	RelatedID  int64
	Content    string
	IsRead     int8
	CreatedAt  string
}

type ListResult struct {
	Items      []MessageItem
	NextCursor string
}

type MarkReadRequest struct {
	MessageID *int64
	Type      *int8
}

type CreateMessageRequest struct {
	ReceiverID int64
	SenderID   int64
	Type       int8
	RelatedID  int64
	Content    string
}

func New(repo model.Repository, redisClient *redis.Client) *Service {
	return &Service{
		repo:  repo,
		redis: redisClient,
	}
}

func (s *Service) CreateMessage(ctx context.Context, req CreateMessageRequest) error {
	if req.ReceiverID <= 0 || req.Type <= 0 || req.RelatedID <= 0 || req.Content == "" {
		return errno.ErrInvalidParam
	}
	if s.repo == nil {
		return errno.ErrInternalRPC
	}

	message := &model.Message{
		ReceiverID: req.ReceiverID,
		SenderID:   req.SenderID,
		Type:       req.Type,
		RelatedID:  req.RelatedID,
		Content:    req.Content,
		IsRead:     0,
	}
	if err := s.repo.Create(ctx, message); err != nil {
		return errno.ErrInternalRPC
	}

	if s.redis != nil {
		if err := s.redis.HIncrBy(ctx, rediskey.MessageUnread(req.ReceiverID), strconv.FormatInt(int64(req.Type), 10), 1).Err(); err != nil {
			return errno.ErrInternalRPC
		}
	}

	return nil
}

func (s *Service) CreateVideoProcessResult(ctx context.Context, receiverID, relatedID int64, content string) error {
	return s.CreateMessage(ctx, CreateMessageRequest{
		ReceiverID: receiverID,
		SenderID:   0,
		Type:       model.MessageTypeVideoProcessResult,
		RelatedID:  relatedID,
		Content:    content,
	})
}

func (s *Service) GetUnreadCount(ctx context.Context, userID int64) (map[string]int64, error) {
	if userID <= 0 {
		return nil, errno.ErrInvalidParam
	}

	result := map[string]int64{}
	if s.redis == nil {
		return result, nil
	}

	values, err := s.redis.HGetAll(ctx, rediskey.MessageUnread(userID)).Result()
	if err != nil {
		return nil, errno.ErrInternalRPC
	}
	for key, raw := range values {
		count, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil {
			continue
		}
		result[key] = count
	}
	return result, nil
}

func (s *Service) ListMessages(ctx context.Context, userID int64, req ListRequest) (*ListResult, error) {
	if userID <= 0 {
		return nil, errno.ErrInvalidParam
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	messages, err := s.repo.List(ctx, userID, req.Type, req.Cursor, limit)
	if err != nil {
		return nil, errno.ErrInternalRPC
	}

	items := make([]MessageItem, 0, len(messages))
	for _, message := range messages {
		items = append(items, MessageItem{
			ID:         message.ID,
			ReceiverID: message.ReceiverID,
			SenderID:   message.SenderID,
			Type:       message.Type,
			RelatedID:  message.RelatedID,
			Content:    message.Content,
			IsRead:     message.IsRead,
			CreatedAt:  message.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	nextCursor := ""
	if len(messages) == limit {
		nextCursor = strconv.FormatInt(messages[len(messages)-1].ID, 10)
	}

	return &ListResult{
		Items:      items,
		NextCursor: nextCursor,
	}, nil
}

func (s *Service) MarkRead(ctx context.Context, userID int64, req MarkReadRequest) error {
	if userID <= 0 {
		return errno.ErrInvalidParam
	}

	switch {
	case req.MessageID != nil && *req.MessageID > 0:
		message, err := s.repo.MarkRead(ctx, userID, *req.MessageID)
		if err != nil {
			if err == model.ErrMessageNotFound {
				return errno.ErrResourceNotFound
			}
			return errno.ErrInternalRPC
		}
		if s.redis != nil {
			field := strconv.FormatInt(int64(message.Type), 10)
			count, err := s.redis.HIncrBy(ctx, rediskey.MessageUnread(userID), field, -1).Result()
			if err == nil && count < 0 {
				_ = s.redis.HSet(ctx, rediskey.MessageUnread(userID), field, 0).Err()
			}
		}
		return nil
	case req.Type != nil:
		if err := s.repo.MarkAllReadByType(ctx, userID, *req.Type); err != nil {
			return errno.ErrInternalRPC
		}
		if s.redis != nil {
			_ = s.redis.HSet(ctx, rediskey.MessageUnread(userID), strconv.FormatInt(int64(*req.Type), 10), 0).Err()
		}
		return nil
	default:
		return errno.ErrInvalidParam
	}
}
