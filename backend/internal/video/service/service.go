package service

import (
	"context"
	"fmt"
	"path/filepath"
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

	if len(req.HashtagIDs) > 0 {
		count, err := s.repo.CountHashtagsByIDs(ctx, req.HashtagIDs)
		if err != nil {
			return nil, errno.ErrInternalRPC
		}
		if count != int64(len(req.HashtagIDs)) {
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

	if err := s.repo.CreateVideo(ctx, video, req.HashtagIDs); err != nil {
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

func buildObjectKey(userID, objectID int64, fileName string) string {
	ext := filepath.Ext(strings.TrimSpace(fileName))
	if ext == "" {
		ext = ".mp4"
	}
	now := time.Now()
	return fmt.Sprintf("video/source/%d/%04d%02d%02d/%d%s", userID, now.Year(), now.Month(), now.Day(), objectID, strings.ToLower(ext))
}
