package model

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type messageRecord struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement"`
	ReceiverID int64     `gorm:"column:receiver_id"`
	SenderID   int64     `gorm:"column:sender_id"`
	Type       int8      `gorm:"column:type"`
	RelatedID  int64     `gorm:"column:related_id"`
	Content    string    `gorm:"column:content"`
	IsRead     int8      `gorm:"column:is_read"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (messageRecord) TableName() string { return "t_message" }

type MySQLRepository struct {
	db *gorm.DB
}

func NewMySQLRepository(db *gorm.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) Create(ctx context.Context, message *Message) error {
	record := messageRecord{
		ReceiverID: message.ReceiverID,
		SenderID:   message.SenderID,
		Type:       message.Type,
		RelatedID:  message.RelatedID,
		Content:    message.Content,
		IsRead:     message.IsRead,
	}

	if err := r.db.WithContext(ctx).Table("t_message").Create(&record).Error; err != nil {
		return fmt.Errorf("create message: %w", err)
	}

	message.ID = record.ID
	message.CreatedAt = record.CreatedAt
	return nil
}

func (r *MySQLRepository) List(ctx context.Context, receiverID int64, messageType *int8, cursor int64, limit int) ([]Message, error) {
	query := r.db.WithContext(ctx).
		Table("t_message").
		Where("receiver_id = ?", receiverID)

	if messageType != nil {
		query = query.Where("type = ?", *messageType)
	}
	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}

	var records []messageRecord
	if err := query.Order("id DESC").Limit(limit).Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}

	items := make([]Message, 0, len(records))
	for _, record := range records {
		items = append(items, Message{
			ID:         record.ID,
			ReceiverID: record.ReceiverID,
			SenderID:   record.SenderID,
			Type:       record.Type,
			RelatedID:  record.RelatedID,
			Content:    record.Content,
			IsRead:     record.IsRead,
			CreatedAt:  record.CreatedAt,
		})
	}
	return items, nil
}

func (r *MySQLRepository) MarkRead(ctx context.Context, receiverID, messageID int64) (*Message, error) {
	var existing messageRecord
	if err := r.db.WithContext(ctx).
		Table("t_message").
		Where("id = ? AND receiver_id = ?", messageID, receiverID).
		First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMessageNotFound
		}
		return nil, fmt.Errorf("query message before mark read: %w", err)
	}

	result := r.db.WithContext(ctx).
		Table("t_message").
		Where("id = ? AND receiver_id = ? AND is_read = 0", messageID, receiverID).
		Update("is_read", 1)
	if result.Error != nil {
		return nil, fmt.Errorf("mark message read: %w", result.Error)
	}

	return &Message{
		ID:         existing.ID,
		ReceiverID: existing.ReceiverID,
		SenderID:   existing.SenderID,
		Type:       existing.Type,
		RelatedID:  existing.RelatedID,
		Content:    existing.Content,
		IsRead:     1,
		CreatedAt:  existing.CreatedAt,
	}, nil
}

func (r *MySQLRepository) MarkAllReadByType(ctx context.Context, receiverID int64, messageType int8) error {
	if err := r.db.WithContext(ctx).
		Table("t_message").
		Where("receiver_id = ? AND type = ? AND is_read = 0", receiverID, messageType).
		Update("is_read", 1).Error; err != nil {
		return fmt.Errorf("mark all messages read by type: %w", err)
	}
	return nil
}
