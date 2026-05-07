package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/AbePhh/TikTide/backend/internal/app"
	ginmiddleware "github.com/AbePhh/TikTide/backend/internal/http/middleware"
	"github.com/AbePhh/TikTide/backend/internal/http/types"
	messageservice "github.com/AbePhh/TikTide/backend/internal/message/service"
	"github.com/AbePhh/TikTide/backend/pkg/errno"
)

type MessageHandler struct {
	appCtx *app.Context
}

func NewMessageHandler(appCtx *app.Context) *MessageHandler {
	return &MessageHandler{appCtx: appCtx}
}

func (h *MessageHandler) GetUnreadCount(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	result, err := h.appCtx.MessageService.GetUnreadCount(ctx.Request.Context(), authInfo.UserID)
	if err != nil {
		fail(ctx, err)
		return
	}

	success(ctx, types.MessageUnreadCountData{Items: result})
}

func (h *MessageHandler) List(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	var messageType *int8
	if raw := ctx.Query("type"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 8)
		if err != nil {
			fail(ctx, errno.ErrInvalidParam)
			return
		}
		value := int8(parsed)
		messageType = &value
	}

	var cursor int64
	if raw := ctx.Query("cursor"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			fail(ctx, errno.ErrInvalidParam)
			return
		}
		cursor = parsed
	}

	limit := 20
	if raw := ctx.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			fail(ctx, errno.ErrInvalidParam)
			return
		}
		limit = parsed
	}

	result, err := h.appCtx.MessageService.ListMessages(ctx.Request.Context(), authInfo.UserID, messageservice.ListRequest{
		Type:   messageType,
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		fail(ctx, err)
		return
	}

	items := make([]types.MessageData, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, types.MessageData{
			ID:         item.ID,
			ReceiverID: item.ReceiverID,
			SenderID:   item.SenderID,
			Type:       item.Type,
			RelatedID:  item.RelatedID,
			Content:    item.Content,
			IsRead:     item.IsRead,
			CreatedAt:  item.CreatedAt,
		})
	}

	success(ctx, types.MessageListData{
		Items:      items,
		NextCursor: result.NextCursor,
	})
}

func (h *MessageHandler) Read(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	var req types.MessageReadRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fail(ctx, errno.ErrInvalidParam)
		return
	}

	var messageID *int64
	if req.MsgID != nil {
		value := req.MsgID.Int64()
		messageID = &value
	}

	err := h.appCtx.MessageService.MarkRead(ctx.Request.Context(), authInfo.UserID, messageservice.MarkReadRequest{
		MessageID: messageID,
		Type:      req.Type,
	})
	if err != nil {
		fail(ctx, err)
		return
	}

	success(ctx, gin.H{"read": true})
}
