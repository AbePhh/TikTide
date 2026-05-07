package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/AbePhh/TikTide/backend/internal/app"
	ginmiddleware "github.com/AbePhh/TikTide/backend/internal/http/middleware"
	"github.com/AbePhh/TikTide/backend/internal/http/types"
	relationservice "github.com/AbePhh/TikTide/backend/internal/relation/service"
	"github.com/AbePhh/TikTide/backend/pkg/errno"
)

// RelationHandler 处理关注关系相关的 HTTP 请求。
type RelationHandler struct {
	appCtx *app.Context
}

// NewRelationHandler 创建关注关系处理器。
func NewRelationHandler(appCtx *app.Context) *RelationHandler {
	return &RelationHandler{appCtx: appCtx}
}

// Action 处理关注或取关操作。
func (h *RelationHandler) Action(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	var req types.RelationActionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fail(ctx, errno.ErrInvalidParam)
		return
	}

	result, err := h.appCtx.RelationService.Action(ctx.Request.Context(), authInfo.UserID, relationservice.ActionRequest{
		ToUserID:   req.ToUserID.Int64(),
		ActionType: req.ActionType,
	})
	if err != nil {
		fail(ctx, err)
		return
	}

	response := types.RelationActionData{
		ToUserID:   result.ToUserID,
		ActionType: result.ActionType,
		IsFollowed: result.IsFollowed,
		IsMutual:   result.IsMutual,
	}
	if result.FollowedAt != nil {
		response.FollowedAt = result.FollowedAt.Format(time.RFC3339)
	}

	success(ctx, response)
}

// ListFollowing 返回指定用户的关注列表。
func (h *RelationHandler) ListFollowing(ctx *gin.Context) {
	h.listUsers(ctx, true)
}

// ListFollowers 返回指定用户的粉丝列表。
func (h *RelationHandler) ListFollowers(ctx *gin.Context) {
	h.listUsers(ctx, false)
}

func (h *RelationHandler) listUsers(ctx *gin.Context, following bool) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	targetUserID, err := strconv.ParseInt(ctx.Param("uid"), 10, 64)
	if err != nil || targetUserID <= 0 {
		fail(ctx, errno.ErrInvalidParam)
		return
	}

	var cursor int64
	if cursorRaw := ctx.Query("cursor"); cursorRaw != "" {
		cursor, err = strconv.ParseInt(cursorRaw, 10, 64)
		if err != nil || cursor <= 0 {
			fail(ctx, errno.ErrInvalidParam)
			return
		}
	}

	limit := 20
	if limitRaw := ctx.Query("limit"); limitRaw != "" {
		limit, err = strconv.Atoi(limitRaw)
		if err != nil {
			fail(ctx, errno.ErrInvalidParam)
			return
		}
	}

	req := relationservice.ListRequest{
		Cursor: cursor,
		Limit:  limit,
	}

	var result *relationservice.UserListResult
	if following {
		result, err = h.appCtx.RelationService.ListFollowing(ctx.Request.Context(), authInfo.UserID, targetUserID, req)
	} else {
		result, err = h.appCtx.RelationService.ListFollowers(ctx.Request.Context(), authInfo.UserID, targetUserID, req)
	}
	if err != nil {
		fail(ctx, err)
		return
	}

	items := make([]types.RelationUserData, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, types.RelationUserData{
			ID:            item.ID,
			Username:      item.Username,
			Nickname:      item.Nickname,
			AvatarURL:     item.AvatarURL,
			Signature:     item.Signature,
			Gender:        item.Gender,
			Status:        item.Status,
			FollowCount:   item.FollowCount,
			FollowerCount: item.FollowerCount,
			IsFollowed:    item.IsFollowed,
			IsMutual:      item.IsMutual,
			CreatedAt:     item.CreatedAt.Format(time.RFC3339),
		})
	}

	success(ctx, types.RelationUserListData{
		Items:      items,
		NextCursor: result.NextCursor,
	})
}
