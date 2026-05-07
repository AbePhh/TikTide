package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/AbePhh/TikTide/backend/internal/app"
	ginmiddleware "github.com/AbePhh/TikTide/backend/internal/http/middleware"
	"github.com/AbePhh/TikTide/backend/internal/http/types"
	interactservice "github.com/AbePhh/TikTide/backend/internal/interact/service"
	"github.com/AbePhh/TikTide/backend/pkg/errno"
)

type InteractHandler struct {
	appCtx *app.Context
}

func NewInteractHandler(appCtx *app.Context) *InteractHandler {
	return &InteractHandler{appCtx: appCtx}
}

func (h *InteractHandler) LikeVideo(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}
	var req types.InteractActionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fail(ctx, errno.ErrInvalidParam)
		return
	}
	if err := h.appCtx.InteractService.ActionLikeVideo(ctx.Request.Context(), authInfo.UserID, interactservice.ActionRequest{
		VideoID:    req.VideoID.Int64(),
		ActionType: req.ActionType,
	}); err != nil {
		fail(ctx, err)
		return
	}
	success(ctx, gin.H{"done": true})
}

func (h *InteractHandler) FavoriteVideo(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}
	var req types.InteractActionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fail(ctx, errno.ErrInvalidParam)
		return
	}
	if err := h.appCtx.InteractService.ActionFavoriteVideo(ctx.Request.Context(), authInfo.UserID, interactservice.ActionRequest{
		VideoID:    req.VideoID.Int64(),
		ActionType: req.ActionType,
	}); err != nil {
		fail(ctx, err)
		return
	}
	success(ctx, gin.H{"done": true})
}

func (h *InteractHandler) ListFavorites(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}
	var cursor *time.Time
	if raw := ctx.Query("cursor"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			fail(ctx, errno.ErrInvalidParam)
			return
		}
		cursor = &parsed
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
	result, err := h.appCtx.InteractService.ListFavorites(ctx.Request.Context(), authInfo.UserID, interactservice.FavoriteListRequest{
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		fail(ctx, err)
		return
	}
	items := make([]types.FavoriteVideoData, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, types.FavoriteVideoData{
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
			CreatedAt:       item.CreatedAt.Format(time.RFC3339),
			UpdatedAt:       item.UpdatedAt.Format(time.RFC3339),
			FavoritedAt:     item.FavoritedAt.Format(time.RFC3339),
		})
	}
	payload := types.FavoriteVideoListData{Items: items}
	if result.NextCursor != nil {
		payload.NextCursor = result.NextCursor.Format(time.RFC3339)
	}
	success(ctx, payload)
}

func (h *InteractHandler) PublishComment(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}
	var req types.CommentPublishRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fail(ctx, errno.ErrInvalidParam)
		return
	}
	result, err := h.appCtx.InteractService.PublishComment(ctx.Request.Context(), authInfo.UserID, interactservice.PublishCommentRequest{
		VideoID:  req.VideoID.Int64(),
		Content:  req.Content,
		ParentID: req.ParentID.Int64(),
		RootID:   req.RootID.Int64(),
		ToUserID: req.ToUserID.Int64(),
	})
	if err != nil {
		fail(ctx, err)
		return
	}
	success(ctx, types.CommentData{
		ID:        result.ID,
		VideoID:   result.VideoID,
		UserID:    result.UserID,
		Content:   result.Content,
		ParentID:  result.ParentID,
		RootID:    result.RootID,
		ToUserID:  result.ToUserID,
		LikeCount: result.LikeCount,
		IsDeleted: false,
		CreatedAt: result.CreatedAt.Format(time.RFC3339),
	})
}

func (h *InteractHandler) ListComments(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}
	videoID, err := strconv.ParseInt(ctx.Query("video_id"), 10, 64)
	if err != nil || videoID <= 0 {
		fail(ctx, errno.ErrInvalidParam)
		return
	}
	var rootID int64
	if raw := ctx.Query("root_id"); raw != "" {
		rootID, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || rootID < 0 {
			fail(ctx, errno.ErrInvalidParam)
			return
		}
	}
	var cursor *time.Time
	if raw := ctx.Query("cursor"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			fail(ctx, errno.ErrInvalidParam)
			return
		}
		cursor = &parsed
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
	result, err := h.appCtx.InteractService.ListComments(ctx.Request.Context(), authInfo.UserID, interactservice.CommentListRequest{
		VideoID: videoID,
		RootID:  rootID,
		Cursor:  cursor,
		Limit:   limit,
	})
	if err != nil {
		fail(ctx, err)
		return
	}
	items := make([]types.CommentData, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, types.CommentData{
			ID:        item.ID,
			VideoID:   item.VideoID,
			UserID:    item.UserID,
			Content:   item.Content,
			ParentID:  item.ParentID,
			RootID:    item.RootID,
			ToUserID:  item.ToUserID,
			LikeCount: item.LikeCount,
			IsDeleted: item.IsDeleted,
			CreatedAt: item.CreatedAt.Format(time.RFC3339),
		})
	}
	payload := types.CommentListData{Items: items}
	if result.NextCursor != nil {
		payload.NextCursor = result.NextCursor.Format(time.RFC3339)
	}
	success(ctx, payload)
}

func (h *InteractHandler) DeleteComment(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}
	var req types.CommentDeleteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fail(ctx, errno.ErrInvalidParam)
		return
	}
	if err := h.appCtx.InteractService.DeleteComment(ctx.Request.Context(), authInfo.UserID, req.CommentID.Int64()); err != nil {
		return
	}
	success(ctx, gin.H{"done": true})
}

func (h *InteractHandler) LikeComment(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}
	var req types.CommentLikeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fail(ctx, errno.ErrInvalidParam)
		return
	}
	if err := h.appCtx.InteractService.ActionLikeComment(ctx.Request.Context(), authInfo.UserID, interactservice.CommentLikeRequest{
		CommentID:  req.CommentID.Int64(),
		ActionType: req.ActionType,
	}); err != nil {
		fail(ctx, err)
		return
	}
	success(ctx, gin.H{"done": true})
}
