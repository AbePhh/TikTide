package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/AbePhh/TikTide/backend/internal/video/model"
	"github.com/AbePhh/TikTide/backend/pkg/config"
	"github.com/AbePhh/TikTide/backend/pkg/errno"
	"github.com/AbePhh/TikTide/backend/pkg/rediskey"
	"github.com/AbePhh/TikTide/backend/pkg/utils"
)

type OSSClient interface {
	GeneratePutSignedURL(ctx context.Context, objectKey, contentType string, expire time.Duration) (string, error)
	GenerateGetSignedURL(ctx context.Context, objectKey string, expire time.Duration) (string, error)
	ObjectExists(ctx context.Context, objectKey string) (bool, error)
	ObjectURL(objectKey string) string
	GetObject(ctx context.Context, objectKey string) (io.ReadCloser, error)
	PutObject(ctx context.Context, objectKey string, reader io.Reader) error
}

type TranscodeDispatcher interface {
	Dispatch(ctx context.Context, videoID int64) error
}

type VideoService interface {
	CreateUploadCredential(ctx context.Context, userID int64, req CreateUploadCredentialRequest) (*UploadCredential, error)
	PublishVideo(ctx context.Context, userID int64, req PublishVideoRequest) (*PublishResult, error)
	GetVideoDetail(ctx context.Context, viewerUserID, videoID int64) (*VideoDetailResult, error)
	GetVideoResources(ctx context.Context, viewerUserID, videoID int64) (*VideoResourceListResult, error)
	ReportPlay(ctx context.Context, userID int64, req ReportPlayRequest) error
	ListUserVideos(ctx context.Context, viewerUserID, targetUserID int64, req ListUserVideosRequest) (*UserVideoListResult, error)
	GetVideoForTranscode(ctx context.Context, videoID int64) (*VideoDetailResult, error)
	StartTranscode(ctx context.Context, videoID int64) error
	CompleteTranscode(ctx context.Context, req CompleteTranscodeRequest) error
	FailTranscode(ctx context.Context, req FailTranscodeRequest) error
	SaveDraft(ctx context.Context, userID int64, req SaveDraftRequest) (*DraftResult, error)
	GetDraft(ctx context.Context, userID, draftID int64) (*DraftResult, error)
	ListDrafts(ctx context.Context, userID int64) (*DraftListResult, error)
	DeleteDraft(ctx context.Context, userID, draftID int64) error
	CreateHashtag(ctx context.Context, userID int64, req CreateHashtagRequest) (*HashtagResult, error)
	GetHashtag(ctx context.Context, hashtagID int64) (*HashtagResult, error)
	ListHotHashtags(ctx context.Context, req ListHotHashtagsRequest) (*HashtagListResult, error)
	ListHashtagVideos(ctx context.Context, hashtagID int64, req ListHashtagVideosRequest) (*HashtagVideoListResult, error)
}

type Service struct {
	repo       model.Repository
	oss        OSSClient
	ids        utils.IDGenerator
	cfg        config.Config
	redis      *redis.Client
	dispatcher TranscodeDispatcher
	search     SearchIndexer
}

type SearchIndexer interface {
	UpsertHashtagDocument(ctx context.Context, hashtagID int64) error
	UpsertVideoDocument(ctx context.Context, videoID int64) error
}

type CreateUploadCredentialRequest struct {
	FileName    string
	ContentType string
	ObjectKey   string
}

type UploadCredential struct {
	ObjectKey    string
	UploadURL    string
	UploadMethod string
	ContentType  string
	ExpiresAt    time.Time
	UploadToken  string
}

type PublishVideoRequest struct {
	ObjectKey    string
	Title        string
	HashtagIDs   []int64
	HashtagNames []string
	AllowComment int8
	Visibility   int8
}

type PublishResult struct {
	VideoID         int64
	ObjectKey       string
	SourceURL       string
	TranscodeStatus int8
}

