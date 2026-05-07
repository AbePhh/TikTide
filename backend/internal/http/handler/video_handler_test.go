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
	"github.com/AbePhh/TikTide/backend/pkg/rediskey"
	"github.com/AbePhh/TikTide/backend/tests/mocks"
)

func TestVideoPublishFlow(t *testing.T) {
	t.Parallel()

	router, videoOSS, _, _, cleanup := newVideoTestRouter(t)
	defer cleanup()

	registerResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/register", `{"username":"video_user","password":"password123"}`, "")
	if registerResp.Code != http.StatusOK {
		t.Fatalf("unexpected register status: %d", registerResp.Code)
	}

	loginResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/login", `{"username":"video_user","password":"password123"}`, "")
	if loginResp.Code != http.StatusOK {
		t.Fatalf("unexpected login status: %d", loginResp.Code)
	}

	var loginEnvelope struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginResp.Body.Bytes(), &loginEnvelope); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	token := loginEnvelope.Data.Token
	credentialResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/video/upload-credential", `{"file_name":"demo.mp4"}`, token)
	if credentialResp.Code != http.StatusOK {
		t.Fatalf("unexpected upload credential status: %d", credentialResp.Code)
	}

	var credentialEnvelope struct {
		Data struct {
			ObjectKey string `json:"object_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(credentialResp.Body.Bytes(), &credentialEnvelope); err != nil {
		t.Fatalf("decode upload credential response: %v", err)
	}
	if credentialEnvelope.Data.ObjectKey == "" {
		t.Fatal("expected object key")
	}
	videoOSS.AddObject(credentialEnvelope.Data.ObjectKey)

	publishResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/video/publish", `{
		"object_key":"`+credentialEnvelope.Data.ObjectKey+`",
		"title":"hello video",
		"hashtag_ids":[11],
		"allow_comment":1,
		"visibility":1
	}`, token)
	if publishResp.Code != http.StatusOK {
		t.Fatalf("unexpected publish status: %d", publishResp.Code)
	}
}

func TestHashtagEndpoints(t *testing.T) {
	t.Parallel()

	router, videoOSS, _, _, cleanup := newVideoTestRouter(t)
	defer cleanup()

	registerResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/register", `{"username":"hashtag_user","password":"password123"}`, "")
	if registerResp.Code != http.StatusOK {
		t.Fatalf("unexpected register status: %d", registerResp.Code)
	}

	loginResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/login", `{"username":"hashtag_user","password":"password123"}`, "")
	if loginResp.Code != http.StatusOK {
		t.Fatalf("unexpected login status: %d", loginResp.Code)
	}

	var loginEnvelope struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginResp.Body.Bytes(), &loginEnvelope); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	token := loginEnvelope.Data.Token

	credentialResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/video/upload-credential", `{"file_name":"topic.mp4"}`, token)
	if credentialResp.Code != http.StatusOK {
		t.Fatalf("unexpected upload credential status: %d", credentialResp.Code)
	}

	var credentialEnvelope struct {
		Data struct {
			ObjectKey string `json:"object_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(credentialResp.Body.Bytes(), &credentialEnvelope); err != nil {
		t.Fatalf("decode upload credential response: %v", err)
	}
	videoOSS.AddObject(credentialEnvelope.Data.ObjectKey)

	publishResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/video/publish", `{
		"object_key":"`+credentialEnvelope.Data.ObjectKey+`",
		"title":"topic video",
		"hashtag_ids":[11],
		"allow_comment":1,
		"visibility":1
	}`, token)
	if publishResp.Code != http.StatusOK {
		t.Fatalf("unexpected publish status: %d", publishResp.Code)
	}

	hashtagResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/hashtag/11", "", token)
	if hashtagResp.Code != http.StatusOK {
		t.Fatalf("unexpected hashtag status: %d", hashtagResp.Code)
	}

	hotHashtagsResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/hashtag/hot?limit=10", "", token)
	if hotHashtagsResp.Code != http.StatusOK {
		t.Fatalf("unexpected hot hashtags status: %d", hotHashtagsResp.Code)
	}

	hashtagVideosResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/hashtag/11/videos?limit=20", "", token)
	if hashtagVideosResp.Code != http.StatusOK {
		t.Fatalf("unexpected hashtag videos status: %d", hashtagVideosResp.Code)
	}
}

func TestCreateHashtagEndpoint(t *testing.T) {
	t.Parallel()

	router, _, _, _, cleanup := newVideoTestRouter(t)
	defer cleanup()

	registerResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/register", `{"username":"topic_user","password":"password123"}`, "")
	if registerResp.Code != http.StatusOK {
		t.Fatalf("unexpected register status: %d", registerResp.Code)
	}

	loginResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/login", `{"username":"topic_user","password":"password123"}`, "")
	if loginResp.Code != http.StatusOK {
		t.Fatalf("unexpected login status: %d", loginResp.Code)
	}

	var loginEnvelope struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginResp.Body.Bytes(), &loginEnvelope); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	createResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/hashtag", `{"name":"travel"}`, loginEnvelope.Data.Token)
	if createResp.Code != http.StatusOK {
		t.Fatalf("unexpected create hashtag status: %d", createResp.Code)
	}
}

func TestDraftEndpoints(t *testing.T) {
	t.Parallel()

	router, _, _, _, cleanup := newVideoTestRouter(t)
	defer cleanup()

	registerResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/register", `{"username":"draft_user","password":"password123"}`, "")
	if registerResp.Code != http.StatusOK {
		t.Fatalf("unexpected register status: %d", registerResp.Code)
	}

	loginResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/login", `{"username":"draft_user","password":"password123"}`, "")
	if loginResp.Code != http.StatusOK {
		t.Fatalf("unexpected login status: %d", loginResp.Code)
	}

	var loginEnvelope struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginResp.Body.Bytes(), &loginEnvelope); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	token := loginEnvelope.Data.Token

	saveResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/draft", `{
		"object_key":"video/source/1001/20260424/draft.mp4",
		"cover_url":"https://example.com/cover.png",
		"title":"draft title",
		"tag_names":"travel,sunset",
		"allow_comment":1,
		"visibility":0
	}`, token)
	if saveResp.Code != http.StatusOK {
		t.Fatalf("unexpected save draft status: %d", saveResp.Code)
	}

	listResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/draft/list", "", token)
	if listResp.Code != http.StatusOK {
		t.Fatalf("unexpected list draft status: %d", listResp.Code)
	}

	var listEnvelope struct {
		Data struct {
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &listEnvelope); err != nil {
		t.Fatalf("decode draft list response: %v", err)
	}
	if len(listEnvelope.Data.Items) != 1 {
		t.Fatalf("expected 1 draft, got %d", len(listEnvelope.Data.Items))
	}

	deleteResp := performJSONRequest(t, router, http.MethodDelete, "/api/v1/draft/"+listEnvelope.Data.Items[0].ID, "", token)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("unexpected delete draft status: %d", deleteResp.Code)
	}
}

func TestVideoDetailAndResourcesEndpoints(t *testing.T) {
	t.Parallel()

	router, videoOSS, videoService, _, cleanup := newVideoTestRouter(t)
	defer cleanup()

	registerResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/register", `{"username":"detail_user","password":"password123"}`, "")
	if registerResp.Code != http.StatusOK {
		t.Fatalf("unexpected register status: %d", registerResp.Code)
	}

	loginResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/login", `{"username":"detail_user","password":"password123"}`, "")
	if loginResp.Code != http.StatusOK {
		t.Fatalf("unexpected login status: %d", loginResp.Code)
	}

	var loginEnvelope struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginResp.Body.Bytes(), &loginEnvelope); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	token := loginEnvelope.Data.Token

	credentialResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/video/upload-credential", `{"file_name":"detail.mp4"}`, token)
	if credentialResp.Code != http.StatusOK {
		t.Fatalf("unexpected upload credential status: %d", credentialResp.Code)
	}

	var credentialEnvelope struct {
		Data struct {
			ObjectKey string `json:"object_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(credentialResp.Body.Bytes(), &credentialEnvelope); err != nil {
		t.Fatalf("decode upload credential response: %v", err)
	}
	videoOSS.AddObject(credentialEnvelope.Data.ObjectKey)

	publishResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/video/publish", `{
		"object_key":"`+credentialEnvelope.Data.ObjectKey+`",
		"title":"detail video",
		"allow_comment":1,
		"visibility":1
	}`, token)
	if publishResp.Code != http.StatusOK {
		t.Fatalf("unexpected publish status: %d", publishResp.Code)
	}

	var publishEnvelope struct {
		Data struct {
			VideoID string `json:"video_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(publishResp.Body.Bytes(), &publishEnvelope); err != nil {
		t.Fatalf("decode publish response: %v", err)
	}

	videoID, err := strconv.ParseInt(publishEnvelope.Data.VideoID, 10, 64)
	if err != nil {
		t.Fatalf("parse video id: %v", err)
	}

	if err := videoService.StartTranscode(t.Context(), videoID); err != nil {
		t.Fatalf("start transcode failed: %v", err)
	}
	if err := videoService.CompleteTranscode(t.Context(), videoservice.CompleteTranscodeRequest{
		VideoID:    videoID,
		CoverURL:   "https://example.com/cover.jpg",
		DurationMS: 8000,
		Resources: []videoservice.TranscodedResource{
			{Resolution: "720p", FileURL: "https://example.com/720.m3u8", FileSize: 1024, Bitrate: 1500},
			{Resolution: "1080p", FileURL: "https://example.com/1080.m3u8", FileSize: 2048, Bitrate: 2400},
		},
	}); err != nil {
		t.Fatalf("complete transcode failed: %v", err)
	}

	detailResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/video/"+publishEnvelope.Data.VideoID, "", token)
	if detailResp.Code != http.StatusOK {
		t.Fatalf("unexpected detail status: %d", detailResp.Code)
	}

	var detailEnvelope struct {
		Data struct {
			VideoID         string `json:"video_id"`
			DurationMS      int32  `json:"duration_ms"`
			TranscodeStatus int8   `json:"transcode_status"`
			CoverURL        string `json:"cover_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(detailResp.Body.Bytes(), &detailEnvelope); err != nil {
		t.Fatalf("decode detail response: %v", err)
	}
	if detailEnvelope.Data.VideoID != publishEnvelope.Data.VideoID || detailEnvelope.Data.TranscodeStatus != 2 || detailEnvelope.Data.CoverURL == "" {
		t.Fatalf("unexpected detail payload: %+v", detailEnvelope.Data)
	}

	resourceResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/video/"+publishEnvelope.Data.VideoID+"/resources", "", token)
	if resourceResp.Code != http.StatusOK {
		t.Fatalf("unexpected resource status: %d", resourceResp.Code)
	}

	var resourceEnvelope struct {
		Data struct {
			Items []struct {
				Resolution string `json:"resolution"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resourceResp.Body.Bytes(), &resourceEnvelope); err != nil {
		t.Fatalf("decode resource response: %v", err)
	}
	if len(resourceEnvelope.Data.Items) != 2 || resourceEnvelope.Data.Items[0].Resolution != "1080p" {
		t.Fatalf("unexpected resource payload: %+v", resourceEnvelope.Data.Items)
	}
}

