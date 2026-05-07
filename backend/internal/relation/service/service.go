package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	messagemodel "github.com/AbePhh/TikTide/backend/internal/message/model"
	messageservice "github.com/AbePhh/TikTide/backend/internal/message/service"
	relationmodel "github.com/AbePhh/TikTide/backend/internal/relation/model"
	usermodel "github.com/AbePhh/TikTide/backend/internal/user/model"
	"github.com/AbePhh/TikTide/backend/pkg/errno"
)

const (
	// ActionFollow 表示执行关注操作。
	ActionFollow int8 = 1
	// ActionUnfollow 表示执行取关操作。
	ActionUnfollow int8 = 2
)

// UserRepository 定义关注模块依赖的用户查询能力。
type UserRepository interface {
	GetByID(ctx context.Context, userID int64) (*usermodel.User, error)
	GetStatsByID(ctx context.Context, userID int64) (*usermodel.UserStats, error)
}

// RelationService 定义关注关系模块对外暴露的业务能力。
type RelationService interface {
	Action(ctx context.Context, actorUserID int64, req ActionRequest) (*ActionResult, error)
	ListFollowing(ctx context.Context, viewerUserID, targetUserID int64, req ListRequest) (*UserListResult, error)
	ListFollowers(ctx context.Context, viewerUserID, targetUserID int64, req ListRequest) (*UserListResult, error)
	GetRelationState(ctx context.Context, viewerUserID, targetUserID int64) (RelationState, error)
}

type MessageNotifier interface {
	CreateMessage(ctx context.Context, req messageservice.CreateMessageRequest) error
}

// Service 实现关注关系模块的业务规则。
type Service struct {
	repo     relationmodel.Repository
	userRepo UserRepository
	message  MessageNotifier
}

// ActionRequest 表示关注操作请求。
type ActionRequest struct {
	ToUserID   int64
	ActionType int8
}

// ActionResult 表示关注操作结果。
type ActionResult struct {
	ToUserID   int64
	IsFollowed bool
	IsMutual   bool
	FollowedAt *time.Time
	ActionType int8
}

// ListRequest 表示关注列表分页请求。
type ListRequest struct {
	Cursor int64
	Limit  int
}

// RelationState 表示当前登录用户与目标用户的关注态。
type RelationState struct {
	IsFollowed bool
	IsMutual   bool
}

// UserSummary 表示关注列表中的用户摘要。
type UserSummary struct {
	ID            int64
	Username      string
	Nickname      string
	AvatarURL     string
	Signature     string
	Gender        int8
	Status        int8
	FollowCount   int64
	FollowerCount int64
	IsFollowed    bool
	IsMutual      bool
	CreatedAt     time.Time
}

// UserListResult 表示关注列表查询结果。
type UserListResult struct {
	Items      []UserSummary
	NextCursor string
}

// New 创建关注关系服务。
func New(repo relationmodel.Repository, userRepo UserRepository, message MessageNotifier) *Service {
	return &Service{
		repo:     repo,
		userRepo: userRepo,
		message:  message,
	}
}

// Action 处理关注或取关请求。
func (s *Service) Action(ctx context.Context, actorUserID int64, req ActionRequest) (*ActionResult, error) {
	if actorUserID <= 0 || req.ToUserID <= 0 || actorUserID == req.ToUserID {
		return nil, errno.ErrInvalidParam
	}

	if _, err := s.ensureActiveUser(ctx, actorUserID); err != nil {
		return nil, err
	}
	if _, err := s.ensureActiveUser(ctx, req.ToUserID); err != nil {
		return nil, err
	}

	switch req.ActionType {
	case ActionFollow:
		relation, err := s.repo.Create(ctx, actorUserID, req.ToUserID)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "unique") {
				return nil, errno.ErrDuplicateFollow
			}
			return nil, errno.ErrInternalRPC
		}

		followedAt := relation.CreatedAt
		s.notifyFollow(ctx, actorUserID, req.ToUserID)
		return &ActionResult{
			ToUserID:   req.ToUserID,
			IsFollowed: true,
			IsMutual:   relation.IsMutual,
			FollowedAt: &followedAt,
			ActionType: req.ActionType,
		}, nil
	case ActionUnfollow:
		if err := s.repo.Delete(ctx, actorUserID, req.ToUserID); err != nil {
			if errors.Is(err, relationmodel.ErrRelationNotFound) {
				return nil, errno.ErrRelationNotFound
			}
			return nil, errno.ErrInternalRPC
		}

		return &ActionResult{
			ToUserID:   req.ToUserID,
			IsFollowed: false,
			IsMutual:   false,
			ActionType: req.ActionType,
		}, nil
	default:
		return nil, errno.ErrInvalidParam
	}
}

func (s *Service) notifyFollow(ctx context.Context, senderID, receiverID int64) {
	if s.message == nil || senderID <= 0 || receiverID <= 0 || senderID == receiverID {
		return
	}
	_ = s.message.CreateMessage(ctx, messageservice.CreateMessageRequest{
		ReceiverID: receiverID,
		SenderID:   senderID,
		Type:       messagemodel.MessageTypeNewFollower,
		RelatedID:  senderID,
		Content:    "你收到了一个新关注",
	})
}