type VideoDetailResult struct {
	VideoID             int64
	UserID              int64
	Title               string
	ObjectKey           string
	SourceURL           string
	CoverURL            string
	DurationMS          int32
	AllowComment        int8
	Visibility          int8
	TranscodeStatus     int8
	AuditStatus         int8
	TranscodeFailReason string
	AuditRemark         string
	PlayCount           int64
	LikeCount           int64
	CommentCount        int64
	FavoriteCount       int64
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type VideoResourceResult struct {
	VideoID    int64
	Resolution string
	FileURL    string
	FileSize   int64
	Bitrate    int32
	CreatedAt  time.Time
}

type VideoResourceListResult struct {
	Items []VideoResourceResult
}

type ListUserVideosRequest struct {
	Cursor *time.Time
	Limit  int
}

type ReportPlayRequest struct {
	VideoID int64
}

type UserVideoListResult struct {
	Items      []VideoDetailResult
	NextCursor *time.Time
}

type CompleteTranscodeRequest struct {
	VideoID    int64
	CoverURL   string
	DurationMS int32
	Resources  []TranscodedResource
}

type TranscodedResource struct {
	Resolution string
	FileURL    string
	FileSize   int64
	Bitrate    int32
}

type FailTranscodeRequest struct {
	VideoID    int64
	FailReason string
}

type SaveDraftRequest struct {
	DraftID      int64
	ObjectKey    string
	CoverURL     string
	Title        string
	TagNames     string
	AllowComment int8
	Visibility   int8
}

type DraftResult struct {
	ID           int64
	ObjectKey    string
	SourceURL    string
	CoverURL     string
	Title        string
	TagNames     string
	AllowComment int8
	Visibility   int8
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type DraftListResult struct {
	Items []DraftResult
}

type HashtagResult struct {
	ID        int64
	Name      string
	UseCount  int64
	CreatedAt time.Time
}

type CreateHashtagRequest struct {
	Name string
}

type ListHotHashtagsRequest struct {
	Limit int
}

type HashtagListResult struct {
	Items []HashtagResult
}

type HashtagVideoResult struct {
	VideoID         int64
	UserID          int64
	Title           string
	ObjectKey       string
	SourceURL       string
	CoverURL        string
	Visibility      int8
	TranscodeStatus int8
	AuditStatus     int8
	CreatedAt       time.Time
}

type ListHashtagVideosRequest struct {
	Cursor *time.Time
	Limit  int
}

type HashtagVideoListResult struct {
	Items      []HashtagVideoResult
	NextCursor *time.Time
}

func New(repo model.Repository, oss OSSClient, ids utils.IDGenerator, cfg config.Config) *Service {
	return &Service{
		repo: repo,
		oss:  oss,
		ids:  ids,
		cfg:  cfg,
	}
}

func (s *Service) SetTranscodeDispatcher(dispatcher TranscodeDispatcher) {
	s.dispatcher = dispatcher
}

func (s *Service) SetRedisClient(redisClient *redis.Client) {
	s.redis = redisClient
}

func (s *Service) SetSearchIndexer(indexer SearchIndexer) {
	s.search = indexer
}

func (s *Service) CreateUploadCredential(ctx context.Context, userID int64, req CreateUploadCredentialRequest) (*UploadCredential, error) {
	if userID <= 0 {
		return nil, errno.ErrUnauthorized
	}
	if s.oss == nil {
		return nil, errno.ErrInternalRPC
	}

	objectKey := strings.TrimSpace(req.ObjectKey)
	if objectKey == "" {
		objectKey = buildObjectKey(userID, s.ids.NewID(), req.FileName)
	}
	contentType := normalizeUploadContentType(req.ContentType, req.FileName)
	uploadToken, err := s.oss.GeneratePutSignedURL(ctx, objectKey, contentType, s.cfg.OSSUploadExpire)
	if err != nil {
		return nil, errno.ErrInternalRPC
	}

	return &UploadCredential{
		ObjectKey:    objectKey,
		UploadURL:    uploadToken,
		UploadMethod: "TOKEN",
		ContentType:  contentType,
		ExpiresAt:    time.Now().Add(s.cfg.OSSUploadExpire),
		UploadToken:  uploadToken,
	}, nil
}

func (s *Service) PublishVideo(ctx context.Context, userID int64, req PublishVideoRequest) (*PublishResult, error) {
	if userID <= 0 {
		return nil, errno.ErrUnauthorized
	}
	if s.oss == nil {
		return nil, errno.ErrInternalRPC
	}

	objectKey := strings.TrimSpace(req.ObjectKey)
	title := strings.TrimSpace(req.Title)
	if objectKey == "" || title == "" {
		return nil, errno.ErrInvalidParam
	}
	if req.AllowComment != 0 && req.AllowComment != 1 {
		return nil, errno.ErrInvalidParam
	}
	if req.Visibility != model.VisibilityPrivate && req.Visibility != model.VisibilityPublic {
		return nil, errno.ErrInvalidParam
	}

	exists, err := s.oss.ObjectExists(ctx, objectKey)
	if err != nil {
		return nil, errno.ErrInternalRPC
	}
	if !exists {
		return nil, errno.ErrUploadObjectNotFound
	}

	hashtagIDs, err := s.resolveHashtagIDs(ctx, req.HashtagIDs, req.HashtagNames)
	if err != nil {
		return nil, err
	}

	if len(hashtagIDs) > 0 {
		count, err := s.repo.CountHashtagsByIDs(ctx, hashtagIDs)
		if err != nil {
			return nil, errno.ErrInternalRPC
		}
		if count != int64(len(hashtagIDs)) {
			return nil, errno.ErrHashtagNotFound
		}
	}

	videoID := s.ids.NewID()
	video := &model.Video{
		ID:              videoID,
		UserID:          userID,
		ObjectKey:       objectKey,
		SourceURL:       s.oss.ObjectURL(objectKey),
		Title:           title,
		AllowComment:    req.AllowComment,
		Visibility:      req.Visibility,
		TranscodeStatus: model.TranscodePending,
		AuditStatus:     model.AuditPassed,
	}

	if err := s.repo.CreateVideo(ctx, video, hashtagIDs); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, errno.ErrDuplicateRequest
		}
		return nil, errno.ErrInternalRPC
	}

	if s.dispatcher != nil {
		if err := s.dispatcher.Dispatch(ctx, videoID); err != nil {
			return nil, errno.ErrInternalRPC
		}
	}
	if s.search != nil {
		_ = s.search.UpsertVideoDocument(ctx, videoID)
	}

	return &PublishResult{
		VideoID:         videoID,
		ObjectKey:       objectKey,
		SourceURL:       video.SourceURL,
		TranscodeStatus: model.TranscodePending,
	}, nil
}

