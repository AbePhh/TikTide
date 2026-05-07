package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	searchmodel "github.com/AbePhh/TikTide/backend/internal/search/model"
	relationservice "github.com/AbePhh/TikTide/backend/internal/relation/service"
	usermodel "github.com/AbePhh/TikTide/backend/internal/user/model"
	videomodel "github.com/AbePhh/TikTide/backend/internal/video/model"
	videoservice "github.com/AbePhh/TikTide/backend/internal/video/service"
	"github.com/AbePhh/TikTide/backend/pkg/errno"
)

type UserRepository interface {
	GetByID(ctx context.Context, userID int64) (*usermodel.User, error)
	GetStatsByID(ctx context.Context, userID int64) (*usermodel.UserStats, error)
	ListUsersWithStats(ctx context.Context, limit, offset int) ([]usermodel.UserWithStats, error)
}

type VideoRepository interface {
	GetVideoByID(ctx context.Context, videoID int64) (*videomodel.Video, error)
	ListVideosByIDs(ctx context.Context, videoIDs []int64) ([]videomodel.Video, error)
	ListVideosForSearch(ctx context.Context, limit, offset int) ([]videomodel.Video, error)
	ListHashtagNamesByVideoIDs(ctx context.Context, videoIDs []int64) (map[int64][]string, error)
	GetHashtagByID(ctx context.Context, hashtagID int64) (*videomodel.Hashtag, error)
	ListHashtags(ctx context.Context, limit, offset int) ([]videomodel.Hashtag, error)
}

type VideoDetailProvider interface {
	GetVideoDetail(ctx context.Context, viewerUserID, videoID int64) (*videoservice.VideoDetailResult, error)
}

type SearchService interface {
	Initialize(ctx context.Context) error
	SearchUsers(ctx context.Context, viewerUserID int64, req SearchRequest) (*UserSearchResult, error)
	SearchHashtags(ctx context.Context, req SearchRequest) (*HashtagSearchResult, error)
	SearchVideos(ctx context.Context, viewerUserID int64, req SearchRequest) (*VideoSearchResult, error)
	SearchAll(ctx context.Context, viewerUserID int64, req SearchRequest) (*AllSearchResult, error)
	UpsertUserDocument(ctx context.Context, userID int64) error
	UpsertHashtagDocument(ctx context.Context, hashtagID int64) error
	UpsertVideoDocument(ctx context.Context, videoID int64) error
	DeleteUserDocument(ctx context.Context, userID int64) error
	DeleteHashtagDocument(ctx context.Context, hashtagID int64) error
	DeleteVideoDocument(ctx context.Context, videoID int64) error
	RebuildAll(ctx context.Context) error
}

type SearchIndexer interface {
	UpsertUserDocument(ctx context.Context, userID int64) error
	UpsertHashtagDocument(ctx context.Context, hashtagID int64) error
	UpsertVideoDocument(ctx context.Context, videoID int64) error
}

type SearchRequest struct {
	Query  string
	Cursor string
	Limit  int
}

type UserSearchItem struct {
	ID            int64
	Username      string
	Nickname      string
	AvatarURL     string
	Signature     string
	FollowerCount int64
	FollowCount   int64
	WorkCount     int64
	IsFollowed    bool
	IsMutual      bool
}

type UserSearchResult struct {
	Items      []UserSearchItem
	NextCursor string
}

type HashtagSearchItem struct {
	ID       int64
	Name     string
	UseCount int64
}

type HashtagSearchResult struct {
	Items      []HashtagSearchItem
	NextCursor string
}

type VideoSearchItem struct {
	VideoID         int64
	UserID          int64
	Title           string
	CoverURL        string
	SourceURL       string
	PlayCount       int64
	LikeCount       int64
	CommentCount    int64
	FavoriteCount   int64
	Visibility      int8
	AuditStatus     int8
	TranscodeStatus int8
	AuthorUsername  string
	AuthorNickname  string
	AuthorAvatarURL string
	IsFollowed      bool
	IsMutual        bool
}

type VideoSearchResult struct {
	Items      []VideoSearchItem
	NextCursor string
}

