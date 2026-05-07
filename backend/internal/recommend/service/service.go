package service

import (
	"context"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/redis/go-redis/v9"

	interactmodel "github.com/AbePhh/TikTide/backend/internal/interact/model"
	relationservice "github.com/AbePhh/TikTide/backend/internal/relation/service"
	usermodel "github.com/AbePhh/TikTide/backend/internal/user/model"
	videomodel "github.com/AbePhh/TikTide/backend/internal/video/model"
	videoservice "github.com/AbePhh/TikTide/backend/internal/video/service"
	"github.com/AbePhh/TikTide/backend/pkg/errno"
	"github.com/AbePhh/TikTide/backend/pkg/rediskey"
)

const (
	defaultCandidateLimit = 500
	defaultActionLimit    = 200
	defaultCacheTTL       = 10 * time.Minute
	defaultSeenTTL        = 24 * time.Hour
	defaultSeenMaxSize    = 2000
	recallStageFreshOnly  = 0
	recallStageOlder12h   = 1
	recallStageOlder6h    = 2
	recallStageAllSeen    = 3
)

type VideoRepository interface {
	ListVideosByIDs(ctx context.Context, videoIDs []int64) ([]videomodel.Video, error)
	ListRecommendVideos(ctx context.Context, limit int) ([]videomodel.Video, error)
	ListHashtagNamesByVideoIDs(ctx context.Context, videoIDs []int64) (map[int64][]string, error)
}

type InteractRepository interface {
	ListUserVideoActions(ctx context.Context, userID int64, limit int) ([]interactmodel.UserVideoAction, error)
	HasLikedVideo(ctx context.Context, userID, videoID int64) (bool, error)
	HasFavoritedVideo(ctx context.Context, userID, videoID int64) (bool, error)
}

type VideoService interface {
	GetVideoDetail(ctx context.Context, viewerUserID, videoID int64) (*videoservice.VideoDetailResult, error)
}

type UserRepository interface {
	GetByID(ctx context.Context, userID int64) (*usermodel.User, error)
}

type RelationService interface {
	GetRelationState(ctx context.Context, viewerUserID, targetUserID int64) (relationservice.RelationState, error)
}

type RecommendService interface {
	ListRecommend(ctx context.Context, userID int64, req ListRequest) (*ListResult, error)
}

type Service struct {
	redis           *redis.Client
	videoRepo       VideoRepository
	interactRepo    InteractRepository
	videoService    VideoService
	userRepo        UserRepository
	relationService RelationService
	cacheTTL        time.Duration
	seenTTL         time.Duration
	seenMaxSize     int64
	candidateLimit  int
	userActionLimit int
}

type ListRequest struct {
	Cursor int64
	Limit  int
}

type RecommendItem struct {
	VideoID      int64
	Score        float64
	Detail       videoservice.VideoDetailResult
	AuthorID     int64
	AuthorName   string
	AuthorAvatar string
	AuthorHandle string
	IsFollowed   bool
	IsLiked      bool
	IsFavorited  bool
}

type ListResult struct {
	Items      []RecommendItem
	NextCursor string
}

type scoredVideo struct {
	VideoID      int64
	Score        float64
	CreatedAt    time.Time
	SeenAt       time.Time
	RecallStage  int
	RecallWeight float64
}

type seenPolicy struct {
	cutoff      time.Time
	recallStage int
}

func New(
	redisClient *redis.Client,
	videoRepo VideoRepository,
	interactRepo InteractRepository,
	videoService VideoService,
	userRepo UserRepository,
	relationSvc RelationService,
) *Service {
	return &Service{
		redis:           redisClient,
		videoRepo:       videoRepo,
		interactRepo:    interactRepo,
		videoService:    videoService,
		userRepo:        userRepo,
		relationService: relationSvc,
		cacheTTL:        defaultCacheTTL,
		seenTTL:         defaultSeenTTL,
		seenMaxSize:     defaultSeenMaxSize,
		candidateLimit:  defaultCandidateLimit,
		userActionLimit: defaultActionLimit,
	}
}

