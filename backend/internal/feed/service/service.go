package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	interactmodel "github.com/AbePhh/TikTide/backend/internal/interact/model"
	relationmodel "github.com/AbePhh/TikTide/backend/internal/relation/model"
	relationservice "github.com/AbePhh/TikTide/backend/internal/relation/service"
	usermodel "github.com/AbePhh/TikTide/backend/internal/user/model"
	videoservice "github.com/AbePhh/TikTide/backend/internal/video/service"
	"github.com/AbePhh/TikTide/backend/pkg/errno"
	"github.com/AbePhh/TikTide/backend/pkg/rediskey"
)

type VideoService interface {
	GetVideoDetail(ctx context.Context, viewerUserID, videoID int64) (*videoservice.VideoDetailResult, error)
}

type UserRepository interface {
	GetByID(ctx context.Context, userID int64) (*usermodel.User, error)
}

type RelationService interface {
	GetRelationState(ctx context.Context, viewerUserID, targetUserID int64) (relationservice.RelationState, error)
}

type FeedService interface {
	DistributeVideo(ctx context.Context, videoID, authorUserID int64, createdAt time.Time) error
	ListFollowing(ctx context.Context, userID int64, req ListRequest) (*ListResult, error)
}

type InteractRepository interface {
	HasLikedVideo(ctx context.Context, userID, videoID int64) (bool, error)
	HasFavoritedVideo(ctx context.Context, userID, videoID int64) (bool, error)
}

type Service struct {
	redis         *redis.Client
	relationRepo  relationmodel.Repository
	videoService  VideoService
	userRepo      UserRepository
	relations     RelationService
	interactRepo  InteractRepository
	inboxLimit    int64
	outboxLimit   int64
	inboxTTL      time.Duration
	outboxTTL     time.Duration
	bigVThreshold int
}

type ListRequest struct {
	Cursor int64
	Limit  int
}

type FeedItem struct {
	VideoID      int64
	Detail       videoservice.VideoDetailResult
	AuthorID     int64
	AuthorName   string
	AuthorAvatar string
	AuthorHandle string
	IsFollowed   bool
	IsLiked      bool
	IsFavorited  bool
}

type scoredVideo struct {
	VideoID int64
	Score   float64
}

type ListResult struct {
	Items      []FeedItem
	NextCursor string
}

func New(
	redisClient *redis.Client,
	relationRepo relationmodel.Repository,
	videoService VideoService,
	userRepo UserRepository,
	relations RelationService,
	interactRepo interactmodel.Repository,
) *Service {
	return &Service{
		redis:         redisClient,
		relationRepo:  relationRepo,
		videoService:  videoService,
		userRepo:      userRepo,
		relations:     relations,
		interactRepo:  interactRepo,
		inboxLimit:    1000,
		outboxLimit:   1000,
		inboxTTL:      7 * 24 * time.Hour,
		outboxTTL:     7 * 24 * time.Hour,
		bigVThreshold: 1000,
	}
}

func (s *Service) DistributeVideo(ctx context.Context, videoID, authorUserID int64, createdAt time.Time) error {
	if s.redis == nil {
		return nil
	}

	followers, err := s.relationRepo.ListFollowersAll(ctx, authorUserID)
	if err != nil {
		return fmt.Errorf("list followers for feed distribute: %w", err)
	}

	score := float64(createdAt.UnixMilli())
	member := strconv.FormatInt(videoID, 10)
	if len(followers) >= s.bigVThreshold {
		return s.writeOutbox(ctx, authorUserID, member, score)
	}
	for _, follower := range followers {
		key := rediskey.FeedInbox(follower.UserID)
		if err := s.redis.ZAdd(ctx, key, redis.Z{
			Score:  score,
			Member: member,
		}).Err(); err != nil {
			return fmt.Errorf("zadd feed inbox: %w", err)
		}
		if s.inboxLimit > 0 {
			if err := s.redis.ZRemRangeByRank(ctx, key, 0, -(s.inboxLimit + 1)).Err(); err != nil {
				return fmt.Errorf("trim feed inbox: %w", err)
			}
		}
		if s.inboxTTL > 0 {
			if err := s.redis.Expire(ctx, key, s.inboxTTL).Err(); err != nil {
				return fmt.Errorf("expire feed inbox: %w", err)
			}
		}
	}
	return nil
}

