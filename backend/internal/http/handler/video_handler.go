package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/AbePhh/TikTide/backend/internal/app"
	ginmiddleware "github.com/AbePhh/TikTide/backend/internal/http/middleware"
	"github.com/AbePhh/TikTide/backend/internal/http/types"
	videoservice "github.com/AbePhh/TikTide/backend/internal/video/service"
	"github.com/AbePhh/TikTide/backend/pkg/errno"
)

// VideoHandler 处理视频发布与 OSS 直传相关 HTTP 请求。
type VideoHandler struct {
	appCtx *app.Context
}

// NewVideoHandler 创建视频处理器。
func NewVideoHandler(appCtx *app.Context) *VideoHandler {
	return &VideoHandler{appCtx: appCtx}
}

// CreateUploadCredential 生成阿里云 OSS 直传凭证。
func (h *VideoHandler) CreateUploadCredential(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	var req types.UploadCredentialRequest
	if ctx.Request.ContentLength > 0 {
		if err := ctx.ShouldBindJSON(&req); err != nil {
			fail(ctx, errno.ErrInvalidParam)
			return
		}
	}

	result, err := h.appCtx.VideoService.CreateUploadCredential(ctx.Request.Context(), authInfo.UserID, videoservice.CreateUploadCredentialRequest{
		FileName: req.FileName,
	})
	if err != nil {
		fail(ctx, err)
		return
	}

	success(ctx, types.UploadCredentialData{
		ObjectKey:    result.ObjectKey,
		UploadURL:    result.UploadURL,
		UploadMethod: result.UploadMethod,
		ExpiresAt:    result.ExpiresAt.Format(time.RFC3339),
	})
}

// PublishVideo 发布视频元数据。
func (h *VideoHandler) PublishVideo(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	var req types.PublishVideoRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fail(ctx, errno.ErrInvalidParam)
		return
	}

	result, err := h.appCtx.VideoService.PublishVideo(ctx.Request.Context(), authInfo.UserID, videoservice.PublishVideoRequest{
		ObjectKey:    req.ObjectKey,
		Title:        req.Title,
		HashtagIDs:   req.HashtagIDs,
		HashtagNames: req.HashtagNames,
		AllowComment: req.AllowComment,
		Visibility:   req.Visibility,
	})
	if err != nil {
		fail(ctx, err)
		return
	}

	success(ctx, types.PublishVideoData{
		VideoID:         result.VideoID,
		ObjectKey:       result.ObjectKey,
		SourceURL:       result.SourceURL,
		TranscodeStatus: result.TranscodeStatus,
	})
}

// CreateHashtag 创建话题，若已存在则返回已有话题。
func (h *VideoHandler) CreateHashtag(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	var req types.CreateHashtagRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fail(ctx, errno.ErrInvalidParam)
		return
	}

	result, err := h.appCtx.VideoService.CreateHashtag(ctx.Request.Context(), authInfo.UserID, videoservice.CreateHashtagRequest{
		Name: req.Name,
	})
	if err != nil {
		fail(ctx, err)
		return
	}

	success(ctx, types.HashtagData{
		ID:        result.ID,
		Name:      result.Name,
		UseCount:  result.UseCount,
		CreatedAt: result.CreatedAt.Format(time.RFC3339),
	})
}

// GetHashtag 返回话题详情。
func (h *VideoHandler) GetHashtag(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	hashtagID, err := strconv.ParseInt(ctx.Param("hid"), 10, 64)
	if err != nil || hashtagID <= 0 {
		fail(ctx, errno.ErrInvalidParam)
		return
	}

	result, err := h.appCtx.VideoService.GetHashtag(ctx.Request.Context(), hashtagID)
	if err != nil {
		fail(ctx, err)
		return
	}

	success(ctx, types.HashtagData{
		ID:        result.ID,
		Name:      result.Name,
		UseCount:  result.UseCount,
		CreatedAt: result.CreatedAt.Format(time.RFC3339),
	})
}

// ListHashtagVideos 返回话题下视频列表。
func (h *VideoHandler) ListHashtagVideos(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	hashtagID, err := strconv.ParseInt(ctx.Param("hid"), 10, 64)
	if err != nil || hashtagID <= 0 {
		fail(ctx, errno.ErrInvalidParam)
		return
	}

	var cursor *time.Time
	cursorRaw := ctx.Query("cursor")
	if cursorRaw != "" {
		parsed, err := time.Parse(time.RFC3339, cursorRaw)
		if err != nil {
			fail(ctx, errno.ErrInvalidParam)
			return
		}
		cursor = &parsed
	}

	limit := 20
	if limitRaw := ctx.Query("limit"); limitRaw != "" {
		parsedLimit, err := strconv.Atoi(limitRaw)
		if err != nil {
			fail(ctx, errno.ErrInvalidParam)
			return
		}
		limit = parsedLimit
	}

	result, err := h.appCtx.VideoService.ListHashtagVideos(ctx.Request.Context(), hashtagID, videoservice.ListHashtagVideosRequest{
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		fail(ctx, err)
		return
	}

	items := make([]types.HashtagVideoData, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, types.HashtagVideoData{
			VideoID:         item.VideoID,
			UserID:          item.UserID,
			Title:           item.Title,
			ObjectKey:       item.ObjectKey,
			SourceURL:       item.SourceURL,
			CoverURL:        item.CoverURL,
			Visibility:      item.Visibility,
			TranscodeStatus: item.TranscodeStatus,
			AuditStatus:     item.AuditStatus,
			CreatedAt:       item.CreatedAt.Format(time.RFC3339),
		})
	}

	payload := types.HashtagVideoListData{
		Items: items,
	}
	if result.NextCursor != nil {
		payload.NextCursor = result.NextCursor.Format(time.RFC3339)
	}

	success(ctx, payload)
}