type AllSearchResult struct {
	Users    []UserSearchItem
	Hashtags []HashtagSearchItem
	Videos   []VideoSearchItem
}

type Service struct {
	repo      searchmodel.Repository
	userRepo  UserRepository
	videoRepo VideoRepository
	videoSvc  VideoDetailProvider
	relations relationservice.RelationService
}

func New(repo searchmodel.Repository, userRepo UserRepository, videoRepo VideoRepository, videoSvc VideoDetailProvider, relations relationservice.RelationService) *Service {
	return &Service{
		repo:      repo,
		userRepo:  userRepo,
		videoRepo: videoRepo,
		videoSvc:  videoSvc,
		relations: relations,
	}
}

func (s *Service) Initialize(ctx context.Context) error {
	if s.repo == nil {
		return errno.ErrSearchUnavailable
	}
	if err := s.repo.Initialize(ctx); err != nil {
		return fmt.Errorf("%w: %v", errno.ErrSearchFailed, err)
	}
	return nil
}

func (s *Service) SearchUsers(ctx context.Context, viewerUserID int64, req SearchRequest) (*UserSearchResult, error) {
	if s.repo == nil {
		return nil, errno.ErrSearchUnavailable
	}
	query := strings.TrimSpace(req.Query)
	if len(query) < 1 {
		return &UserSearchResult{Items: []UserSearchItem{}}, nil
	}
	result, err := s.repo.SearchUsers(ctx, searchmodel.SearchRequest{
		Query:  query,
		Cursor: strings.TrimSpace(req.Cursor),
		Limit:  req.Limit,
	})
	if err != nil {
		return nil, normalizeSearchError(err)
	}

	items := make([]UserSearchItem, 0, len(result.Hits))
	for _, hit := range result.Hits {
		userID, parseErr := strconv.ParseInt(hit.ID, 10, 64)
		if parseErr != nil {
			continue
		}
		user, stats, err := s.loadUserSummary(ctx, userID)
		if err != nil {
			continue
		}
		item := UserSearchItem{
			ID:            user.ID,
			Username:      user.Username,
			Nickname:      user.Nickname,
			AvatarURL:     user.AvatarURL,
			Signature:     user.Signature,
			FollowerCount: stats.FollowerCount,
			FollowCount:   stats.FollowCount,
			WorkCount:     stats.WorkCount,
		}
		if viewerUserID > 0 && viewerUserID != user.ID && s.relations != nil {
			state, relationErr := s.relations.GetRelationState(ctx, viewerUserID, user.ID)
			if relationErr == nil {
				item.IsFollowed = state.IsFollowed
				item.IsMutual = state.IsMutual
			}
		}
		items = append(items, item)
	}

	return &UserSearchResult{
		Items:      items,
		NextCursor: result.NextCursor,
	}, nil
}

func (s *Service) SearchHashtags(ctx context.Context, req SearchRequest) (*HashtagSearchResult, error) {
	if s.repo == nil {
		return nil, errno.ErrSearchUnavailable
	}
	query := strings.TrimSpace(req.Query)
	if len(query) < 1 {
		return &HashtagSearchResult{Items: []HashtagSearchItem{}}, nil
	}
	result, err := s.repo.SearchHashtags(ctx, searchmodel.SearchRequest{
		Query:  query,
		Cursor: strings.TrimSpace(req.Cursor),
		Limit:  req.Limit,
	})
	if err != nil {
		return nil, normalizeSearchError(err)
	}

	items := make([]HashtagSearchItem, 0, len(result.Hits))
	for _, hit := range result.Hits {
		hashtagID, parseErr := strconv.ParseInt(hit.ID, 10, 64)
		if parseErr != nil {
			continue
		}
		hashtag, err := s.videoRepo.GetHashtagByID(ctx, hashtagID)
		if err != nil {
			continue
		}
		items = append(items, HashtagSearchItem{
			ID:       hashtag.ID,
			Name:     hashtag.Name,
			UseCount: hashtag.UseCount,
		})
	}

	return &HashtagSearchResult{
		Items:      items,
		NextCursor: result.NextCursor,
	}, nil
}

