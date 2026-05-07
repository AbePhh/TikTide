package model

import (
	"context"
	"errors"
	"time"
)

var ErrMessageNotFound = errors.New("message not found")

const (
	MessageTypeLikeVideo          int8 = 1
	MessageTypeCommentVideo       int8 = 2
	MessageTypeReplyComment       int8 = 3
	MessageTypeNewFollower        int8 = 4
	MessageTypeSystemNotification int8 = 5
	MessageTypeVideoProcessResult int8 = 6
)

type Message struct {
	ID         int64
	ReceiverID int64
	SenderID   int64
	Type       int8
	RelatedID  int64
	Content    string
	IsRead     int8
	CreatedAt  time.Time
}

type Repository interface {
	Create(ctx context.Context, message *Message) error
	List(ctx context.Context, receiverID int64, messageType *int8, cursor int64, limit int) ([]Message, error)
	MarkRead(ctx context.Context, receiverID, messageID int64) (*Message, error)
	MarkAllReadByType(ctx context.Context, receiverID int64, messageType int8) error
}