func (s *Service) ListFollowing(ctx context.Context, userID int64, req ListRequest) (*ListResult, error) {
	if userID <= 0 {
		return nil, errno.ErrInvalidParam
	}
	if s.redis == nil {
		return nil, errno.ErrFeedFetchFailed
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	zRange := &redis.ZRangeBy{
		Min:   "-inf",
		Max:   "+inf",
		Count: int64(limit),
	}
	if req.Cursor > 0 {
		zRange.Max = fmt.Sprintf("(%d", req.Cursor)
	}

	inboxValues, err := s.redis.ZRevRangeByScoreWithScores(ctx, rediskey.FeedInbox(userID), zRange).Result()
	if err != nil {
		return nil, errno.ErrFeedFetchFailed
	}

	followings, err := s.relationRepo.ListFollowingAll(ctx, userID)
	if err != nil {
		return nil, errno.ErrFeedFetchFailed
	}

	candidates := make(map[string]float64, len(inboxValues))
	for _, item := range inboxValues {
		member, ok := item.Member.(string)
		if !ok {
			continue
		}
		candidates[member] = item.Score
	}
	for _, relation := range followings {
		outboxValues, outboxErr := s.redis.ZRevRangeByScoreWithScores(ctx, rediskey.FeedOutbox(relation.FollowID), zRange).Result()
		if outboxErr != nil {
			return nil, errno.ErrFeedFetchFailed
		}
		for _, item := range outboxValues {
			member, ok := item.Member.(string)
			if !ok {
				continue
			}
			if currentScore, exists := candidates[member]; !exists || item.Score > currentScore {
				candidates[member] = item.Score
			}
		}
	}

	if len(candidates) == 0 {
		return &ListResult{Items: []FeedItem{}}, nil
	}

	videos := make([]scoredVideo, 0, len(candidates))
	for member, score := range candidates {
		videoID, parseErr := strconv.ParseInt(member, 10, 64)
		if parseErr != nil {
			continue
		}
		videos = append(videos, scoredVideo{
			VideoID: videoID,
			Score:   score,
		})
	}

	sort.Slice(videos, func(i, j int) bool {
		if videos[i].Score == videos[j].Score {
			return videos[i].VideoID > videos[j].VideoID
		}
		return videos[i].Score > videos[j].Score
	})

	items := make([]FeedItem, 0, minInt(len(videos), limit))
	var nextCursor string
	for _, video := range videos {
		detail, detailErr := s.videoService.GetVideoDetail(ctx, userID, video.VideoID)
		if detailErr != nil {
			continue
		}

		items = append(items, FeedItem{
			VideoID: video.VideoID,
			Detail:  *detail,
		})
		if s.userRepo != nil {
			author, authorErr := s.userRepo.GetByID(ctx, detail.UserID)
			if authorErr == nil && author != nil {
				items[len(items)-1].AuthorID = author.ID
				items[len(items)-1].AuthorName = author.Nickname
				items[len(items)-1].AuthorAvatar = author.AvatarURL
				items[len(items)-1].AuthorHandle = author.Username
				if items[len(items)-1].AuthorName == "" {
					items[len(items)-1].AuthorName = author.Username
				}
			}
		}
		if s.relations != nil && userID != detail.UserID {
			state, stateErr := s.relations.GetRelationState(ctx, userID, detail.UserID)
			if stateErr == nil {
				items[len(items)-1].IsFollowed = state.IsFollowed
			}
		}
		if s.interactRepo != nil {
			liked, likeErr := s.interactRepo.HasLikedVideo(ctx, userID, detail.VideoID)
			if likeErr == nil {
				items[len(items)-1].IsLiked = liked
			}
			favorited, favoriteErr := s.interactRepo.HasFavoritedVideo(ctx, userID, detail.VideoID)
			if favoriteErr == nil {
				items[len(items)-1].IsFavorited = favorited
			}
		}
		if len(items) == limit {
			nextCursor = strconv.FormatInt(detail.CreatedAt.UnixMilli(), 10)
			break
		}
	}

	return &ListResult{
		Items:      items,
		NextCursor: nextCursor,
	}, nil
}

func (s *Service) writeOutbox(ctx context.Context, authorUserID int64, member string, score float64) error {
	key := rediskey.FeedOutbox(authorUserID)
	if err := s.redis.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: member,
	}).Err(); err != nil {
		return fmt.Errorf("zadd feed outbox: %w", err)
	}
	if s.outboxLimit > 0 {
		if err := s.redis.ZRemRangeByRank(ctx, key, 0, -(s.outboxLimit + 1)).Err(); err != nil {
			return fmt.Errorf("trim feed outbox: %w", err)
		}
	}
	if s.outboxTTL > 0 {
		if err := s.redis.Expire(ctx, key, s.outboxTTL).Err(); err != nil {
			return fmt.Errorf("expire feed outbox: %w", err)
		}
	}
	return nil
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func (s *Service) SetBigVThresholdForTest(threshold int) {
	s.bigVThreshold = threshold
}