func TestVideoPlayReportIncreasesCountWithDedupe(t *testing.T) {
	t.Parallel()

	router, videoOSS, videoService, redisClient, cleanup := newVideoTestRouter(t)
	defer cleanup()

	registerResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/register", `{"username":"play_user","password":"password123"}`, "")
	if registerResp.Code != http.StatusOK {
		t.Fatalf("unexpected register status: %d", registerResp.Code)
	}

	loginResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/login", `{"username":"play_user","password":"password123"}`, "")
	if loginResp.Code != http.StatusOK {
		t.Fatalf("unexpected login status: %d", loginResp.Code)
	}

	var loginEnvelope struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginResp.Body.Bytes(), &loginEnvelope); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	token := loginEnvelope.Data.Token

	credentialResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/video/upload-credential", `{"file_name":"play.mp4"}`, token)
	if credentialResp.Code != http.StatusOK {
		t.Fatalf("unexpected upload credential status: %d", credentialResp.Code)
	}

	var credentialEnvelope struct {
		Data struct {
			ObjectKey string `json:"object_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(credentialResp.Body.Bytes(), &credentialEnvelope); err != nil {
		t.Fatalf("decode upload credential response: %v", err)
	}
	videoOSS.AddObject(credentialEnvelope.Data.ObjectKey)

	publishResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/video/publish", `{
		"object_key":"`+credentialEnvelope.Data.ObjectKey+`",
		"title":"play video",
		"allow_comment":1,
		"visibility":1
	}`, token)
	if publishResp.Code != http.StatusOK {
		t.Fatalf("unexpected publish status: %d", publishResp.Code)
	}

	var publishEnvelope struct {
		Data struct {
			VideoID string `json:"video_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(publishResp.Body.Bytes(), &publishEnvelope); err != nil {
		t.Fatalf("decode publish response: %v", err)
	}

	videoID, err := strconv.ParseInt(publishEnvelope.Data.VideoID, 10, 64)
	if err != nil {
		t.Fatalf("parse video id: %v", err)
	}

	if err := videoService.StartTranscode(t.Context(), videoID); err != nil {
		t.Fatalf("start transcode failed: %v", err)
	}
	if err := videoService.CompleteTranscode(t.Context(), videoservice.CompleteTranscodeRequest{
		VideoID:    videoID,
		CoverURL:   "https://example.com/cover.jpg",
		DurationMS: 8000,
		Resources: []videoservice.TranscodedResource{
			{Resolution: "720p", FileURL: "https://example.com/720.m3u8", FileSize: 1024, Bitrate: 1500},
		},
	}); err != nil {
		t.Fatalf("complete transcode failed: %v", err)
	}

	reportResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/video/play/report", `{"video_id":"`+publishEnvelope.Data.VideoID+`"}`, token)
	if reportResp.Code != http.StatusOK {
		t.Fatalf("unexpected first report status: %d", reportResp.Code)
	}
	reportAgainResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/video/play/report", `{"video_id":"`+publishEnvelope.Data.VideoID+`"}`, token)
	if reportAgainResp.Code != http.StatusOK {
		t.Fatalf("unexpected second report status: %d", reportAgainResp.Code)
	}

	detailResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/video/"+publishEnvelope.Data.VideoID, "", token)
	if detailResp.Code != http.StatusOK {
		t.Fatalf("unexpected detail status: %d", detailResp.Code)
	}

	var detailEnvelope struct {
		Data struct {
			UserID    string `json:"user_id"`
			PlayCount int64  `json:"play_count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(detailResp.Body.Bytes(), &detailEnvelope); err != nil {
		t.Fatalf("decode detail response: %v", err)
	}
	if detailEnvelope.Data.PlayCount != 1 {
		t.Fatalf("expected play count 1 after dedupe, got %d", detailEnvelope.Data.PlayCount)
	}

	userID, err := strconv.ParseInt(detailEnvelope.Data.UserID, 10, 64)
	if err != nil {
		t.Fatalf("parse user id: %v", err)
	}
	dedupeExists, err := redisClient.Exists(t.Context(), rediskey.VideoPlayReported(userID, videoID)).Result()
	if err != nil {
		t.Fatalf("check play dedupe key failed: %v", err)
	}
	if dedupeExists != 1 {
		t.Fatal("expected play dedupe key to exist")
	}
}

func newVideoTestRouter(t *testing.T) (http.Handler, *mocks.MemoryOSSClient, *videoservice.Service, *redis.Client, func()) {
	t.Helper()

	jwtManager, err := jwt.NewManager("tiktide-system", "tiktide-test", "tiktide-web", 24*time.Hour)
	if err != nil {
		t.Fatalf("create jwt manager: %v", err)
	}

	userRepo := mocks.NewMemoryUserRepository()
	relationRepo := mocks.NewMemoryRelationRepository(userRepo)
	blocklist := mocks.NewMemoryTokenBlacklist()
	idGenerator := mocks.NewIncrementalIDGenerator(2000)
	videoRepo := mocks.NewMemoryVideoRepository(11)
	videoOSS := mocks.NewMemoryOSSClient()
	miniRedis, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	redisClient := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	videoService := videoservice.New(videoRepo, videoOSS, mocks.NewIncrementalIDGenerator(5000), config.Config{
		OSSUploadExpire: 15 * time.Minute,
	})
	videoService.SetRedisClient(redisClient)
	messageSvc := messageservice.New(nil, nil)
	relationService := relationservice.New(relationRepo, userRepo, messageSvc)
	userService := userservice.New(userRepo, relationService, idGenerator, jwtManager, blocklist)
	feedSvc := feedservice.New(nil, relationRepo, videoService, userRepo, relationService, nil)
	recommendSvc := recommendservice.New(nil, videoRepo, mocks.NewMemoryInteractRepository(videoRepo, userRepo), videoService, userRepo, relationService)
	interactSvc := interactservice.New(nil, nil, nil, messageSvc, nil)

	cfg := config.Config{
		CORSAllowedOrigins: []string{"http://localhost:5173"},
	}

	cleanup := func() {
		_ = redisClient.Close()
		miniRedis.Close()
	}

return httprouter.NewEngine(app.NewForTest(cfg, userService, relationService, videoService, interactSvc, feedSvc, recommendSvc, messageSvc, nil, jwtManager, blocklist)), videoOSS, videoService, redisClient, cleanup
}
