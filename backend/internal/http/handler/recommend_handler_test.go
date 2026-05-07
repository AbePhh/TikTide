package handler_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/AbePhh/TikTide/backend/internal/app"
	feedservice "github.com/AbePhh/TikTide/backend/internal/feed/service"
	httprouter "github.com/AbePhh/TikTide/backend/internal/http/router"
	interactservice "github.com/AbePhh/TikTide/backend/internal/interact/service"
	messageservice "github.com/AbePhh/TikTide/backend/internal/message/service"
	recommendservice "github.com/AbePhh/TikTide/backend/internal/recommend/service"
	relationservice "github.com/AbePhh/TikTide/backend/internal/relation/service"
	userservice "github.com/AbePhh/TikTide/backend/internal/user/service"
	videoservice "github.com/AbePhh/TikTide/backend/internal/video/service"
	"github.com/AbePhh/TikTide/backend/pkg/config"
	"github.com/AbePhh/TikTide/backend/pkg/jwt"
	"github.com/AbePhh/TikTide/backend/tests/mocks"
)

func TestRecommendFeedReturnsScoredVideos(t *testing.T) {
	t.Parallel()

	router, videoOSS, videoService, cleanup := newRecommendTestRouter(t)
	defer cleanup()

	authorToken, _ := registerAndLogin(t, router, "recommend_author")
	userToken, _ := registerAndLogin(t, router, "recommend_user")

	travelVideoID := publishVideoForTestWithStatus(t, router, authorToken, "travel.mp4", "travel beach sunset", videoService, videoOSS)
	foodVideoID := publishVideoForTestWithStatus(t, router, authorToken, "food.mp4", "food cooking kitchen", videoService, videoOSS)

	likeResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/interact/like", `{"video_id":`+strconv.FormatInt(travelVideoID, 10)+`,"action_type":1}`, userToken)
	if likeResp.Code != http.StatusOK {
		t.Fatalf("unexpected like status: %d", likeResp.Code)
	}

	newTravelID := publishVideoForTestWithStatus(t, router, authorToken, "travel-new.mp4", "travel sea island", videoService, videoOSS)
	_ = foodVideoID

	resp := performJSONRequest(t, router, http.MethodGet, "/api/v1/feed/recommend?limit=20", "", userToken)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected recommend status: %d", resp.Code)
	}

	var envelope struct {
		Data struct {
			Items []struct {
				VideoID string `json:"video_id"`
				Title   string `json:"title"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode recommend response: %v", err)
	}
	if len(envelope.Data.Items) == 0 {
		t.Fatal("expected recommend items")
	}
	if envelope.Data.Items[0].VideoID != strconv.FormatInt(newTravelID, 10) {
		t.Fatalf("expected similar travel video first, got %+v", envelope.Data.Items)
	}
}

func TestRecommendFeedSupportsCursor(t *testing.T) {
	t.Parallel()

	router, videoOSS, videoService, cleanup := newRecommendTestRouter(t)
	defer cleanup()

	authorToken, _ := registerAndLogin(t, router, "cursor_author")
	userToken, _ := registerAndLogin(t, router, "cursor_user")

	firstID := publishVideoForTestWithStatus(t, router, authorToken, "city.mp4", "city walk skyline", videoService, videoOSS)
	secondID := publishVideoForTestWithStatus(t, router, authorToken, "night.mp4", "night city street", videoService, videoOSS)
	thirdID := publishVideoForTestWithStatus(t, router, authorToken, "park.mp4", "green park relax", videoService, videoOSS)
	_ = firstID
	_ = thirdID

	resp := performJSONRequest(t, router, http.MethodGet, "/api/v1/feed/recommend?limit=1", "", userToken)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected recommend first page status: %d", resp.Code)
	}

	var firstPage struct {
		Data struct {
			Items []struct {
				VideoID string `json:"video_id"`
			} `json:"items"`
			NextCursor string `json:"next_cursor"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &firstPage); err != nil {
		t.Fatalf("decode first page: %v", err)
	}
	if len(firstPage.Data.Items) != 1 {
		t.Fatalf("expected 1 recommend item, got %d", len(firstPage.Data.Items))
	}
	if firstPage.Data.NextCursor == "" {
		t.Fatal("expected next cursor")
	}

	secondResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/feed/recommend?limit=1&cursor="+firstPage.Data.NextCursor, "", userToken)
	if secondResp.Code != http.StatusOK {
		t.Fatalf("unexpected recommend second page status: %d", secondResp.Code)
	}

	var secondPage struct {
		Data struct {
			Items []struct {
				VideoID string `json:"video_id"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(secondResp.Body.Bytes(), &secondPage); err != nil {
		t.Fatalf("decode second page: %v", err)
	}
	if len(secondPage.Data.Items) == 0 {
		t.Fatal("expected second page items")
	}
	if secondPage.Data.Items[0].VideoID == strconv.FormatInt(secondID, 10) && firstPage.Data.Items[0].VideoID == strconv.FormatInt(secondID, 10) {
		t.Fatal("expected cursor to advance to next item")
	}
}

func newRecommendTestRouter(t *testing.T) (http.Handler, *mocks.MemoryOSSClient, *videoservice.Service, func()) {
	t.Helper()

	jwtManager, err := jwt.NewManager("tiktide-system", "tiktide-test", "tiktide-web", 24*time.Hour)
	if err != nil {
		t.Fatalf("create jwt manager: %v", err)
	}

	miniRedis, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	redisClient := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})

	userRepo := mocks.NewMemoryUserRepository()
	relationRepo := mocks.NewMemoryRelationRepository(userRepo)
	videoRepo := mocks.NewMemoryVideoRepository()
	videoOSS := mocks.NewMemoryOSSClient()
	interactRepo := mocks.NewMemoryInteractRepository(videoRepo, userRepo)
	blocklist := mocks.NewMemoryTokenBlacklist()

	messageSvc := messageservice.New(nil, redisClient)
	relationSvc := relationservice.New(relationRepo, userRepo, messageSvc)
	userSvc := userservice.New(userRepo, relationSvc, mocks.NewIncrementalIDGenerator(2000), jwtManager, blocklist)
	videoSvc := videoservice.New(videoRepo, videoOSS, mocks.NewIncrementalIDGenerator(5000), config.Config{
		OSSUploadExpire: 15 * time.Minute,
	})
	feedSvc := feedservice.New(redisClient, relationRepo, videoSvc, userRepo, relationSvc, interactRepo)
	recommendSvc := recommendservice.New(redisClient, videoRepo, interactRepo, videoSvc, userRepo, relationSvc)
	interactSvc := interactservice.New(interactRepo, userRepo, videoRepo, messageSvc, mocks.NewIncrementalIDGenerator(8000))

	cfg := config.Config{
		CORSAllowedOrigins: []string{"http://localhost:5173"},
	}
router := httprouter.NewEngine(app.NewForTest(cfg, userSvc, relationSvc, videoSvc, interactSvc, feedSvc, recommendSvc, messageSvc, nil, jwtManager, blocklist))
	cleanup := func() {
		_ = redisClient.Close()
		miniRedis.Close()
	}
	return router, videoOSS, videoSvc, cleanup
}