func (s *Service) ListRecommend(ctx context.Context, userID int64, req ListRequest) (*ListResult, error) {
	if userID <= 0 {
		return nil, errno.ErrInvalidParam
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	candidates, err := s.loadScoredCandidates(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return &ListResult{Items: []RecommendItem{}}, nil
	}

	start := 0
	if req.Cursor > 0 {
		for index, item := range candidates {
			if item.VideoID == req.Cursor {
				start = index + 1
				break
			}
		}
	}

	items := make([]RecommendItem, 0, limit)
	var nextCursor string
	for i := start; i < len(candidates); i++ {
		candidate := candidates[i]
		detail, detailErr := s.videoService.GetVideoDetail(ctx, userID, candidate.VideoID)
		if detailErr != nil {
			continue
		}

		item := RecommendItem{
			VideoID: candidate.VideoID,
			Score:   candidate.Score,
			Detail:  *detail,
		}

		if s.userRepo != nil {
			author, authorErr := s.userRepo.GetByID(ctx, detail.UserID)
			if authorErr == nil && author != nil {
				item.AuthorID = author.ID
				item.AuthorName = author.Nickname
				item.AuthorAvatar = author.AvatarURL
				item.AuthorHandle = author.Username
				if item.AuthorName == "" {
					item.AuthorName = author.Username
				}
			}
		}
		if s.relationService != nil && detail.UserID != userID {
			state, stateErr := s.relationService.GetRelationState(ctx, userID, detail.UserID)
			if stateErr == nil {
				item.IsFollowed = state.IsFollowed
			}
		}
		if s.interactRepo != nil {
			liked, likeErr := s.interactRepo.HasLikedVideo(ctx, userID, detail.VideoID)
			if likeErr == nil {
				item.IsLiked = liked
			}
			favorited, favoriteErr := s.interactRepo.HasFavoritedVideo(ctx, userID, detail.VideoID)
			if favoriteErr == nil {
				item.IsFavorited = favorited
			}
		}

		items = append(items, item)
		if s.redis != nil {
			nowScore := float64(time.Now().Unix())
			seenKey := rediskey.FeedRecommendSeen(userID)
			_ = s.redis.ZAdd(ctx, seenKey, redis.Z{
				Score:  nowScore,
				Member: strconv.FormatInt(candidate.VideoID, 10),
			}).Err()
			_ = s.redis.Expire(ctx, seenKey, s.seenTTL).Err()
			if s.seenMaxSize > 0 {
				_ = s.redis.ZRemRangeByRank(ctx, seenKey, 0, -(s.seenMaxSize + 1)).Err()
			}
		}

		if len(items) == limit {
			if i+1 < len(candidates) {
				nextCursor = strconv.FormatInt(candidate.VideoID, 10)
			}
			break
		}
	}

	return &ListResult{
		Items:      items,
		NextCursor: nextCursor,
	}, nil
}

func (s *Service) loadScoredCandidates(ctx context.Context, userID int64, target int) ([]scoredVideo, error) {
	if s.redis != nil {
		cached, err := s.readCache(ctx, userID)
		if err == nil && len(cached) > 0 {
			return cached, nil
		}
	}

	candidates, err := s.buildCandidates(ctx, userID, target)
	if err != nil {
		return nil, err
	}

	if s.redis != nil && len(candidates) > 0 {
		_ = s.writeCache(ctx, userID, candidates)
	}
	return candidates, nil
}

func (s *Service) buildCandidates(ctx context.Context, userID int64, target int) ([]scoredVideo, error) {
	if s.videoRepo == nil || s.interactRepo == nil {
		return nil, errno.ErrFeedFetchFailed
	}
	if target <= 0 {
		target = 20
	}

	videos, err := s.videoRepo.ListRecommendVideos(ctx, s.candidateLimit)
	if err != nil {
		return nil, errno.ErrFeedFetchFailed
	}
	if len(videos) == 0 {
		return []scoredVideo{}, nil
	}

	videoIDs := make([]int64, 0, len(videos))
	for _, video := range videos {
		videoIDs = append(videoIDs, video.ID)
	}
	tagMap, err := s.videoRepo.ListHashtagNamesByVideoIDs(ctx, videoIDs)
	if err != nil {
		return nil, errno.ErrFeedFetchFailed
	}

	seenMap := s.loadSeenMap(ctx, userID)
	interests, err := s.buildUserInterest(ctx, userID, tagMap)
	if err != nil {
		return nil, errno.ErrFeedFetchFailed
	}

	df := make(map[string]int)
	videoTerms := make(map[int64]map[string]float64, len(videos))
	for _, video := range videos {
		terms := buildVideoTerms(video, tagMap[video.ID])
		videoTerms[video.ID] = terms
		for term := range terms {
			df[term]++
		}
	}

	totalDocs := float64(len(videos))
	policies := s.buildSeenPolicies()
	seenFallbackCutoff := time.Now().Add(-s.seenTTL)
	for _, policy := range policies {
		scored := make([]scoredVideo, 0, len(videos))
		for _, video := range videos {
			if video.UserID == userID {
				continue
			}

			liked, _ := s.interactRepo.HasLikedVideo(ctx, userID, video.ID)
			if liked {
				continue
			}
			favorited, _ := s.interactRepo.HasFavoritedVideo(ctx, userID, video.ID)
			if favorited {
				continue
			}

			seenAt, hasSeen := seenMap[video.ID]
			recallStage := policy.recallStage
			recallWeight := 1.0
			if hasSeen {
				if seenAt.After(policy.cutoff) {
					continue
				}
				if seenAt.Before(seenFallbackCutoff) {
					recallStage = 0
				} else {
					recallWeight = computeRecallWeight(policy.recallStage)
				}
			} else {
				recallStage = 0
			}

			contentScore := cosineScore(interests, applyIDF(videoTerms[video.ID], df, totalDocs))
			hotScore := computeHotScore(video)
			freshScore := computeFreshScore(video.CreatedAt)
			finalScore := contentScore*0.7 + hotScore*0.2 + freshScore*0.1
			if len(interests) == 0 {
				finalScore = hotScore*0.65 + freshScore*0.35
			}
			finalScore *= recallWeight

			scored = append(scored, scoredVideo{
				VideoID:      video.ID,
				Score:        finalScore,
				CreatedAt:    video.CreatedAt,
				SeenAt:       seenAt,
				RecallStage:  recallStage,
				RecallWeight: recallWeight,
			})
		}

		sortScoredVideos(scored)
		if len(scored) >= target || policy.recallStage == recallStageAllSeen {
			return scored, nil
		}
	}

	return []scoredVideo{}, nil
}

func (s *Service) buildUserInterest(ctx context.Context, userID int64, tagMap map[int64][]string) (map[string]float64, error) {
	actions, err := s.interactRepo.ListUserVideoActions(ctx, userID, s.userActionLimit)
	if err != nil {
		return nil, err
	}
	if len(actions) == 0 {
		return map[string]float64{}, nil
	}

	videoIDs := make([]int64, 0, len(actions))
	seenVideos := make(map[int64]struct{}, len(actions))
	for _, action := range actions {
		if _, exists := seenVideos[action.VideoID]; exists {
			continue
		}
		seenVideos[action.VideoID] = struct{}{}
		videoIDs = append(videoIDs, action.VideoID)
	}

	videos, err := s.videoRepo.ListVideosByIDs(ctx, videoIDs)
	if err != nil {
		return nil, err
	}

	videoMap := make(map[int64]videomodel.Video, len(videos))
	for _, video := range videos {
		videoMap[video.ID] = video
	}

	interest := make(map[string]float64)
	for _, action := range actions {
		video, ok := videoMap[action.VideoID]
		if !ok {
			continue
		}
		terms := buildVideoTerms(video, tagMap[action.VideoID])
		for term, weight := range terms {
			interest[term] += weight * action.Weight
		}
	}
	return normalizeVector(interest), nil
}

func (s *Service) readCache(ctx context.Context, userID int64) ([]scoredVideo, error) {
	values, err := s.redis.ZRevRangeWithScores(ctx, rediskey.FeedRecommend(userID), 0, -1).Result()
	if err != nil {
		return nil, err
	}
	items := make([]scoredVideo, 0, len(values))
	for _, item := range values {
		member, ok := item.Member.(string)
		if !ok {
			continue
		}
		videoID, parseErr := strconv.ParseInt(member, 10, 64)
		if parseErr != nil {
			continue
		}
		items = append(items, scoredVideo{
			VideoID: videoID,
			Score:   item.Score,
		})
	}
	return items, nil
}

func (s *Service) writeCache(ctx context.Context, userID int64, items []scoredVideo) error {
	key := rediskey.FeedRecommend(userID)
	pipe := s.redis.TxPipeline()
	pipe.Del(ctx, key)
	members := make([]redis.Z, 0, len(items))
	for _, item := range items {
		members = append(members, redis.Z{
			Score:  item.Score,
			Member: strconv.FormatInt(item.VideoID, 10),
		})
	}
	if len(members) > 0 {
		pipe.ZAdd(ctx, key, members...)
	}
	pipe.Expire(ctx, key, s.cacheTTL)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *Service) loadSeenMap(ctx context.Context, userID int64) map[int64]time.Time {
	result := make(map[int64]time.Time)
	if s.redis == nil {
		return result
	}
	expireBefore := float64(time.Now().Add(-s.seenTTL).Unix())
	_ = s.redis.ZRemRangeByScore(ctx, rediskey.FeedRecommendSeen(userID), "-inf", strconv.FormatFloat(expireBefore, 'f', -1, 64)).Err()
	values, err := s.redis.ZRangeWithScores(ctx, rediskey.FeedRecommendSeen(userID), 0, -1).Result()
	if err != nil {
		return result
	}
	for _, value := range values {
		member, ok := value.Member.(string)
		if !ok {
			continue
		}
		videoID, parseErr := strconv.ParseInt(member, 10, 64)
		if parseErr == nil {
			result[videoID] = time.Unix(int64(value.Score), 0)
		}
	}
	return result
}

func (s *Service) buildSeenPolicies() []seenPolicy {
	now := time.Now()
	return []seenPolicy{
		{cutoff: now.Add(-12 * time.Hour), recallStage: recallStageFreshOnly},
		{cutoff: now.Add(-6 * time.Hour), recallStage: recallStageOlder12h},
		{cutoff: now.Add(-1 * time.Nanosecond), recallStage: recallStageOlder6h},
		{cutoff: now.Add(24 * time.Hour), recallStage: recallStageAllSeen},
	}
}

func computeRecallWeight(stage int) float64 {
	switch stage {
	case recallStageOlder12h:
		return 0.88
	case recallStageOlder6h:
		return 0.76
	case recallStageAllSeen:
		return 0.64
	default:
		return 1.0
	}
}

func sortScoredVideos(scored []scoredVideo) {
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].RecallStage != scored[j].RecallStage {
			return scored[i].RecallStage < scored[j].RecallStage
		}
		if !scored[i].SeenAt.Equal(scored[j].SeenAt) {
			if scored[i].SeenAt.IsZero() {
				return true
			}
			if scored[j].SeenAt.IsZero() {
				return false
			}
			return scored[i].SeenAt.Before(scored[j].SeenAt)
		}
		if scored[i].Score == scored[j].Score {
			if scored[i].CreatedAt.Equal(scored[j].CreatedAt) {
				return scored[i].VideoID > scored[j].VideoID
			}
			return scored[i].CreatedAt.After(scored[j].CreatedAt)
		}
		return scored[i].Score > scored[j].Score
	})
}