// ListFollowing 查询用户关注列表。
func (s *Service) ListFollowing(ctx context.Context, viewerUserID, targetUserID int64, req ListRequest) (*UserListResult, error) {
	if viewerUserID <= 0 || targetUserID <= 0 {
		return nil, errno.ErrInvalidParam
	}
	if _, err := s.ensureActiveUser(ctx, targetUserID); err != nil {
		return nil, err
	}

	relations, limit, err := s.listRelations(ctx, func(ctx context.Context, cursor int64, limit int) ([]relationmodel.Relation, error) {
		return s.repo.ListFollowing(ctx, targetUserID, cursor, limit)
	}, req)
	if err != nil {
		return nil, err
	}

	items, nextCursor, err := s.buildUserList(ctx, viewerUserID, relations, limit, true)
	if err != nil {
		return nil, err
	}

	return &UserListResult{
		Items:      items,
		NextCursor: nextCursor,
	}, nil
}

// ListFollowers 查询用户粉丝列表。
func (s *Service) ListFollowers(ctx context.Context, viewerUserID, targetUserID int64, req ListRequest) (*UserListResult, error) {
	if viewerUserID <= 0 || targetUserID <= 0 {
		return nil, errno.ErrInvalidParam
	}
	if _, err := s.ensureActiveUser(ctx, targetUserID); err != nil {
		return nil, err
	}

	relations, limit, err := s.listRelations(ctx, func(ctx context.Context, cursor int64, limit int) ([]relationmodel.Relation, error) {
		return s.repo.ListFollowers(ctx, targetUserID, cursor, limit)
	}, req)
	if err != nil {
		return nil, err
	}

	items, nextCursor, err := s.buildUserList(ctx, viewerUserID, relations, limit, false)
	if err != nil {
		return nil, err
	}

	return &UserListResult{
		Items:      items,
		NextCursor: nextCursor,
	}, nil
}

// GetRelationState 返回当前用户对目标用户的关注状态。
func (s *Service) GetRelationState(ctx context.Context, viewerUserID, targetUserID int64) (RelationState, error) {
	if viewerUserID <= 0 || targetUserID <= 0 || viewerUserID == targetUserID {
		return RelationState{}, nil
	}

	relation, err := s.repo.Get(ctx, viewerUserID, targetUserID)
	if err != nil {
		if errors.Is(err, relationmodel.ErrRelationNotFound) {
			return RelationState{}, nil
		}
		return RelationState{}, errno.ErrInternalRPC
	}

	return RelationState{
		IsFollowed: true,
		IsMutual:   relation.IsMutual,
	}, nil
}

func (s *Service) ensureActiveUser(ctx context.Context, userID int64) (*usermodel.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, usermodel.ErrUserNotFound) {
			return nil, errno.ErrResourceNotFound
		}
		return nil, errno.ErrInternalRPC
	}
	if user.Status == usermodel.UserStatusBanned {
		return nil, errno.ErrUserBanned
	}
	return user, nil
}

func (s *Service) listRelations(
	ctx context.Context,
	loader func(context.Context, int64, int) ([]relationmodel.Relation, error),
	req ListRequest,
) ([]relationmodel.Relation, int, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	relations, err := loader(ctx, req.Cursor, limit)
	if err != nil {
		return nil, 0, errno.ErrInternalRPC
	}
	return relations, limit, nil
}

func (s *Service) buildUserList(
	ctx context.Context,
	viewerUserID int64,
	relations []relationmodel.Relation,
	limit int,
	isFollowingList bool,
) ([]UserSummary, string, error) {
	items := make([]UserSummary, 0, len(relations))
	for _, relation := range relations {
		relatedUserID := relation.UserID
		if isFollowingList {
			relatedUserID = relation.FollowID
		}

		user, err := s.ensureActiveUser(ctx, relatedUserID)
		if err != nil {
			if errno.IsCode(err, errno.ErrUserBanned.Code) || errno.IsCode(err, errno.ErrResourceNotFound.Code) {
				continue
			}
			return nil, "", err
		}

		stats, err := s.userRepo.GetStatsByID(ctx, relatedUserID)
		if err != nil {
			return nil, "", errno.ErrInternalRPC
		}

		state, err := s.GetRelationState(ctx, viewerUserID, relatedUserID)
		if err != nil {
			return nil, "", err
		}

		items = append(items, UserSummary{
			ID:            user.ID,
			Username:      user.Username,
			Nickname:      user.Nickname,
			AvatarURL:     user.AvatarURL,
			Signature:     user.Signature,
			Gender:        user.Gender,
			Status:        user.Status,
			FollowCount:   stats.FollowCount,
			FollowerCount: stats.FollowerCount,
			IsFollowed:    state.IsFollowed,
			IsMutual:      state.IsMutual,
			CreatedAt:     user.CreatedAt,
		})
	}

	nextCursor := ""
	if len(relations) == limit {
		nextCursor = strconv.FormatInt(relations[len(relations)-1].ID, 10)
	}
	return items, nextCursor, nil
}
