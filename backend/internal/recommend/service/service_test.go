package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	relationservice "github.com/AbePhh/TikTide/backend/internal/relation/service"
	usermodel "github.com/AbePhh/TikTide/backend/internal/user/model"
	videomodel "github.com/AbePhh/TikTide/backend/internal/video/model"
	videoservice "github.com/AbePhh/TikTide/backend/internal/video/service"
	"github.com/AbePhh/TikTide/backend/pkg/config"
	"github.com/AbePhh/TikTide/backend/pkg/rediskey"
	"github.com/AbePhh/TikTide/backend/tests/mocks"
)

func TestRecommendRecallReleasesOlderThan12HoursBefore6Hours(t *testing.T) {
	t.Parallel()

	svc, redisClient, videoRepo, _, cleanup := newRecommendTestService(t)
	defer cleanup()

	userID := int64(1001)
	now := time.Now()

	createRecommendVideo(t, videoRepo, 2001, 3001, "travel one", now.Add(-2*time.Hour))
	createRecommendVideo(t, videoRepo, 2002, 3001, "travel two", now.Add(-3*time.Hour))

	seenKey := rediskey.FeedRecommendSeen(userID)
	if err := redisClient.ZAdd(context.Background(), seenKey,
		redis.Z{Score: float64(now.Add(-13 * time.Hour).Unix()), Member: "2001"},
		redis.Z{Score: float64(now.Add(-7 * time.Hour).Unix()), Member: "2002"},
	).Err(); err != nil {
		t.Fatalf("seed seen failed: %v", err)
	}

	result, err := svc.ListRecommend(context.Background(), userID, ListRequest{Limit: 1})
	if err != nil {
		t.Fatalf("list recommend failed: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected only one recalled item, got %d", len(result.Items))
	}
	if result.Items[0].VideoID != 2001 {
		t.Fatalf("expected >12h seen video to be recalled first, got %d", result.Items[0].VideoID)
	}
}

func TestRecommendRecallPrefersOldestSeenWhenAllNeedRecall(t *testing.T) {
	t.Parallel()

	svc, redisClient, videoRepo, _, cleanup := newRecommendTestService(t)
	defer cleanup()

	userID := int64(1002)
	now := time.Now()

	createRecommendVideo(t, videoRepo, 2101, 3101, "food old", now.Add(-2*time.Hour))
	createRecommendVideo(t, videoRepo, 2102, 3101, "food newer", now.Add(-90*time.Minute))

	seenKey := rediskey.FeedRecommendSeen(userID)
	if err := redisClient.ZAdd(context.Background(), seenKey,
		redis.Z{Score: float64(now.Add(-5 * time.Hour).Unix()), Member: "2101"},
		redis.Z{Score: float64(now.Add(-1 * time.Hour).Unix()), Member: "2102"},
	).Err(); err != nil {
		t.Fatalf("seed seen failed: %v", err)
	}

	result, err := svc.ListRecommend(context.Background(), userID, ListRequest{Limit: 2})
	if err != nil {
		t.Fatalf("list recommend failed: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected recalled items, got %d", len(result.Items))
	}
	if result.Items[0].VideoID != 2101 {
		t.Fatalf("expected oldest seen item first, got %d then %d", result.Items[0].VideoID, result.Items[1].VideoID)
	}
}

func TestRecommendRecallStillFiltersLikedAndFavoritedVideos(t *testing.T) {
	t.Parallel()

	svc, _, videoRepo, interactRepo, cleanup := newRecommendTestService(t)
	defer cleanup()

	userID := int64(1003)
	now := time.Now()

	createRecommendVideo(t, videoRepo, 2201, 3201, "mountain one", now.Add(-2*time.Hour))
	createRecommendVideo(t, videoRepo, 2202, 3201, "mountain two", now.Add(-3*time.Hour))

	if err := interactRepo.LikeVideo(context.Background(), userID, 2201, 3201); err != nil {
		t.Fatalf("like video failed: %v", err)
	}
	if err := interactRepo.FavoriteVideo(context.Background(), userID, 2202); err != nil {
		t.Fatalf("favorite video failed: %v", err)
	}

	result, err := svc.ListRecommend(context.Background(), userID, ListRequest{Limit: 10})
	if err != nil {
		t.Fatalf("list recommend failed: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected liked/favorited videos to stay filtered, got %+v", result.Items)
	}
}

func newRecommendTestService(t *testing.T) (*Service, *redis.Client, *mocks.MemoryVideoRepository, *mocks.MemoryInteractRepository, func()) {
	t.Helper()

	miniRedis, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis failed: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	userRepo := mocks.NewMemoryUserRepository()
	videoRepo := mocks.NewMemoryVideoRepository()
	videoOSS := mocks.NewMemoryOSSClient()
	interactRepo := mocks.NewMemoryInteractRepository(videoRepo, userRepo)
	relationRepo := mocks.NewMemoryRelationRepository(userRepo)
	relationSvc := relationservice.New(relationRepo, userRepo, nil)
	videoSvc := videoservice.New(videoRepo, videoOSS, mocks.NewIncrementalIDGenerator(9000), config.Config{
		OSSUploadExpire: 15 * time.Minute,
		OSSReadExpire:   15 * time.Minute,
	})

	seedAuthor(t, userRepo, 3001)
	seedAuthor(t, userRepo, 3101)
	seedAuthor(t, userRepo, 3201)

	svc := New(redisClient, videoRepo, interactRepo, videoSvc, userRepo, relationSvc)
	svc.candidateLimit = 100
	svc.cacheTTL = time.Minute
	svc.seenTTL = 24 * time.Hour
	svc.seenMaxSize = 2000

	cleanup := func() {
		_ = redisClient.Close()
		miniRedis.Close()
	}
	return svc, redisClient, videoRepo, interactRepo, cleanup
}

func createRecommendVideo(t *testing.T, repo *mocks.MemoryVideoRepository, videoID, authorID int64, title string, createdAt time.Time) {
	t.Helper()

	objectName := fmt.Sprintf("video-%d.mp4", videoID)

	video := &videomodel.Video{
		ID:              videoID,
		UserID:          authorID,
		ObjectKey:       "video/source/test/" + objectName,
		SourceURL:       "https://example.com/object/" + objectName,
		Title:           title,
		AllowComment:    1,
		Visibility:      videomodel.VisibilityPublic,
		TranscodeStatus: videomodel.TranscodeSuccess,
		AuditStatus:     videomodel.AuditPassed,
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt,
	}
	if err := repo.CreateVideo(context.Background(), video, nil); err != nil {
		t.Fatalf("create recommend video failed: %v", err)
	}
}

func seedAuthor(t *testing.T, repo *mocks.MemoryUserRepository, userID int64) {
	t.Helper()

	user := &usermodel.User{
		ID:       userID,
		Username: "user_" + time.Unix(userID, 0).Format("150405"),
		Nickname: "Author",
		Status:   1,
	}
	stats := &usermodel.UserStats{ID: userID}
	_ = repo.Create(context.Background(), user, stats)
}
