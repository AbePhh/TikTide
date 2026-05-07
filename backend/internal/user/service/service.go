package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	relationservice "github.com/AbePhh/TikTide/backend/internal/relation/service"
	"github.com/AbePhh/TikTide/backend/internal/user/model"
	"github.com/AbePhh/TikTide/backend/pkg/errno"
	"github.com/AbePhh/TikTide/backend/pkg/jwt"
	"github.com/AbePhh/TikTide/backend/pkg/utils"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)

// UserService 定义用户与鉴权相关用例。
type UserService interface {
	Register(ctx context.Context, req RegisterRequest) (*RegisterResult, error)
	Login(ctx context.Context, req LoginRequest) (*LoginResult, error)
	Logout(ctx context.Context, rawToken string) error
	GetProfile(ctx context.Context, userID int64) (*Profile, error)
	UpdateUsername(ctx context.Context, userID int64, username string) (*Profile, error)
	UpdateProfile(ctx context.Context, userID int64, req UpdateProfileRequest) (*Profile, error)
	ChangePassword(ctx context.Context, userID int64, req ChangePasswordRequest) error
	GetHomepage(ctx context.Context, viewerUserID, targetUserID int64) (*Profile, error)
}

// Service 实现用户与鉴权用例。
type Service struct {
	repo      model.Repository
	relations relationservice.RelationService
	ids       utils.IDGenerator
	jwt       *jwt.Manager
	blocklist jwt.TokenBlacklist
	search    SearchIndexer
}

type SearchIndexer interface {
	UpsertUserDocument(ctx context.Context, userID int64) error
}

// RegisterRequest 表示注册请求。
type RegisterRequest struct {
	Username string
	Password string
}

// RegisterResult 返回注册后的用户摘要信息。
type RegisterResult struct {
	User Profile `json:"user"`
}

// LoginRequest 表示登录请求。
type LoginRequest struct {
	Username string
	Password string
}

// LoginResult 返回访问令牌和用户资料。
type LoginResult struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      Profile   `json:"user"`
}

// UpdateProfileRequest 表示资料更新请求。
type UpdateProfileRequest struct {
	Nickname      *string
	AvatarURL     *string
	Signature     *string
	Gender        *int8
	Birthday      *time.Time
	BirthdayIsSet bool
}

// ChangePasswordRequest 表示修改密码请求。
type ChangePasswordRequest struct {
	OldPassword string
	NewPassword string
}

// Profile 表示接口返回的用户资料与统计信息。
type Profile struct {
	ID              int64      `json:"id"`
	Username        string     `json:"username"`
	Nickname        string     `json:"nickname"`
	AvatarURL       string     `json:"avatar_url"`
	Signature       string     `json:"signature"`
	Gender          int8       `json:"gender"`
	Birthday        *time.Time `json:"birthday,omitempty"`
	Status          int8       `json:"status"`
	FollowCount     int64      `json:"follow_count"`
	FollowerCount   int64      `json:"follower_count"`
	TotalLikedCount int64      `json:"total_liked_count"`
	WorkCount       int64      `json:"work_count"`
	FavoriteCount   int64      `json:"favorite_count"`
	IsFollowed      bool       `json:"is_followed"`
	IsMutual        bool       `json:"is_mutual"`
	CreatedAt       time.Time  `json:"created_at"`
}

// New 创建用户服务。
func New(
	repo model.Repository,
	relations relationservice.RelationService,
	ids utils.IDGenerator,
	jwtManager *jwt.Manager,
	blocklist jwt.TokenBlacklist,
) *Service {
	return &Service{
		repo:      repo,
		relations: relations,
		ids:       ids,
		jwt:       jwtManager,
		blocklist: blocklist,
	}
}

func (s *Service) SetSearchIndexer(indexer SearchIndexer) {
	s.search = indexer
}