func (s *Service) GetVideoDetail(ctx context.Context, viewerUserID, videoID int64) (*VideoDetailResult, error) {
	video, err := s.repo.GetVideoByID(ctx, videoID)
	if err != nil {
		if err == model.ErrVideoNotFound {
			return nil, errno.ErrResourceNotFound
		}
		return nil, errno.ErrInternalRPC
	}

	if err := validateVideoViewPermission(viewerUserID, video); err != nil {
		return nil, err
	}

	return s.buildVideoDetailResult(ctx, video), nil
}

func (s *Service) GetVideoResources(ctx context.Context, viewerUserID, videoID int64) (*VideoResourceListResult, error) {
	video, err := s.repo.GetVideoByID(ctx, videoID)
	if err != nil {
		if err == model.ErrVideoNotFound {
			return nil, errno.ErrResourceNotFound
		}
		return nil, errno.ErrInternalRPC
	}

	if err := validateVideoViewPermission(viewerUserID, video); err != nil {
		return nil, err
	}
	if video.TranscodeStatus != model.TranscodeSuccess {
		if video.TranscodeStatus == model.TranscodeFailed {
			return nil, errno.ErrVideoTranscodeFailed
		}
		return nil, errno.ErrVideoTranscoding
	}

	resources, err := s.repo.ListVideoResources(ctx, videoID)
	if err != nil {
		return nil, errno.ErrInternalRPC
	}

	items := make([]VideoResourceResult, 0, len(resources))
	for _, item := range resources {
		items = append(items, VideoResourceResult{
			VideoID:    item.VideoID,
			Resolution: item.Resolution,
			FileURL:    s.buildResourceReadURL(ctx, video.ObjectKey, item.Resolution, item.FileURL),
			FileSize:   item.FileSize,
			Bitrate:    item.Bitrate,
			CreatedAt:  item.CreatedAt,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		leftRank := resolutionRank(items[i].Resolution)
		rightRank := resolutionRank(items[j].Resolution)
		if leftRank != rightRank {
			return leftRank > rightRank
		}
		if items[i].Bitrate != items[j].Bitrate {
			return items[i].Bitrate > items[j].Bitrate
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})

	return &VideoResourceListResult{Items: items}, nil
}

func (s *Service) ReportPlay(ctx context.Context, userID int64, req ReportPlayRequest) error {
	if userID <= 0 || req.VideoID <= 0 {
		return errno.ErrInvalidParam
	}

	video, err := s.repo.GetVideoByID(ctx, req.VideoID)
	if err != nil {
		if err == model.ErrVideoNotFound {
			return errno.ErrResourceNotFound
		}
		return errno.ErrInternalRPC
	}
	if err := validateVideoViewPermission(userID, video); err != nil {
		return err
	}

	if s.redis != nil {
		dedupeKey := rediskey.VideoPlayReported(userID, req.VideoID)
		added, redisErr := s.redis.SetNX(ctx, dedupeKey, "1", 30*time.Minute).Result()
		if redisErr != nil {
			log.Printf("video report play dedupe failed: user_id=%d video_id=%d err=%v", userID, req.VideoID, redisErr)
		} else if !added {
			return nil
		}
	}

	if err := s.repo.IncreasePlayCount(ctx, req.VideoID); err != nil {
		if err == model.ErrVideoNotFound {
			return errno.ErrResourceNotFound
		}
		return errno.ErrInternalRPC
	}
	return nil
}

func (s *Service) ListUserVideos(ctx context.Context, viewerUserID, targetUserID int64, req ListUserVideosRequest) (*UserVideoListResult, error) {
	if targetUserID <= 0 {
		return nil, errno.ErrInvalidParam
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	includeInvisible := viewerUserID > 0 && viewerUserID == targetUserID
	videos, err := s.repo.ListVideosByUser(ctx, targetUserID, req.Cursor, limit, includeInvisible)
	if err != nil {
		return nil, errno.ErrInternalRPC
	}

	items := make([]VideoDetailResult, 0, len(videos))
	for _, video := range videos {
		item := s.buildVideoDetailResult(ctx, &video)
		items = append(items, *item)
	}

	var nextCursor *time.Time
	if len(videos) == limit {
		lastCreatedAt := videos[len(videos)-1].CreatedAt
		nextCursor = &lastCreatedAt
	}

	return &UserVideoListResult{
		Items:      items,
		NextCursor: nextCursor,
	}, nil
}

func (s *Service) GetVideoForTranscode(ctx context.Context, videoID int64) (*VideoDetailResult, error) {
	video, err := s.repo.GetVideoByID(ctx, videoID)
	if err != nil {
		return nil, err
	}
	return s.buildVideoDetailResult(ctx, video), nil
}

func (s *Service) StartTranscode(ctx context.Context, videoID int64) error {
	return s.repo.MarkVideoTranscoding(ctx, videoID)
}

func (s *Service) CompleteTranscode(ctx context.Context, req CompleteTranscodeRequest) error {
	if req.VideoID <= 0 || req.DurationMS < 0 {
		return errno.ErrInvalidParam
	}

	resources := make([]model.VideoResource, 0, len(req.Resources))
	for _, item := range req.Resources {
		resources = append(resources, model.VideoResource{
			VideoID:    req.VideoID,
			Resolution: strings.TrimSpace(item.Resolution),
			FileURL:    strings.TrimSpace(item.FileURL),
			FileSize:   item.FileSize,
			Bitrate:    item.Bitrate,
		})
	}

	if err := s.repo.MarkVideoTranscodeSuccess(
		ctx,
		req.VideoID,
		strings.TrimSpace(req.CoverURL),
		req.DurationMS,
		resources,
	); err != nil {
		return err
	}
	if s.search != nil {
		_ = s.search.UpsertVideoDocument(ctx, req.VideoID)
	}
	return nil
}

func (s *Service) FailTranscode(ctx context.Context, req FailTranscodeRequest) error {
	if req.VideoID <= 0 {
		return errno.ErrInvalidParam
	}
	return s.repo.MarkVideoTranscodeFailed(ctx, req.VideoID, strings.TrimSpace(req.FailReason))
}

func (s *Service) SaveDraft(ctx context.Context, userID int64, req SaveDraftRequest) (*DraftResult, error) {
	if userID <= 0 {
		return nil, errno.ErrUnauthorized
	}
	if req.AllowComment != 0 && req.AllowComment != 1 {
		return nil, errno.ErrInvalidParam
	}
	if req.Visibility != model.VisibilityPrivate && req.Visibility != model.VisibilityPublic {
		return nil, errno.ErrInvalidParam
	}

	trimmedObjectKey := strings.TrimSpace(req.ObjectKey)
	trimmedCoverURL := s.normalizeDraftCoverReference(trimmedObjectKey, req.CoverURL)

	draft := &model.Draft{
		ID:           req.DraftID,
		UserID:       userID,
		ObjectKey:    trimmedObjectKey,
		CoverURL:     trimmedCoverURL,
		Title:        strings.TrimSpace(req.Title),
		TagNames:     strings.TrimSpace(req.TagNames),
		AllowComment: req.AllowComment,
		Visibility:   req.Visibility,
	}

	if draft.ID > 0 {
		if err := s.repo.UpdateDraft(ctx, draft); err != nil {
			if err == model.ErrDraftNotFound {
				return nil, errno.ErrDraftNotFound
			}
			return nil, errno.ErrInternalRPC
		}
	} else {
		if err := s.repo.CreateDraft(ctx, draft); err != nil {
			return nil, errno.ErrInternalRPC
		}
	}

	return s.buildDraftResult(ctx, draft), nil
}

func (s *Service) GetDraft(ctx context.Context, userID, draftID int64) (*DraftResult, error) {
	if userID <= 0 || draftID <= 0 {
		return nil, errno.ErrInvalidParam
	}

	draft, err := s.repo.GetDraft(ctx, userID, draftID)
	if err != nil {
		if err == model.ErrDraftNotFound {
			return nil, errno.ErrDraftNotFound
		}
		return nil, errno.ErrInternalRPC
	}

	return s.buildDraftResult(ctx, draft), nil
}

func (s *Service) ListDrafts(ctx context.Context, userID int64) (*DraftListResult, error) {
	if userID <= 0 {
		return nil, errno.ErrUnauthorized
	}

	drafts, err := s.repo.ListDrafts(ctx, userID)
	if err != nil {
		return nil, errno.ErrInternalRPC
	}

	items := make([]DraftResult, 0, len(drafts))
	for _, item := range drafts {
		items = append(items, *s.buildDraftResult(ctx, &item))
	}

	return &DraftListResult{Items: items}, nil
}

func (s *Service) DeleteDraft(ctx context.Context, userID, draftID int64) error {
	if userID <= 0 || draftID <= 0 {
		return errno.ErrInvalidParam
	}

	if err := s.repo.DeleteDraft(ctx, userID, draftID); err != nil {
		if err == model.ErrDraftNotFound {
			return errno.ErrDraftNotFound
		}
		return errno.ErrInternalRPC
	}
	return nil
}

func (s *Service) CreateHashtag(ctx context.Context, userID int64, req CreateHashtagRequest) (*HashtagResult, error) {
	if userID <= 0 {
		return nil, errno.ErrUnauthorized
	}

	name, err := normalizeHashtagName(req.Name)
	if err != nil {
		return nil, err
	}

	hashtag, err := s.repo.CreateHashtag(ctx, name)
	if err != nil {
		return nil, errno.ErrInternalRPC
	}
	if s.search != nil {
		_ = s.search.UpsertHashtagDocument(ctx, hashtag.ID)
	}

	return &HashtagResult{
		ID:        hashtag.ID,
		Name:      hashtag.Name,
		UseCount:  hashtag.UseCount,
		CreatedAt: hashtag.CreatedAt,
	}, nil
}

func (s *Service) GetHashtag(ctx context.Context, hashtagID int64) (*HashtagResult, error) {
	if hashtagID <= 0 {
		return nil, errno.ErrInvalidParam
	}

	hashtag, err := s.repo.GetHashtagByID(ctx, hashtagID)
	if err != nil {
		if err == model.ErrHashtagNotFound {
			return nil, errno.ErrHashtagNotFound
		}
		return nil, errno.ErrInternalRPC
	}

	return &HashtagResult{
		ID:        hashtag.ID,
		Name:      hashtag.Name,
		UseCount:  hashtag.UseCount,
		CreatedAt: hashtag.CreatedAt,
	}, nil
}

func (s *Service) ListHotHashtags(ctx context.Context, req ListHotHashtagsRequest) (*HashtagListResult, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	items, err := s.repo.ListHotHashtags(ctx, limit)
	if err != nil {
		return nil, errno.ErrInternalRPC
	}

	results := make([]HashtagResult, 0, len(items))
	for _, item := range items {
		results = append(results, HashtagResult{
			ID:        item.ID,
			Name:      item.Name,
			UseCount:  item.UseCount,
			CreatedAt: item.CreatedAt,
		})
	}

	return &HashtagListResult{Items: results}, nil
}

func (s *Service) ListHashtagVideos(ctx context.Context, hashtagID int64, req ListHashtagVideosRequest) (*HashtagVideoListResult, error) {
	if hashtagID <= 0 {
		return nil, errno.ErrInvalidParam
	}

	if _, err := s.repo.GetHashtagByID(ctx, hashtagID); err != nil {
		if err == model.ErrHashtagNotFound {
			return nil, errno.ErrHashtagNotFound
		}
		return nil, errno.ErrInternalRPC
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	items, err := s.repo.ListVideosByHashtag(ctx, hashtagID, req.Cursor, limit)
	if err != nil {
		return nil, errno.ErrInternalRPC
	}

	results := make([]HashtagVideoResult, 0, len(items))
	for _, item := range items {
		results = append(results, HashtagVideoResult{
			VideoID:         item.VideoID,
			UserID:          item.UserID,
			Title:           item.Title,
			ObjectKey:       item.ObjectKey,
			SourceURL:       s.buildSourceReadURL(ctx, item.ObjectKey, item.SourceURL),
			CoverURL:        s.buildCoverReadURL(ctx, item.ObjectKey, item.CoverURL),
			Visibility:      item.Visibility,
			TranscodeStatus: item.TranscodeStatus,
			AuditStatus:     item.AuditStatus,
			CreatedAt:       item.CreatedAt,
		})
	}

	var nextCursor *time.Time
	if len(items) == limit {
		lastCreatedAt := items[len(items)-1].CreatedAt
		nextCursor = &lastCreatedAt
	}

	return &HashtagVideoListResult{
		Items:      results,
		NextCursor: nextCursor,
	}, nil
}

func buildObjectKey(userID, objectID int64, fileName string) string {
	ext := filepath.Ext(strings.TrimSpace(fileName))
	if ext == "" {
		ext = ".mp4"
	}
	now := time.Now()
	return fmt.Sprintf("video/source/%d/%04d%02d%02d/%d%s", userID, now.Year(), now.Month(), now.Day(), objectID, strings.ToLower(ext))
}

func normalizeHashtagName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	name = strings.TrimPrefix(name, "#")
	if name == "" || len([]rune(name)) > 64 {
		return "", errno.ErrInvalidParam
	}
	return name, nil
}

func normalizeUploadContentType(raw, fileName string) string {
	contentType := strings.TrimSpace(raw)
	if contentType != "" {
		return contentType
	}

	switch strings.ToLower(filepath.Ext(strings.TrimSpace(fileName))) {
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	case ".m4v":
		return "video/x-m4v"
	default:
		return "application/octet-stream"
	}
}

func validateVideoViewPermission(viewerUserID int64, video *model.Video) error {
	if video == nil {
		return errno.ErrResourceNotFound
	}

	isOwner := viewerUserID > 0 && viewerUserID == video.UserID
	if isOwner {
		return nil
	}

	if video.Visibility != model.VisibilityPublic || video.AuditStatus != model.AuditPassed {
		return errno.ErrVideoInvisible
	}

	switch video.TranscodeStatus {
	case model.TranscodeSuccess:
		return nil
	case model.TranscodeFailed:
		return errno.ErrVideoTranscodeFailed
	default:
		return errno.ErrVideoTranscoding
	}
}

func (s *Service) buildVideoDetailResult(ctx context.Context, video *model.Video) *VideoDetailResult {
	return &VideoDetailResult{
		VideoID:             video.ID,
		UserID:              video.UserID,
		Title:               video.Title,
		ObjectKey:           video.ObjectKey,
		SourceURL:           s.buildSourceReadURL(ctx, video.ObjectKey, video.SourceURL),
		CoverURL:            s.buildCoverReadURL(ctx, video.ObjectKey, video.CoverURL),
		DurationMS:          video.DurationMS,
		AllowComment:        video.AllowComment,
		Visibility:          video.Visibility,
		TranscodeStatus:     video.TranscodeStatus,
		AuditStatus:         video.AuditStatus,
		TranscodeFailReason: video.TranscodeFailReason,
		AuditRemark:         video.AuditRemark,
		PlayCount:           video.PlayCount,
		LikeCount:           video.LikeCount,
		CommentCount:        video.CommentCount,
		FavoriteCount:       video.FavoriteCount,
		CreatedAt:           video.CreatedAt,
		UpdatedAt:           video.UpdatedAt,
	}
}

func (s *Service) buildDraftResult(ctx context.Context, draft *model.Draft) *DraftResult {
	sourceFallback := ""
	if s.oss != nil && strings.TrimSpace(draft.ObjectKey) != "" {
		sourceFallback = s.oss.ObjectURL(draft.ObjectKey)
	}

	return &DraftResult{
		ID:           draft.ID,
		ObjectKey:    draft.ObjectKey,
		SourceURL:    s.buildSourceReadURL(ctx, draft.ObjectKey, sourceFallback),
		CoverURL:     s.buildDraftCoverReadURL(ctx, draft.ObjectKey, draft.CoverURL),
		Title:        draft.Title,
		TagNames:     draft.TagNames,
		AllowComment: draft.AllowComment,
		Visibility:   draft.Visibility,
		CreatedAt:    draft.CreatedAt,
		UpdatedAt:    draft.UpdatedAt,
	}
}

func (s *Service) buildSourceReadURL(ctx context.Context, objectKey, fallbackURL string) string {
	trimmedObjectKey := strings.TrimSpace(objectKey)
	if trimmedObjectKey == "" {
		return strings.TrimSpace(fallbackURL)
	}
	return s.buildSignedReadURL(ctx, trimmedObjectKey, fallbackURL)
}

func (s *Service) normalizeDraftCoverReference(sourceObjectKey, rawCoverURL string) string {
	trimmedCoverURL := strings.TrimSpace(rawCoverURL)
	if trimmedCoverURL == "" {
		if strings.TrimSpace(sourceObjectKey) == "" {
			return ""
		}
		return s.buildDraftCoverStorageURL(buildDerivedObjectKey(sourceObjectKey, "cover.jpg"))
	}

	if objectKey := s.resolveStorageObjectKey(trimmedCoverURL); objectKey != "" {
		return s.buildDraftCoverStorageURL(objectKey)
	}
	return trimmedCoverURL
}

func (s *Service) buildDraftCoverReadURL(ctx context.Context, sourceObjectKey, storedCoverRef string) string {
	trimmedCoverRef := strings.TrimSpace(storedCoverRef)
	if objectKey := s.resolveStorageObjectKey(trimmedCoverRef); objectKey != "" {
		fallbackURL := ""
		if s.oss != nil {
			fallbackURL = s.oss.ObjectURL(objectKey)
		}
		return s.buildSignedReadURL(ctx, objectKey, fallbackURL)
	}

	if trimmedCoverRef != "" {
		log.Printf("video draft cover uses raw url fallback: source_object_key=%s stored_cover_ref=%s", sourceObjectKey, trimmedCoverRef)
		return trimmedCoverRef
	}

	if strings.TrimSpace(sourceObjectKey) == "" {
		return ""
	}

	derivedObjectKey := buildDerivedObjectKey(sourceObjectKey, "cover.jpg")
	fallbackURL := ""
	if s.oss != nil {
		fallbackURL = s.oss.ObjectURL(derivedObjectKey)
	}
	return s.buildSignedReadURL(ctx, derivedObjectKey, fallbackURL)
}

func (s *Service) buildDraftCoverStorageURL(objectKey string) string {
	trimmedObjectKey := strings.TrimSpace(objectKey)
	if trimmedObjectKey == "" {
		return ""
	}
	if s.oss == nil {
		return trimmedObjectKey
	}
	return s.oss.ObjectURL(trimmedObjectKey)
}

func (s *Service) buildCoverReadURL(ctx context.Context, sourceObjectKey, fallbackURL string) string {
	trimmedFallback := strings.TrimSpace(fallbackURL)
	if strings.TrimSpace(sourceObjectKey) == "" {
		return trimmedFallback
	}
	if trimmedFallback == "" {
		return ""
	}
	return s.buildSignedReadURL(ctx, buildDerivedObjectKey(sourceObjectKey, "cover.jpg"), trimmedFallback)
}

func (s *Service) buildResourceReadURL(ctx context.Context, sourceObjectKey, resolution, fallbackURL string) string {
	trimmedFallback := strings.TrimSpace(fallbackURL)
	trimmedResolution := strings.TrimSpace(resolution)
	if strings.TrimSpace(sourceObjectKey) == "" || trimmedResolution == "" {
		return trimmedFallback
	}
	return s.buildSignedReadURL(ctx, buildDerivedObjectKey(sourceObjectKey, trimmedResolution+".mp4"), trimmedFallback)
}

func (s *Service) buildSignedReadURL(ctx context.Context, objectKey, fallbackURL string) string {
	trimmedFallback := strings.TrimSpace(fallbackURL)
	if s.oss == nil {
		log.Printf("video buildSignedReadURL skipped: object_key=%s reason=oss_nil fallback=%s", objectKey, trimmedFallback)
		return trimmedFallback
	}

	expire := s.cfg.OSSReadExpire
	if expire <= 0 {
		expire = 15 * time.Minute
	}

	signedURL, err := s.oss.GenerateGetSignedURL(ctx, objectKey, expire)
	if err != nil || strings.TrimSpace(signedURL) == "" {
		log.Printf("video buildSignedReadURL fallback: object_key=%s expire=%s fallback=%s signed_url=%s err=%v", objectKey, expire, trimmedFallback, signedURL, err)
		return trimmedFallback
	}
	log.Printf("video buildSignedReadURL success: object_key=%s signed_url=%s", objectKey, signedURL)
	return signedURL
}

func (s *Service) resolveStorageObjectKey(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if !looksLikeAbsoluteURL(trimmed) {
		return strings.Trim(trimmed, "/")
	}
	return s.extractObjectKeyFromStorageURL(trimmed)
}

func (s *Service) extractObjectKeyFromStorageURL(raw string) string {
	parsedURL, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsedURL.Host == "" {
		return ""
	}

	if objectKey := s.extractObjectKeyByOSSObjectURLPrefix(parsedURL); objectKey != "" {
		return objectKey
	}

	storageHost := hostFromURLLike(s.cfg.OSSEndpoint)
	if storageHost == "" || !strings.EqualFold(parsedURL.Host, storageHost) {
		return ""
	}

	objectKey := strings.Trim(strings.TrimSpace(parsedURL.Path), "/")
	bucketName := strings.TrimSpace(s.cfg.OSSBucket)
	if bucketName != "" && strings.HasPrefix(objectKey, bucketName+"/") {
		objectKey = strings.TrimPrefix(objectKey, bucketName+"/")
	}
	return strings.Trim(objectKey, "/")
}

func (s *Service) extractObjectKeyByOSSObjectURLPrefix(parsedURL *url.URL) string {
	if s.oss == nil || parsedURL == nil {
		return ""
	}

	baseURL := strings.TrimRight(strings.TrimSpace(s.oss.ObjectURL("")), "/")
	if baseURL == "" {
		return ""
	}

	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil || parsedBaseURL.Host == "" {
		return ""
	}
	if !strings.EqualFold(parsedURL.Host, parsedBaseURL.Host) {
		return ""
	}

	basePath := strings.TrimRight(parsedBaseURL.Path, "/")
	requestPath := parsedURL.Path
	if basePath != "" {
		prefix := basePath + "/"
		if !strings.HasPrefix(requestPath, prefix) {
			return ""
		}
		requestPath = strings.TrimPrefix(requestPath, prefix)
	} else {
		requestPath = strings.TrimPrefix(requestPath, "/")
	}

	return strings.Trim(requestPath, "/")
}

func looksLikeAbsoluteURL(raw string) bool {
	parsedURL, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && parsedURL.Scheme != "" && parsedURL.Host != ""
}

func hostFromURLLike(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}
	parsedURL, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	return parsedURL.Host
}

func buildDerivedObjectKey(sourceObjectKey, fileName string) string {
	trimmed := strings.Trim(sourceObjectKey, "/")
	if index := strings.LastIndex(trimmed, "."); index >= 0 {
		trimmed = trimmed[:index]
	}
	return trimmed + "/" + fileName
}

func resolutionRank(resolution string) int {
	trimmed := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(resolution)), "p")
	value, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0
	}
	return value
}

func (s *Service) resolveHashtagIDs(ctx context.Context, inputIDs []int64, inputNames []string) ([]int64, error) {
	idSet := make(map[int64]struct{}, len(inputIDs))
	for _, hashtagID := range inputIDs {
		if hashtagID <= 0 {
			return nil, errno.ErrInvalidParam
		}
		idSet[hashtagID] = struct{}{}
	}

	for _, rawName := range inputNames {
		name, err := normalizeHashtagName(rawName)
		if err != nil {
			return nil, err
		}
		hashtag, err := s.repo.CreateHashtag(ctx, name)
		if err != nil {
			return nil, errno.ErrInternalRPC
		}
		idSet[hashtag.ID] = struct{}{}
	}

	if len(idSet) == 0 {
		return nil, nil
	}

	hashtagIDs := make([]int64, 0, len(idSet))
	for hashtagID := range idSet {
		hashtagIDs = append(hashtagIDs, hashtagID)
	}
	sort.Slice(hashtagIDs, func(i, j int) bool { return hashtagIDs[i] < hashtagIDs[j] })
	return hashtagIDs, nil
}