func buildVideoTerms(video videomodel.Video, hashtags []string) map[string]float64 {
	terms := make(map[string]float64)
	for _, token := range tokenize(video.Title) {
		terms[token] += 1.0
	}
	for _, hashtag := range hashtags {
		for _, token := range tokenize(hashtag) {
			terms[token] += 1.5
		}
	}
	return normalizeVector(terms)
}

func tokenize(raw string) []string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return nil
	}

	fields := strings.FieldsFunc(raw, func(r rune) bool {
		if unicode.IsSpace(r) {
			return true
		}
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			return true
		}
		return false
	})

	tokens := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		runes := []rune(field)
		if len(runes) > 1 {
			tokens = append(tokens, field)
			continue
		}
		if unicode.IsLetter(runes[0]) || unicode.IsDigit(runes[0]) {
			tokens = append(tokens, field)
		}
	}
	return tokens
}

func normalizeVector(input map[string]float64) map[string]float64 {
	if len(input) == 0 {
		return map[string]float64{}
	}
	total := 0.0
	for _, value := range input {
		total += value * value
	}
	if total <= 0 {
		return input
	}
	norm := math.Sqrt(total)
	result := make(map[string]float64, len(input))
	for key, value := range input {
		result[key] = value / norm
	}
	return result
}

func applyIDF(tf map[string]float64, docFreq map[string]int, totalDocs float64) map[string]float64 {
	weighted := make(map[string]float64, len(tf))
	for term, value := range tf {
		df := float64(docFreq[term])
		idf := math.Log((1+totalDocs)/(1+df)) + 1
		weighted[term] = value * idf
	}
	return normalizeVector(weighted)
}

func cosineScore(left, right map[string]float64) float64 {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	score := 0.0
	if len(left) > len(right) {
		left, right = right, left
	}
	for term, value := range left {
		score += value * right[term]
	}
	return score
}

func computeHotScore(video videomodel.Video) float64 {
	raw := float64(video.LikeCount)*1.2 +
		float64(video.FavoriteCount)*1.6 +
		float64(video.CommentCount)*1.4 +
		float64(video.PlayCount)*0.2
	if raw <= 0 {
		return 0
	}
	return math.Min(1, math.Log1p(raw)/10)
}

func computeFreshScore(createdAt time.Time) float64 {
	if createdAt.IsZero() {
		return 0
	}
	hours := time.Since(createdAt).Hours()
	if hours <= 0 {
		return 1
	}
	score := 1 / (1 + hours/24)
	if score < 0 {
		return 0
	}
	return score
}
