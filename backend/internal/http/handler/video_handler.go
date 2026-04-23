package handler

import (
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
