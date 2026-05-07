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

type VideoHandler struct {
	appCtx *app.Context
}

func NewVideoHandler(appCtx *app.Context) *VideoHandler {
	return &VideoHandler{appCtx: appCtx}
}

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
		FileName:    req.FileName,
		ContentType: req.ContentType,
		ObjectKey:   req.ObjectKey,
	})
	if err != nil {
		fail(ctx, err)
		return
	}

	success(ctx, types.UploadCredentialData{
		ObjectKey:    result.ObjectKey,
		UploadURL:    result.UploadURL,
		UploadMethod: result.UploadMethod,
		ContentType:  result.ContentType,
		ExpiresAt:    result.ExpiresAt.Format(time.RFC3339),
		UploadToken:  result.UploadToken,
	})
}

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

func (h *VideoHandler) GetVideo(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	videoID, err := strconv.ParseInt(ctx.Param("vid"), 10, 64)
	if err != nil || videoID <= 0 {
		fail(ctx, errno.ErrInvalidParam)
		return
	}

	result, err := h.appCtx.VideoService.GetVideoDetail(ctx.Request.Context(), authInfo.UserID, videoID)
	if err != nil {
		fail(ctx, err)
		return
	}

	success(ctx, buildVideoDetailData(*result))
}

func (h *VideoHandler) GetVideoResources(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	videoID, err := strconv.ParseInt(ctx.Param("vid"), 10, 64)
	if err != nil || videoID <= 0 {
		fail(ctx, errno.ErrInvalidParam)
		return
	}

	result, err := h.appCtx.VideoService.GetVideoResources(ctx.Request.Context(), authInfo.UserID, videoID)
	if err != nil {
		fail(ctx, err)
		return
	}

	items := make([]types.VideoResourceData, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, types.VideoResourceData{
			VideoID:    item.VideoID,
			Resolution: item.Resolution,
			FileURL:    item.FileURL,
			FileSize:   item.FileSize,
			Bitrate:    item.Bitrate,
			CreatedAt:  item.CreatedAt.Format(time.RFC3339),
		})
	}

	success(ctx, types.VideoResourceListData{Items: items})
}

func (h *VideoHandler) ReportPlay(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	var req types.VideoPlayReportRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fail(ctx, errno.ErrInvalidParam)
		return
	}

	if err := h.appCtx.VideoService.ReportPlay(ctx.Request.Context(), authInfo.UserID, videoservice.ReportPlayRequest{
		VideoID: req.VideoID.Int64(),
	}); err != nil {
		fail(ctx, err)
		return
	}

	success(ctx, gin.H{"reported": true})
}

func (h *VideoHandler) ListUserVideos(ctx *gin.Context) {
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

	result, err := h.appCtx.VideoService.ListUserVideos(ctx.Request.Context(), authInfo.UserID, targetUserID, videoservice.ListUserVideosRequest{
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		fail(ctx, err)
		return
	}

	items := make([]types.VideoDetailData, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, buildVideoDetailData(item))
	}

	payload := types.UserVideoListData{
		Items: items,
	}
	if result.NextCursor != nil {
		payload.NextCursor = result.NextCursor.Format(time.RFC3339)
	}

	success(ctx, payload)
}

func (h *VideoHandler) SaveDraft(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	var req types.SaveDraftRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fail(ctx, errno.ErrInvalidParam)
		return
	}

	result, err := h.appCtx.VideoService.SaveDraft(ctx.Request.Context(), authInfo.UserID, videoservice.SaveDraftRequest{
		DraftID:      req.DraftID.Int64(),
		ObjectKey:    req.ObjectKey,
		CoverURL:     req.CoverURL,
		Title:        req.Title,
		TagNames:     req.TagNames,
		AllowComment: req.AllowComment,
		Visibility:   req.Visibility,
	})
	if err != nil {
		fail(ctx, err)
		return
	}

	success(ctx, buildDraftData(*result))
}

func (h *VideoHandler) GetDraft(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	draftID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || draftID <= 0 {
		fail(ctx, errno.ErrInvalidParam)
		return
	}

	result, err := h.appCtx.VideoService.GetDraft(ctx.Request.Context(), authInfo.UserID, draftID)
	if err != nil {
		fail(ctx, err)
		return
	}

	success(ctx, buildDraftData(*result))
}

func (h *VideoHandler) ListDrafts(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	result, err := h.appCtx.VideoService.ListDrafts(ctx.Request.Context(), authInfo.UserID)
	if err != nil {
		fail(ctx, err)
		return
	}

	items := make([]types.DraftData, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, buildDraftData(item))
	}

	success(ctx, types.DraftListData{Items: items})
}

func (h *VideoHandler) DeleteDraft(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	draftID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil || draftID <= 0 {
		fail(ctx, errno.ErrInvalidParam)
		return
	}

	if err := h.appCtx.VideoService.DeleteDraft(ctx.Request.Context(), authInfo.UserID, draftID); err != nil {
		fail(ctx, err)
		return
	}

	success(ctx, gin.H{"deleted": true})
}

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

func (h *VideoHandler) ListHotHashtags(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	limit := 10
	if limitRaw := ctx.Query("limit"); limitRaw != "" {
		parsedLimit, err := strconv.Atoi(limitRaw)
		if err != nil {
			fail(ctx, errno.ErrInvalidParam)
			return
		}
		limit = parsedLimit
	}

	result, err := h.appCtx.VideoService.ListHotHashtags(ctx.Request.Context(), videoservice.ListHotHashtagsRequest{
		Limit: limit,
	})
	if err != nil {
		fail(ctx, err)
		return
	}

	items := make([]types.HashtagData, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, types.HashtagData{
			ID:        item.ID,
			Name:      item.Name,
			UseCount:  item.UseCount,
			CreatedAt: item.CreatedAt.Format(time.RFC3339),
		})
	}

	success(ctx, types.HashtagListData{Items: items})
}

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

func buildDraftData(result videoservice.DraftResult) types.DraftData {
	return types.DraftData{
		ID:           result.ID,
		ObjectKey:    result.ObjectKey,
		SourceURL:    result.SourceURL,
		CoverURL:     result.CoverURL,
		Title:        result.Title,
		TagNames:     result.TagNames,
		AllowComment: result.AllowComment,
		Visibility:   result.Visibility,
		CreatedAt:    result.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    result.UpdatedAt.Format(time.RFC3339),
	}
}

func buildVideoDetailData(result videoservice.VideoDetailResult) types.VideoDetailData {
	return types.VideoDetailData{
		VideoID:             result.VideoID,
		UserID:              result.UserID,
		Title:               result.Title,
		ObjectKey:           result.ObjectKey,
		SourceURL:           result.SourceURL,
		CoverURL:            result.CoverURL,
		DurationMS:          result.DurationMS,
		AllowComment:        result.AllowComment,
		Visibility:          result.Visibility,
		TranscodeStatus:     result.TranscodeStatus,
		AuditStatus:         result.AuditStatus,
		TranscodeFailReason: result.TranscodeFailReason,
		AuditRemark:         result.AuditRemark,
		PlayCount:           result.PlayCount,
		LikeCount:           result.LikeCount,
		CommentCount:        result.CommentCount,
		FavoriteCount:       result.FavoriteCount,
		CreatedAt:           result.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           result.UpdatedAt.Format(time.RFC3339),
	}
}
