package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/AbePhh/TikTide/backend/internal/app"
	feedservice "github.com/AbePhh/TikTide/backend/internal/feed/service"
	ginmiddleware "github.com/AbePhh/TikTide/backend/internal/http/middleware"
	"github.com/AbePhh/TikTide/backend/internal/http/types"
	recommendservice "github.com/AbePhh/TikTide/backend/internal/recommend/service"
	"github.com/AbePhh/TikTide/backend/pkg/errno"
)

type FeedHandler struct {
	appCtx *app.Context
}

func NewFeedHandler(appCtx *app.Context) *FeedHandler {
	return &FeedHandler{appCtx: appCtx}
}

func (h *FeedHandler) ListFollowing(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	var cursor int64
	if cursorRaw := ctx.Query("cursor"); cursorRaw != "" {
		parsed, err := strconv.ParseInt(cursorRaw, 10, 64)
		if err != nil || parsed < 0 {
			fail(ctx, errno.ErrFeedInvalidCursor)
			return
		}
		cursor = parsed
	}

	limit := 20
	if limitRaw := ctx.Query("limit"); limitRaw != "" {
		parsed, err := strconv.Atoi(limitRaw)
		if err != nil || parsed <= 0 {
			fail(ctx, errno.ErrInvalidParam)
			return
		}
		limit = parsed
	}

	result, err := h.appCtx.FeedService.ListFollowing(ctx.Request.Context(), authInfo.UserID, feedservice.ListRequest{
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		fail(ctx, err)
		return
	}

	items := make([]types.FeedVideoData, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, types.FeedVideoData{
			VideoID:             item.Detail.VideoID,
			UserID:              item.Detail.UserID,
			Title:               item.Detail.Title,
			ObjectKey:           item.Detail.ObjectKey,
			SourceURL:           item.Detail.SourceURL,
			CoverURL:            item.Detail.CoverURL,
			DurationMS:          item.Detail.DurationMS,
			AllowComment:        item.Detail.AllowComment,
			Visibility:          item.Detail.Visibility,
			TranscodeStatus:     item.Detail.TranscodeStatus,
			AuditStatus:         item.Detail.AuditStatus,
			TranscodeFailReason: item.Detail.TranscodeFailReason,
			AuditRemark:         item.Detail.AuditRemark,
			PlayCount:           item.Detail.PlayCount,
			LikeCount:           item.Detail.LikeCount,
			CommentCount:        item.Detail.CommentCount,
			FavoriteCount:       item.Detail.FavoriteCount,
			CreatedAt:           item.Detail.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:           item.Detail.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Author: types.FeedAuthorData{
				ID:        item.AuthorID,
				Username:  item.AuthorHandle,
				Nickname:  item.AuthorName,
				AvatarURL: item.AuthorAvatar,
			},
			Interact: types.FeedInteractData{
				IsFollowed:  item.IsFollowed,
				IsLiked:     item.IsLiked,
				IsFavorited: item.IsFavorited,
			},
		})
	}

	success(ctx, types.FeedVideoListData{
		Items:      items,
		NextCursor: result.NextCursor,
	})
}

func (h *FeedHandler) ListRecommend(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}
	if h.appCtx.RecommendService == nil {
		fail(ctx, errno.ErrFeedFetchFailed)
		return
	}

	var cursor int64
	if cursorRaw := ctx.Query("cursor"); cursorRaw != "" {
		parsed, err := strconv.ParseInt(cursorRaw, 10, 64)
		if err != nil || parsed < 0 {
			fail(ctx, errno.ErrFeedInvalidCursor)
			return
		}
		cursor = parsed
	}

	limit := 20
	if limitRaw := ctx.Query("limit"); limitRaw != "" {
		parsed, err := strconv.Atoi(limitRaw)
		if err != nil || parsed <= 0 {
			fail(ctx, errno.ErrInvalidParam)
			return
		}
		limit = parsed
	}

	result, err := h.appCtx.RecommendService.ListRecommend(ctx.Request.Context(), authInfo.UserID, recommendservice.ListRequest{
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		fail(ctx, err)
		return
	}

	items := make([]types.FeedVideoData, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, types.FeedVideoData{
			VideoID:             item.Detail.VideoID,
			UserID:              item.Detail.UserID,
			Title:               item.Detail.Title,
			ObjectKey:           item.Detail.ObjectKey,
			SourceURL:           item.Detail.SourceURL,
			CoverURL:            item.Detail.CoverURL,
			DurationMS:          item.Detail.DurationMS,
			AllowComment:        item.Detail.AllowComment,
			Visibility:          item.Detail.Visibility,
			TranscodeStatus:     item.Detail.TranscodeStatus,
			AuditStatus:         item.Detail.AuditStatus,
			TranscodeFailReason: item.Detail.TranscodeFailReason,
			AuditRemark:         item.Detail.AuditRemark,
			PlayCount:           item.Detail.PlayCount,
			LikeCount:           item.Detail.LikeCount,
			CommentCount:        item.Detail.CommentCount,
			FavoriteCount:       item.Detail.FavoriteCount,
			CreatedAt:           item.Detail.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:           item.Detail.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Author: types.FeedAuthorData{
				ID:        item.AuthorID,
				Username:  item.AuthorHandle,
				Nickname:  item.AuthorName,
				AvatarURL: item.AuthorAvatar,
			},
			Interact: types.FeedInteractData{
				IsFollowed:  item.IsFollowed,
				IsLiked:     item.IsLiked,
				IsFavorited: item.IsFavorited,
			},
		})
	}

	success(ctx, types.FeedVideoListData{
		Items:      items,
		NextCursor: result.NextCursor,
	})
}