func (s *Service) SearchVideos(ctx context.Context, viewerUserID int64, req SearchRequest) (*VideoSearchResult, error) {
	if s.repo == nil {
		return nil, errno.ErrSearchUnavailable
	}
	query := strings.TrimSpace(req.Query)
	if len(query) < 1 {
		return &VideoSearchResult{Items: []VideoSearchItem{}}, nil
	}
	result, err := s.repo.SearchVideos(ctx, searchmodel.SearchRequest{
		Query:  query,
		Cursor: strings.TrimSpace(req.Cursor),
		Limit:  req.Limit,
	})
	if err != nil {
		return nil, normalizeSearchError(err)
	}

	items := make([]VideoSearchItem, 0, len(result.Hits))
	videoIDs := make([]int64, 0, len(result.Hits))
	for _, hit := range result.Hits {
		videoID, parseErr := strconv.ParseInt(hit.ID, 10, 64)
		if parseErr == nil {
			videoIDs = append(videoIDs, videoID)
		}
	}
	videos, err := s.videoRepo.ListVideosByIDs(ctx, videoIDs)
	if err != nil {
		return nil, errno.ErrSearchFailed
	}
	videoMap := make(map[int64]videomodel.Video, len(videos))
	for _, video := range videos {
		videoMap[video.ID] = video
	}

	for _, videoID := range videoIDs {
		video, ok := videoMap[videoID]
		if !ok {
			continue
		}
		detail := (*videoservice.VideoDetailResult)(nil)
		if s.videoSvc != nil {
			detail, _ = s.videoSvc.GetVideoDetail(ctx, viewerUserID, video.ID)
		}
		user, _, userErr := s.loadUserSummary(ctx, video.UserID)
		if userErr != nil {
			continue
		}
		item := VideoSearchItem{
			VideoID:         video.ID,
			UserID:          video.UserID,
			Title:           video.Title,
			CoverURL:        video.CoverURL,
			SourceURL:       video.SourceURL,
			PlayCount:       video.PlayCount,
			LikeCount:       video.LikeCount,
			CommentCount:    video.CommentCount,
			FavoriteCount:   video.FavoriteCount,
			Visibility:      video.Visibility,
			AuditStatus:     video.AuditStatus,
			TranscodeStatus: video.TranscodeStatus,
			AuthorUsername:  user.Username,
			AuthorNickname:  user.Nickname,
			AuthorAvatarURL: user.AvatarURL,
		}
		if detail != nil {
			item.CoverURL = detail.CoverURL
			item.SourceURL = detail.SourceURL
			item.TranscodeStatus = detail.TranscodeStatus
			item.AuditStatus = detail.AuditStatus
			item.Visibility = detail.Visibility
		}
		if viewerUserID > 0 && viewerUserID != user.ID && s.relations != nil {
			state, relationErr := s.relations.GetRelationState(ctx, viewerUserID, user.ID)
			if relationErr == nil {
				item.IsFollowed = state.IsFollowed
				item.IsMutual = state.IsMutual
			}
		}
		items = append(items, item)
	}

	return &VideoSearchResult{
		Items:      items,
		NextCursor: result.NextCursor,
	}, nil
}

func (s *Service) SearchAll(ctx context.Context, viewerUserID int64, req SearchRequest) (*AllSearchResult, error) {
	usersResult, err := s.SearchUsers(ctx, viewerUserID, SearchRequest{
		Query: req.Query,
		Limit: minPositive(req.Limit, 3),
	})
	if err != nil {
		return nil, err
	}
	hashtagsResult, err := s.SearchHashtags(ctx, SearchRequest{
		Query: req.Query,
		Limit: minPositive(req.Limit, 3),
	})
	if err != nil {
		return nil, err
	}
	videosResult, err := s.SearchVideos(ctx, viewerUserID, SearchRequest{
		Query: req.Query,
		Limit: minPositive(req.Limit, 3),
	})
	if err != nil {
		return nil, err
	}
	return &AllSearchResult{
		Users:    usersResult.Items,
		Hashtags: hashtagsResult.Items,
		Videos:   videosResult.Items,
	}, nil
}

