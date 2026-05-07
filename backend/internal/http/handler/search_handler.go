package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/AbePhh/TikTide/backend/internal/app"
	ginmiddleware "github.com/AbePhh/TikTide/backend/internal/http/middleware"
	"github.com/AbePhh/TikTide/backend/internal/http/types"
	searchservice "github.com/AbePhh/TikTide/backend/internal/search/service"
	"github.com/AbePhh/TikTide/backend/pkg/errno"
)

type SearchHandler struct {
	appCtx *app.Context
}

func NewSearchHandler(appCtx *app.Context) *SearchHandler {
	return &SearchHandler{appCtx: appCtx}
}

func (h *SearchHandler) SearchUsers(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}
	if h.appCtx.SearchService == nil {
		fail(ctx, errno.ErrSearchUnavailable)
		return
	}

	req, err := parseSearchRequest(ctx)
	if err != nil {
		fail(ctx, err)
		return
	}

	result, err := h.appCtx.SearchService.SearchUsers(ctx.Request.Context(), authInfo.UserID, req)
	if err != nil {
		fail(ctx, err)
		return
	}

	items := make([]types.SearchUserData, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, types.SearchUserData{
			ID:            item.ID,
			Username:      item.Username,
			Nickname:      item.Nickname,
			AvatarURL:     item.AvatarURL,
			Signature:     item.Signature,
			FollowerCount: item.FollowerCount,
			FollowCount:   item.FollowCount,
			WorkCount:     item.WorkCount,
			IsFollowed:    item.IsFollowed,
			IsMutual:      item.IsMutual,
		})
	}

	success(ctx, types.SearchUsersResponseData{
		Items:      items,
		NextCursor: result.NextCursor,
	})
}

func (h *SearchHandler) SearchHashtags(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}
	if h.appCtx.SearchService == nil {
		fail(ctx, errno.ErrSearchUnavailable)
		return
	}

	req, err := parseSearchRequest(ctx)
	if err != nil {
		fail(ctx, err)
		return
	}

	result, err := h.appCtx.SearchService.SearchHashtags(ctx.Request.Context(), req)
	if err != nil {
		fail(ctx, err)
		return
	}

	items := make([]types.SearchHashtagData, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, types.SearchHashtagData{
			ID:       item.ID,
			Name:     item.Name,
			UseCount: item.UseCount,
		})
	}

	success(ctx, types.SearchHashtagsResponseData{
		Items:      items,
		NextCursor: result.NextCursor,
	})
}

func (h *SearchHandler) SearchVideos(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}
	if h.appCtx.SearchService == nil {
		fail(ctx, errno.ErrSearchUnavailable)
		return
	}

	req, err := parseSearchRequest(ctx)
	if err != nil {
		fail(ctx, err)
		return
	}

	result, err := h.appCtx.SearchService.SearchVideos(ctx.Request.Context(), authInfo.UserID, req)
	if err != nil {
		fail(ctx, err)
		return
	}

	items := make([]types.SearchVideoData, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, types.SearchVideoData{
			VideoID:         item.VideoID,
			UserID:          item.UserID,
			Title:           item.Title,
			CoverURL:        item.CoverURL,
			SourceURL:       item.SourceURL,
			PlayCount:       item.PlayCount,
			LikeCount:       item.LikeCount,
			CommentCount:    item.CommentCount,
			FavoriteCount:   item.FavoriteCount,
			Visibility:      item.Visibility,
			AuditStatus:     item.AuditStatus,
			TranscodeStatus: item.TranscodeStatus,
			Author: types.FeedAuthorData{
				ID:        item.UserID,
				Username:  item.AuthorUsername,
				Nickname:  item.AuthorNickname,
				AvatarURL: item.AuthorAvatarURL,
			},
			Interact: types.FeedInteractData{
				IsFollowed: item.IsFollowed,
				IsLiked:    false,
				IsFavorited:false,
			},
		})
	}

	success(ctx, types.SearchVideosResponseData{
		Items:      items,
		NextCursor: result.NextCursor,
	})
}

func (h *SearchHandler) SearchAll(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok || authInfo.UserID <= 0 {
		fail(ctx, errno.ErrUnauthorized)
		return
	}
	if h.appCtx.SearchService == nil {
		fail(ctx, errno.ErrSearchUnavailable)
		return
	}

	req, err := parseSearchRequest(ctx)
	if err != nil {
		fail(ctx, err)
		return
	}

	result, err := h.appCtx.SearchService.SearchAll(ctx.Request.Context(), authInfo.UserID, req)
	if err != nil {
		fail(ctx, err)
		return
	}

	users := make([]types.SearchUserData, 0, len(result.Users))
	for _, item := range result.Users {
		users = append(users, types.SearchUserData{
			ID:            item.ID,
			Username:      item.Username,
			Nickname:      item.Nickname,
			AvatarURL:     item.AvatarURL,
			Signature:     item.Signature,
			FollowerCount: item.FollowerCount,
			FollowCount:   item.FollowCount,
			WorkCount:     item.WorkCount,
			IsFollowed:    item.IsFollowed,
			IsMutual:      item.IsMutual,
		})
	}

	hashtags := make([]types.SearchHashtagData, 0, len(result.Hashtags))
	for _, item := range result.Hashtags {
		hashtags = append(hashtags, types.SearchHashtagData{
			ID:       item.ID,
			Name:     item.Name,
			UseCount: item.UseCount,
		})
	}

	videos := make([]types.SearchVideoData, 0, len(result.Videos))
	for _, item := range result.Videos {
		videos = append(videos, types.SearchVideoData{
			VideoID:         item.VideoID,
			UserID:          item.UserID,
			Title:           item.Title,
			CoverURL:        item.CoverURL,
			SourceURL:       item.SourceURL,
			PlayCount:       item.PlayCount,
			LikeCount:       item.LikeCount,
			CommentCount:    item.CommentCount,
			FavoriteCount:   item.FavoriteCount,
			Visibility:      item.Visibility,
			AuditStatus:     item.AuditStatus,
			TranscodeStatus: item.TranscodeStatus,
			Author: types.FeedAuthorData{
				ID:        item.UserID,
				Username:  item.AuthorUsername,
				Nickname:  item.AuthorNickname,
				AvatarURL: item.AuthorAvatarURL,
			},
			Interact: types.FeedInteractData{
				IsFollowed: item.IsFollowed,
			},
		})
	}

	success(ctx, types.SearchAllResponseData{
		Users:    users,
		Hashtags: hashtags,
		Videos:   videos,
	})
}

func parseSearchRequest(ctx *gin.Context) (searchservice.SearchRequest, error) {
	query := strings.TrimSpace(ctx.Query("q"))
	if query == "" {
		return searchservice.SearchRequest{}, errno.ErrInvalidParam
	}

	limit := 10
	if limitRaw := ctx.Query("limit"); limitRaw != "" {
		parsed, err := strconv.Atoi(limitRaw)
		if err != nil || parsed <= 0 {
			return searchservice.SearchRequest{}, errno.ErrInvalidParam
		}
		limit = parsed
	}

	return searchservice.SearchRequest{
		Query:  query,
		Cursor: strings.TrimSpace(ctx.Query("cursor")),
		Limit:  limit,
	}, nil
}