// Register 创建用户并初始化统计信息。
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*RegisterResult, error) {
	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	if !usernamePattern.MatchString(username) || len(password) < 8 || len(password) > 72 {
		return nil, errno.ErrInvalidParam
	}

	_, err := s.repo.GetByUsername(ctx, username)
	if err == nil {
		return nil, errno.ErrUsernameExists
	}
	if err != nil && !errors.Is(err, model.ErrUserNotFound) {
		return nil, errno.ErrInternalRPC
	}

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, errno.ErrInternalRPC
	}

	userID := s.ids.NewID()
	user := &model.User{
		ID:           userID,
		Username:     username,
		PasswordHash: hashedPassword,
		Nickname:     username,
		Status:       model.UserStatusActive,
	}
	stats := &model.UserStats{ID: userID}

	if err := s.repo.Create(ctx, user, stats); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, errno.ErrUsernameExists
		}
		return nil, errno.ErrInternalRPC
	}

	profile, err := s.GetProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	if s.search != nil {
		_ = s.search.UpsertUserDocument(ctx, userID)
	}
	return &RegisterResult{User: *profile}, nil
}

// Login 校验账号密码并签发 JWT。
func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResult, error) {
	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	if username == "" || password == "" {
		return nil, errno.ErrInvalidParam
	}

	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return nil, errno.ErrInvalidCredential
		}
		return nil, errno.ErrInternalRPC
	}
	if user.Status == model.UserStatusBanned {
		return nil, errno.ErrUserBanned
	}
	if err := utils.ComparePassword(user.PasswordHash, password); err != nil {
		return nil, errno.ErrInvalidCredential
	}

	token, expiresAt, err := s.jwt.IssueToken(user.ID, user.Username)
	if err != nil {
		return nil, errno.ErrInternalRPC
	}

	profile, err := s.GetProfile(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      *profile,
	}, nil
}

// Logout 将当前 JWT 加入黑名单直到自然过期。
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if strings.TrimSpace(rawToken) == "" {
		return errno.ErrUnauthorized
	}

	claims, err := s.jwt.ParseToken(rawToken)
	if err != nil {
		return errno.ErrUnauthorized
	}
	if err := s.blocklist.Add(ctx, rawToken, claims.ExpiresAt.Time); err != nil {
		return errno.ErrInternalRPC
	}
	return nil
}

func (s *Service) UpdateUsername(ctx context.Context, userID int64, username string) (*Profile, error) {
	trimmedUsername := strings.TrimSpace(username)
	if !usernamePattern.MatchString(trimmedUsername) {
		return nil, errno.ErrInvalidParam
	}

	currentUser, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return nil, errno.ErrResourceNotFound
		}
		return nil, errno.ErrInternalRPC
	}
	if currentUser.Status == model.UserStatusBanned {
		return nil, errno.ErrUserBanned
	}
	if currentUser.Username == trimmedUsername {
		return s.GetProfile(ctx, userID)
	}

	existingUser, err := s.repo.GetByUsername(ctx, trimmedUsername)
	if err == nil && existingUser.ID != userID {
		return nil, errno.ErrUsernameExists
	}
	if err != nil && !errors.Is(err, model.ErrUserNotFound) {
		return nil, errno.ErrInternalRPC
	}

	if err := s.repo.UpdateUsername(ctx, userID, trimmedUsername); err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return nil, errno.ErrResourceNotFound
		}
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, errno.ErrUsernameExists
		}
		return nil, errno.ErrInternalRPC
	}
	if s.search != nil {
		_ = s.search.UpsertUserDocument(ctx, userID)
	}
	return s.GetProfile(ctx, userID)
}

