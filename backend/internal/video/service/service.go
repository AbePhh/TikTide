package service

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/AbePhh/TikTide/backend/internal/video/model"
	"github.com/AbePhh/TikTide/backend/pkg/config"
	"github.com/AbePhh/TikTide/backend/pkg/errno"
	"github.com/AbePhh/TikTide/backend/pkg/utils"
)

// OSSClient 定义视频模块依赖的 OSS 能力。
type OSSClient interface {
	GeneratePutSignedURL(ctx context.Context, objectKey string, expire time.Duration) (string, error)
	ObjectExists(ctx context.Context, objectKey string) (bool, error)
	ObjectURL(objectKey string) string
}

// VideoService 定义视频发布相关用例。
type VideoService interface {
	CreateUploadCredential(ctx context.Context, userID int64, req CreateUploadCredentialRequest) (*UploadCredential, error)
	PublishVideo(ctx context.Context, userID int64, req PublishVideoRequest) (*PublishResult, error)
	CreateHashtag(ctx context.Context, userID int64, req CreateHashtagRequest) (*HashtagResult, error)
	GetHashtag(ctx context.Context, hashtagID int64) (*HashtagResult, error)
	ListHashtagVideos(ctx context.Context, hashtagID int64, req ListHashtagVideosRequest) (*HashtagVideoListResult, error)
}

// Service 实现视频发布与直传能力。
type Service struct {
	repo model.Repository
	oss  OSSClient
	ids  utils.IDGenerator
	cfg  config.Config
}

// CreateUploadCredentialRequest 表示上传凭证请求。
type CreateUploadCredentialRequest struct {
	FileName string
}

// UploadCredential 表示直传凭证响应。
type UploadCredential struct {
	ObjectKey    string
	UploadURL    string
	UploadMethod string
	ExpiresAt    time.Time
}

// PublishVideoRequest 表示发布视频请求。
type PublishVideoRequest struct {
	ObjectKey    string
	Title        string
	HashtagIDs   []int64
	HashtagNames []string
	AllowComment int8
	Visibility   int8
}

// PublishResult 表示发布视频返回值。
type PublishResult struct {
	VideoID         int64
	ObjectKey       string
	SourceURL       string
	TranscodeStatus int8
}

// HashtagResult 表示话题详情。
type HashtagResult struct {
	ID        int64
	Name      string
	UseCount  int64
	CreatedAt time.Time
}

// CreateHashtagRequest 表示直接创建话题请求。
type CreateHashtagRequest struct {
	Name string
}

// HashtagVideoResult 表示话题下的视频摘要。
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

// ListHashtagVideosRequest 表示话题视频列表查询条件。
type ListHashtagVideosRequest struct {
	Cursor *time.Time
	Limit  int
}

// HashtagVideoListResult 表示话题视频列表结果。
type HashtagVideoListResult struct {
	Items      []HashtagVideoResult
	NextCursor *time.Time
}

// New 创建视频服务。
func New(repo model.Repository, oss OSSClient, ids utils.IDGenerator, cfg config.Config) *Service {
	return &Service{
		repo: repo,
		oss:  oss,
		ids:  ids,
		cfg:  cfg,
	}
}

// CreateUploadCredential 创建阿里云 OSS 直传凭证。
func (s *Service) CreateUploadCredential(ctx context.Context, userID int64, req CreateUploadCredentialRequest) (*UploadCredential, error) {
	if userID <= 0 {
		return nil, errno.ErrUnauthorized
	}
	if s.oss == nil {
		return nil, errno.ErrInternalRPC
	}

	objectKey := buildObjectKey(userID, s.ids.NewID(), req.FileName)
	uploadURL, err := s.oss.GeneratePutSignedURL(ctx, objectKey, s.cfg.OSSUploadExpire)
	if err != nil {
		return nil, errno.ErrInternalRPC
	}

	return &UploadCredential{
		ObjectKey:    objectKey,
		UploadURL:    uploadURL,
		UploadMethod: "PUT",
		ExpiresAt:    time.Now().Add(s.cfg.OSSUploadExpire),
	}, nil
}

// PublishVideo 创建视频元数据并校验 OSS 对象是否存在。
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

	return &PublishResult{
		VideoID:         videoID,
		ObjectKey:       objectKey,
		SourceURL:       video.SourceURL,
		TranscodeStatus: model.TranscodePending,
	}, nil
}

// CreateHashtag 直接创建话题，若已存在则返回已有话题。
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

	return &HashtagResult{
		ID:        hashtag.ID,
		Name:      hashtag.Name,
		UseCount:  hashtag.UseCount,
		CreatedAt: hashtag.CreatedAt,
	}, nil
}

// GetHashtag 获取话题详情。
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

// ListHashtagVideos 获取话题下的视频列表。
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
			SourceURL:       item.SourceURL,
			CoverURL:        item.CoverURL,
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