func (s *Service) UpsertUserDocument(ctx context.Context, userID int64) error {
	if s.repo == nil || userID <= 0 {
		return nil
	}
	user, stats, err := s.loadUserSummary(ctx, userID)
	if err != nil {
		return err
	}
	return normalizeSearchError(s.repo.UpsertUserDocument(ctx, searchmodel.UserDocument{
		ID:            strconv.FormatInt(user.ID, 10),
		Username:      user.Username,
		Nickname:      user.Nickname,
		Signature:     user.Signature,
		AvatarURL:     user.AvatarURL,
		Status:        user.Status,
		FollowerCount: stats.FollowerCount,
		FollowCount:   stats.FollowCount,
		WorkCount:     stats.WorkCount,
		CreatedAt:     user.CreatedAt,
	}))
}

func (s *Service) UpsertHashtagDocument(ctx context.Context, hashtagID int64) error {
	if s.repo == nil || hashtagID <= 0 {
		return nil
	}
	hashtag, err := s.videoRepo.GetHashtagByID(ctx, hashtagID)
	if err != nil {
		return errno.ErrSearchFailed
	}
	return normalizeSearchError(s.repo.UpsertHashtagDocument(ctx, searchmodel.HashtagDocument{
		ID:        strconv.FormatInt(hashtag.ID, 10),
		Name:      hashtag.Name,
		UseCount:  hashtag.UseCount,
		CreatedAt: hashtag.CreatedAt,
	}))
}

func (s *Service) UpsertVideoDocument(ctx context.Context, videoID int64) error {
	if s.repo == nil || videoID <= 0 {
		return nil
	}
	video, err := s.videoRepo.GetVideoByID(ctx, videoID)
	if err != nil {
		return errno.ErrSearchFailed
	}
	user, _, err := s.loadUserSummary(ctx, video.UserID)
	if err != nil {
		return err
	}
	hashtagMap, err := s.videoRepo.ListHashtagNamesByVideoIDs(ctx, []int64{videoID})
	if err != nil {
		return errno.ErrSearchFailed
	}
	return normalizeSearchError(s.repo.UpsertVideoDocument(ctx, searchmodel.VideoDocument{
		ID:              strconv.FormatInt(video.ID, 10),
		Title:           video.Title,
		UserID:          strconv.FormatInt(video.UserID, 10),
		AuthorUsername:  user.Username,
		AuthorNickname:  user.Nickname,
		Hashtags:        hashtagMap[video.ID],
		CoverURL:        video.CoverURL,
		PlayCount:       video.PlayCount,
		LikeCount:       video.LikeCount,
		CommentCount:    video.CommentCount,
		FavoriteCount:   video.FavoriteCount,
		Visibility:      video.Visibility,
		AuditStatus:     video.AuditStatus,
		TranscodeStatus: video.TranscodeStatus,
		CreatedAt:       video.CreatedAt,
	}))
}

func (s *Service) DeleteUserDocument(ctx context.Context, userID int64) error {
	if s.repo == nil || userID <= 0 {
		return nil
	}
	return normalizeSearchError(s.repo.DeleteUserDocument(ctx, strconv.FormatInt(userID, 10)))
}

func (s *Service) DeleteHashtagDocument(ctx context.Context, hashtagID int64) error {
	if s.repo == nil || hashtagID <= 0 {
		return nil
	}
	return normalizeSearchError(s.repo.DeleteHashtagDocument(ctx, strconv.FormatInt(hashtagID, 10)))
}

func (s *Service) DeleteVideoDocument(ctx context.Context, videoID int64) error {
	if s.repo == nil || videoID <= 0 {
		return nil
	}
	return normalizeSearchError(s.repo.DeleteVideoDocument(ctx, strconv.FormatInt(videoID, 10)))
}

func (s *Service) RebuildAll(ctx context.Context) error {
	if s.repo == nil {
		return nil
	}
	if err := s.rebuildUsers(ctx); err != nil {
		return err
	}
	if err := s.rebuildHashtags(ctx); err != nil {
		return err
	}
	if err := s.rebuildVideos(ctx); err != nil {
		return err
	}
	return nil
}