// GetProfile 返回当前用户的资料与统计信息。
func (s *Service) GetProfile(ctx context.Context, userID int64) (*Profile, error) {
	user, stats, err := s.loadUserWithStats(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.Status == model.UserStatusBanned {
		return nil, errno.ErrUserBanned
	}

	profile := buildProfile(user, stats)
	return &profile, nil
}

// UpdateProfile 更新用户资料。
func (s *Service) UpdateProfile(ctx context.Context, userID int64, req UpdateProfileRequest) (*Profile, error) {
	if req.Gender != nil && (*req.Gender < 0 || *req.Gender > 2) {
		return nil, errno.ErrInvalidParam
	}

	update := model.ProfileUpdate{
		Nickname:      sanitizeOptional(req.Nickname),
		AvatarURL:     sanitizeOptional(req.AvatarURL),
		Signature:     sanitizeOptional(req.Signature),
		Gender:        req.Gender,
		Birthday:      req.Birthday,
		BirthdayIsSet: req.BirthdayIsSet,
	}

	if err := s.repo.UpdateProfile(ctx, userID, update); err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return nil, errno.ErrResourceNotFound
		}
		return nil, errno.ErrInternalRPC
	}
	if s.search != nil {
		_ = s.search.UpsertUserDocument(ctx, userID)
	}
	return s.GetProfile(ctx, userID)
}

// ChangePassword 在校验旧密码后修改密码。
func (s *Service) ChangePassword(ctx context.Context, userID int64, req ChangePasswordRequest) error {
	if strings.TrimSpace(req.OldPassword) == "" || len(strings.TrimSpace(req.NewPassword)) < 8 || len(strings.TrimSpace(req.NewPassword)) > 72 {
		return errno.ErrInvalidParam
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return errno.ErrResourceNotFound
		}
		return errno.ErrInternalRPC
	}
	if user.Status == model.UserStatusBanned {
		return errno.ErrUserBanned
	}
	if err := utils.ComparePassword(user.PasswordHash, req.OldPassword); err != nil {
		return errno.ErrWrongOldPassword
	}

	newHash, err := utils.HashPassword(strings.TrimSpace(req.NewPassword))
	if err != nil {
		return errno.ErrInternalRPC
	}
	if err := s.repo.UpdatePassword(ctx, userID, newHash); err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return errno.ErrResourceNotFound
		}
		return errno.ErrInternalRPC
	}
	return nil
}

// GetHomepage 返回目标用户主页信息。
func (s *Service) GetHomepage(ctx context.Context, viewerUserID, targetUserID int64) (*Profile, error) {
	user, stats, err := s.loadUserWithStats(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if user.Status == model.UserStatusBanned {
		return nil, errno.ErrUserBanned
	}

	profile := buildProfile(user, stats)
	if viewerUserID == targetUserID {
		return &profile, nil
	}

	if s.relations != nil {
		state, err := s.relations.GetRelationState(ctx, viewerUserID, targetUserID)
		if err != nil {
			return nil, err
		}
		profile.IsFollowed = state.IsFollowed
		profile.IsMutual = state.IsMutual
	}

	return &profile, nil
}

func (s *Service) loadUserWithStats(ctx context.Context, userID int64) (*model.User, *model.UserStats, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return nil, nil, errno.ErrResourceNotFound
		}
		return nil, nil, errno.ErrInternalRPC
	}

	stats, err := s.repo.GetStatsByID(ctx, userID)
	if err != nil {
		return nil, nil, errno.ErrInternalRPC
	}

	return user, stats, nil
}

func buildProfile(user *model.User, stats *model.UserStats) Profile {
	return Profile{
		ID:              user.ID,
		Username:        user.Username,
		Nickname:        user.Nickname,
		AvatarURL:       user.AvatarURL,
		Signature:       user.Signature,
		Gender:          user.Gender,
		Birthday:        user.Birthday,
		Status:          user.Status,
		FollowCount:     stats.FollowCount,
		FollowerCount:   stats.FollowerCount,
		TotalLikedCount: stats.TotalLikedCount,
		WorkCount:       stats.WorkCount,
		FavoriteCount:   stats.FavoriteCount,
		CreatedAt:       user.CreatedAt,
	}
}

func sanitizeOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}
