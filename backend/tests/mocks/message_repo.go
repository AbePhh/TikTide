package mocks

import (
	"context"
	"sort"
	"sync"
	"time"

	messagemodel "github.com/AbePhh/TikTide/backend/internal/message/model"
)

// MemoryMessageRepository 提供消息模块的内存仓储，便于 HTTP/服务测试。
type MemoryMessageRepository struct {
	mu       sync.RWMutex
	nextID   int64
	messages map[int64]*messagemodel.Message
}

func NewMemoryMessageRepository() *MemoryMessageRepository {
	return &MemoryMessageRepository{
		nextID:   1,
		messages: make(map[int64]*messagemodel.Message),
	}
}

func (r *MemoryMessageRepository) Create(_ context.Context, message *messagemodel.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	copyMessage := *message
	copyMessage.ID = r.nextID
	r.nextID++
	if copyMessage.CreatedAt.IsZero() {
		copyMessage.CreatedAt = time.Now()
	}
	r.messages[copyMessage.ID] = &copyMessage

	message.ID = copyMessage.ID
	message.CreatedAt = copyMessage.CreatedAt
	return nil
}

func (r *MemoryMessageRepository) List(_ context.Context, receiverID int64, messageType *int8, cursor int64, limit int) ([]messagemodel.Message, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]messagemodel.Message, 0)
	for _, message := range r.messages {
		if message.ReceiverID != receiverID {
			continue
		}
		if messageType != nil && message.Type != *messageType {
			continue
		}
		if cursor > 0 && message.ID >= cursor {
			continue
		}
		items = append(items, *message)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID > items[j].ID
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r *MemoryMessageRepository) MarkRead(_ context.Context, receiverID, messageID int64) (*messagemodel.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	message, ok := r.messages[messageID]
	if !ok || message.ReceiverID != receiverID {
		return nil, messagemodel.ErrMessageNotFound
	}
	message.IsRead = 1
	copyMessage := *message
	return &copyMessage, nil
}

func (r *MemoryMessageRepository) MarkAllReadByType(_ context.Context, receiverID int64, messageType int8) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, message := range r.messages {
		if message.ReceiverID == receiverID && message.Type == messageType {
			message.IsRead = 1
		}
	}
	return nil
}