func (s *Service) rebuildUsers(ctx context.Context) error {
	const batchSize = 100
	for offset := 0; ; offset += batchSize {
		items, err := s.userRepo.ListUsersWithStats(ctx, batchSize, offset)
		if err != nil {
			return errno.ErrSearchFailed
		}
		if len(items) == 0 {
			return nil
		}
		for _, item := range items {
			if err := normalizeSearchError(s.repo.UpsertUserDocument(ctx, searchmodel.UserDocument{
				ID:            strconv.FormatInt(item.User.ID, 10),
				Username:      item.User.Username,
				Nickname:      item.User.Nickname,
				Signature:     item.User.Signature,
				AvatarURL:     item.User.AvatarURL,
				Status:        item.User.Status,
				FollowerCount: item.Stats.FollowerCount,
				FollowCount:   item.Stats.FollowCount,
				WorkCount:     item.Stats.WorkCount,
				CreatedAt:     item.User.CreatedAt,
			})); err != nil {
				return err
			}
		}
	}
}

func (s *Service) rebuildHashtags(ctx context.Context) error {
	const batchSize = 100
	for offset := 0; ; offset += batchSize {
		items, err := s.videoRepo.ListHashtags(ctx, batchSize, offset)
		if err != nil {
			return errno.ErrSearchFailed
		}
		if len(items) == 0 {
			return nil
		}
		for _, item := range items {
			if err := normalizeSearchError(s.repo.UpsertHashtagDocument(ctx, searchmodel.HashtagDocument{
				ID:        strconv.FormatInt(item.ID, 10),
				Name:      item.Name,
				UseCount:  item.UseCount,
				CreatedAt: item.CreatedAt,
			})); err != nil {
				return err
			}
		}
	}
}

func (s *Service) rebuildVideos(ctx context.Context) error {
	const batchSize = 100
	for offset := 0; ; offset += batchSize {
		items, err := s.videoRepo.ListVideosForSearch(ctx, batchSize, offset)
		if err != nil {
			return errno.ErrSearchFailed
		}
		if len(items) == 0 {
			return nil
		}

		videoIDs := make([]int64, 0, len(items))
		for _, item := range items {
			videoIDs = append(videoIDs, item.ID)
		}
		hashtagMap, err := s.videoRepo.ListHashtagNamesByVideoIDs(ctx, videoIDs)
		if err != nil {
			return errno.ErrSearchFailed
		}

		for _, item := range items {
			user, _, userErr := s.loadUserSummary(ctx, item.UserID)
			if userErr != nil {
				continue
			}
			if err := normalizeSearchError(s.repo.UpsertVideoDocument(ctx, searchmodel.VideoDocument{
				ID:              strconv.FormatInt(item.ID, 10),
				Title:           item.Title,
				UserID:          strconv.FormatInt(item.UserID, 10),
				AuthorUsername:  user.Username,
				AuthorNickname:  user.Nickname,
				Hashtags:        hashtagMap[item.ID],
				CoverURL:        item.CoverURL,
				PlayCount:       item.PlayCount,
				LikeCount:       item.LikeCount,
				CommentCount:    item.CommentCount,
				FavoriteCount:   item.FavoriteCount,
				Visibility:      item.Visibility,
				AuditStatus:     item.AuditStatus,
				TranscodeStatus: item.TranscodeStatus,
				CreatedAt:       item.CreatedAt,
			})); err != nil {
				return err
			}
		}
	}
}

func (s *Service) loadUserSummary(ctx context.Context, userID int64) (*usermodel.User, *usermodel.UserStats, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, errno.ErrSearchFailed
	}
	stats, err := s.userRepo.GetStatsByID(ctx, userID)
	if err != nil {
		return nil, nil, errno.ErrSearchFailed
	}
	return user, stats, nil
}

func normalizeSearchError(err error) error {
	if err == nil {
		return nil
	}
	if errno.IsCode(err, errno.ErrSearchInvalidCursor.Code) || errno.IsCode(err, errno.ErrSearchUnavailable.Code) {
		return err
	}
	return errno.ErrSearchFailed
}

func minPositive(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
