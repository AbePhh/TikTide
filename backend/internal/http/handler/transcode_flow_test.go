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
	messagemodel "github.com/AbePhh/TikTide/backend/internal/message/model"
	messageservice "github.com/AbePhh/TikTide/backend/internal/message/service"
	recommendservice "github.com/AbePhh/TikTide/backend/internal/recommend/service"
	relationservice "github.com/AbePhh/TikTide/backend/internal/relation/service"
	userservice "github.com/AbePhh/TikTide/backend/internal/user/service"
	videomodel "github.com/AbePhh/TikTide/backend/internal/video/model"
	videoservice "github.com/AbePhh/TikTide/backend/internal/video/service"
	videotranscode "github.com/AbePhh/TikTide/backend/internal/video/transcode"
	"github.com/AbePhh/TikTide/backend/pkg/config"
	"github.com/AbePhh/TikTide/backend/pkg/jwt"
	"github.com/AbePhh/TikTide/backend/tests/mocks"
)

func TestFeedFollowingAfterTranscodeSuccess(t *testing.T) {
	t.Parallel()

	router, videoOSS, videoService, worker, cleanup := newTranscodeFlowTestEnv(t)
	defer cleanup()

	authorToken, authorID := registerAndLogin(t, router, "author_feed")
	followerToken, followerID := registerAndLogin(t, router, "follower_feed")

	followResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/relation/action", `{
		"to_user_id":`+authorID+`,
		"action_type":1
	}`, followerToken)
	if followResp.Code != http.StatusOK {
		t.Fatalf("unexpected follow status: %d", followResp.Code)
	}

	videoID := publishVideoForTest(t, router, authorToken, "feed-demo.mp4", "feed video", videoOSS)

	if err := videoService.StartTranscode(t.Context(), videoID); err != nil {
		t.Fatalf("start transcode failed: %v", err)
	}

	detail, err := videoService.GetVideoForTranscode(t.Context(), videoID)
	if err != nil {
		t.Fatalf("get video for transcode failed: %v", err)
	}

	if err := videoService.CompleteTranscode(t.Context(), videoservice.CompleteTranscodeRequest{
		VideoID:    videoID,
		CoverURL:   "https://example.com/feed-cover.jpg",
		DurationMS: 8500,
		Resources: []videoservice.TranscodedResource{
			{Resolution: "480p", FileURL: "https://example.com/feed-480.mp4", FileSize: 1024, Bitrate: 900000},
			{Resolution: "720p", FileURL: "https://example.com/feed-720.mp4", FileSize: 2048, Bitrate: 1800000},
			{Resolution: "1080p", FileURL: "https://example.com/feed-1080.mp4", FileSize: 4096, Bitrate: 3000000},
		},
	}); err != nil {
		t.Fatalf("complete transcode failed: %v", err)
	}

	worker.RunPostSuccessHooksForTest(t.Context(), detail, videoID)

	feedResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/feed/following?limit=20", "", followerToken)
	if feedResp.Code != http.StatusOK {
		t.Fatalf("unexpected feed status: %d", feedResp.Code)
	}

	var feedEnvelope struct {
		Data struct {
			Items []struct {
				VideoID         string `json:"video_id"`
				UserID          string `json:"user_id"`
				Title           string `json:"title"`
				TranscodeStatus int8   `json:"transcode_status"`
				CoverURL        string `json:"cover_url"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(feedResp.Body.Bytes(), &feedEnvelope); err != nil {
		t.Fatalf("decode feed response: %v", err)
	}
	if len(feedEnvelope.Data.Items) != 1 {
		t.Fatalf("expected 1 feed item, got %d", len(feedEnvelope.Data.Items))
	}
	item := feedEnvelope.Data.Items[0]
	if item.VideoID != strconv.FormatInt(videoID, 10) || item.UserID != authorID || item.TranscodeStatus != videomodel.TranscodeSuccess || item.CoverURL == "" || item.Title != "feed video" {
		t.Fatalf("unexpected feed item: %+v", item)
	}
	_ = followerID
}

func TestFeedFollowingReadsAuthorOutboxForBigV(t *testing.T) {
	t.Parallel()

	router, videoOSS, videoService, worker, cleanup := newTranscodeFlowTestEnv(t)
	defer cleanup()

	feedImpl, ok := workerFeedService(worker).(*feedservice.Service)
	if !ok {
		t.Fatal("expected feed service implementation")
	}
	feedImpl.SetBigVThresholdForTest(1)

	authorToken, authorID := registerAndLogin(t, router, "author_outbox")
	followerToken, _ := registerAndLogin(t, router, "follower_outbox")

	followResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/relation/action", `{
		"to_user_id":`+authorID+`,
		"action_type":1
	}`, followerToken)
	if followResp.Code != http.StatusOK {
		t.Fatalf("unexpected follow status: %d", followResp.Code)
	}

	videoID := publishVideoForTest(t, router, authorToken, "outbox-demo.mp4", "outbox video", videoOSS)
	if err := videoService.StartTranscode(t.Context(), videoID); err != nil {
		t.Fatalf("start transcode failed: %v", err)
	}
	detail, err := videoService.GetVideoForTranscode(t.Context(), videoID)
	if err != nil {
		t.Fatalf("get video for transcode failed: %v", err)
	}
	if err := videoService.CompleteTranscode(t.Context(), videoservice.CompleteTranscodeRequest{
		VideoID:    videoID,
		CoverURL:   "https://example.com/outbox-cover.jpg",
		DurationMS: 8500,
		Resources: []videoservice.TranscodedResource{
			{Resolution: "720p", FileURL: "https://example.com/outbox-720.mp4", FileSize: 2048, Bitrate: 1800000},
		},
	}); err != nil {
		t.Fatalf("complete transcode failed: %v", err)
	}

	worker.RunPostSuccessHooksForTest(t.Context(), detail, videoID)

	feedResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/feed/following?limit=20", "", followerToken)
	if feedResp.Code != http.StatusOK {
		t.Fatalf("unexpected feed status: %d", feedResp.Code)
	}

	var feedEnvelope struct {
		Data struct {
			Items []struct {
				VideoID string `json:"video_id"`
				Title   string `json:"title"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(feedResp.Body.Bytes(), &feedEnvelope); err != nil {
		t.Fatalf("decode feed response: %v", err)
	}
	if len(feedEnvelope.Data.Items) != 1 {
		t.Fatalf("expected 1 outbox feed item, got %d", len(feedEnvelope.Data.Items))
	}
	if feedEnvelope.Data.Items[0].VideoID != strconv.FormatInt(videoID, 10) || feedEnvelope.Data.Items[0].Title != "outbox video" {
		t.Fatalf("unexpected outbox feed item: %+v", feedEnvelope.Data.Items[0])
	}
}

func TestMessageEndpointsAfterTranscodeSuccess(t *testing.T) {
	t.Parallel()

	router, videoOSS, videoService, worker, cleanup := newTranscodeFlowTestEnv(t)
	defer cleanup()

	authorToken, _ := registerAndLogin(t, router, "author_msg_success")
	videoID := publishVideoForTest(t, router, authorToken, "msg-success.mp4", "message success video", videoOSS)

	if err := videoService.StartTranscode(t.Context(), videoID); err != nil {
		t.Fatalf("start transcode failed: %v", err)
	}

	detail, err := videoService.GetVideoForTranscode(t.Context(), videoID)
	if err != nil {
		t.Fatalf("get video for transcode failed: %v", err)
	}

	if err := videoService.CompleteTranscode(t.Context(), videoservice.CompleteTranscodeRequest{
		VideoID:    videoID,
		CoverURL:   "https://example.com/msg-cover.jpg",
		DurationMS: 8200,
		Resources: []videoservice.TranscodedResource{
			{Resolution: "720p", FileURL: "https://example.com/msg-720.mp4", FileSize: 2048, Bitrate: 1800000},
		},
	}); err != nil {
		t.Fatalf("complete transcode failed: %v", err)
	}

	worker.RunPostSuccessHooksForTest(t.Context(), detail, videoID)

	unreadResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/message/unread-count", "", authorToken)
	if unreadResp.Code != http.StatusOK {
		t.Fatalf("unexpected unread-count status: %d", unreadResp.Code)
	}

	var unreadEnvelope struct {
		Data struct {
			Items map[string]int64 `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(unreadResp.Body.Bytes(), &unreadEnvelope); err != nil {
		t.Fatalf("decode unread-count response: %v", err)
	}
	if unreadEnvelope.Data.Items["6"] != 1 {
		t.Fatalf("expected unread type 6 count to be 1, got %+v", unreadEnvelope.Data.Items)
	}

	listResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/message/list?type=6&limit=20", "", authorToken)
	if listResp.Code != http.StatusOK {
		t.Fatalf("unexpected message list status: %d", listResp.Code)
	}

	var listEnvelope struct {
		Data struct {
			Items []struct {
				ID        string `json:"id"`
				Type      int8   `json:"type"`
				RelatedID string `json:"related_id"`
				Content   string `json:"content"`
				IsRead    int8   `json:"is_read"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &listEnvelope); err != nil {
		t.Fatalf("decode message list response: %v", err)
	}
	if len(listEnvelope.Data.Items) != 1 {
		t.Fatalf("expected 1 message, got %d", len(listEnvelope.Data.Items))
	}
	messageItem := listEnvelope.Data.Items[0]
	if messageItem.Type != messagemodel.MessageTypeVideoProcessResult || messageItem.RelatedID != strconv.FormatInt(videoID, 10) || messageItem.IsRead != 0 || messageItem.Content == "" {
		t.Fatalf("unexpected message item: %+v", messageItem)
	}

	readResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/message/read", `{"msg_id":`+messageItem.ID+`}`, authorToken)
	if readResp.Code != http.StatusOK {
		t.Fatalf("unexpected message read status: %d", readResp.Code)
	}

	readAgainResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/message/read", `{"msg_id":`+messageItem.ID+`}`, authorToken)
	if readAgainResp.Code != http.StatusOK {
		t.Fatalf("unexpected repeated message read status: %d", readAgainResp.Code)
	}

	unreadAfterResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/message/unread-count", "", authorToken)
	if unreadAfterResp.Code != http.StatusOK {
		t.Fatalf("unexpected unread-count-after-read status: %d", unreadAfterResp.Code)
	}
	if err := json.Unmarshal(unreadAfterResp.Body.Bytes(), &unreadEnvelope); err != nil {
		t.Fatalf("decode unread-count-after-read response: %v", err)
	}
	if unreadEnvelope.Data.Items["6"] != 0 {
		t.Fatalf("expected unread type 6 count to be 0 after read, got %+v", unreadEnvelope.Data.Items)
	}
}

func TestMessageEndpointsAfterTranscodeFailure(t *testing.T) {
	t.Parallel()

	router, videoOSS, videoService, worker, cleanup := newTranscodeFlowTestEnv(t)
	defer cleanup()

	authorToken, _ := registerAndLogin(t, router, "author_msg_fail")
	videoID := publishVideoForTest(t, router, authorToken, "msg-fail.mp4", "message fail video", videoOSS)

	if err := videoService.StartTranscode(t.Context(), videoID); err != nil {
		t.Fatalf("start transcode failed: %v", err)
	}

	detail, err := videoService.GetVideoForTranscode(t.Context(), videoID)
	if err != nil {
		t.Fatalf("get video for transcode failed: %v", err)
	}

	if err := worker.FailVideoForTest(t.Context(), detail.UserID, videoID, "ffmpeg crashed"); err != nil {
		t.Fatalf("fail video failed: %v", err)
	}

	listResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/message/list?type=6&limit=20", "", authorToken)
	if listResp.Code != http.StatusOK {
		t.Fatalf("unexpected message list status: %d", listResp.Code)
	}

	var listEnvelope struct {
		Data struct {
			Items []struct {
				Content string `json:"content"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &listEnvelope); err != nil {
		t.Fatalf("decode message list response: %v", err)
	}
	if len(listEnvelope.Data.Items) != 1 {
		t.Fatalf("expected 1 failure message, got %d", len(listEnvelope.Data.Items))
	}
	if got := listEnvelope.Data.Items[0].Content; got == "" || got == "视频处理完成" {
		t.Fatalf("expected failure content, got %q", got)
	}

	videoDetail, err := videoService.GetVideoDetail(t.Context(), detail.UserID, videoID)
	if err != nil {
		t.Fatalf("get failed video detail: %v", err)
	}
	if videoDetail.TranscodeStatus != videomodel.TranscodeFailed || videoDetail.TranscodeFailReason == "" {
		t.Fatalf("unexpected failed video detail: %+v", videoDetail)
	}
}

func TestMessageReadDecrementsActualTypeUnreadCount(t *testing.T) {
	t.Parallel()

	router, _, _, _, cleanup := newTranscodeFlowTestEnv(t)
	defer cleanup()

	authorToken, authorID := registerAndLogin(t, router, "read_author")
	viewerToken, _ := registerAndLogin(t, router, "read_viewer")

	followResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/relation/action", `{
		"to_user_id":`+authorID+`,
		"action_type":1
	}`, viewerToken)
	if followResp.Code != http.StatusOK {
		t.Fatalf("unexpected follow status: %d", followResp.Code)
	}

	unreadResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/message/unread-count", "", authorToken)
	if unreadResp.Code != http.StatusOK {
		t.Fatalf("unexpected unread status: %d", unreadResp.Code)
	}

	var unreadEnvelope struct {
		Data struct {
			Items map[string]int64 `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(unreadResp.Body.Bytes(), &unreadEnvelope); err != nil {
		t.Fatalf("decode unread response: %v", err)
	}
	if unreadEnvelope.Data.Items["4"] != 1 {
		t.Fatalf("expected type 4 unread count to be 1, got %+v", unreadEnvelope.Data.Items)
	}

	listResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/message/list?type=4&limit=20", "", authorToken)
	if listResp.Code != http.StatusOK {
		t.Fatalf("unexpected message list status: %d", listResp.Code)
	}

	var listEnvelope struct {
		Data struct {
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &listEnvelope); err != nil {
		t.Fatalf("decode message list response: %v", err)
	}
	if len(listEnvelope.Data.Items) != 1 {
		t.Fatalf("expected 1 follower message, got %d", len(listEnvelope.Data.Items))
	}

	readResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/message/read", `{"msg_id":`+listEnvelope.Data.Items[0].ID+`}`, authorToken)
	if readResp.Code != http.StatusOK {
		t.Fatalf("unexpected message read status: %d", readResp.Code)
	}

	unreadAfterResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/message/unread-count", "", authorToken)
	if unreadAfterResp.Code != http.StatusOK {
		t.Fatalf("unexpected unread-after-read status: %d", unreadAfterResp.Code)
	}
	if err := json.Unmarshal(unreadAfterResp.Body.Bytes(), &unreadEnvelope); err != nil {
		t.Fatalf("decode unread-after-read response: %v", err)
	}
	if unreadEnvelope.Data.Items["4"] != 0 {
		t.Fatalf("expected type 4 unread count to be 0 after read, got %+v", unreadEnvelope.Data.Items)
	}
}

func newTranscodeFlowTestEnv(t *testing.T) (http.Handler, *mocks.MemoryOSSClient, *videoservice.Service, *videotranscode.Worker, func()) {
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
	messageRepo := mocks.NewMemoryMessageRepository()
	blocklist := mocks.NewMemoryTokenBlacklist()

	videoRepo := mocks.NewMemoryVideoRepository(11)
	videoOSS := mocks.NewMemoryOSSClient()
	videoService := videoservice.New(videoRepo, videoOSS, mocks.NewIncrementalIDGenerator(5000), config.Config{
		OSSUploadExpire: 15 * time.Minute,
	})
	messageSvc := messageservice.New(messageRepo, redisClient)
	relationService := relationservice.New(relationRepo, userRepo, messageSvc)
	feedSvc := feedservice.New(redisClient, relationRepo, videoService, userRepo, relationService, nil)
	recommendSvc := recommendservice.New(redisClient, videoRepo, mocks.NewMemoryInteractRepository(videoRepo, userRepo), videoService, userRepo, relationService)
	userService := userservice.New(userRepo, relationService, mocks.NewIncrementalIDGenerator(2000), jwtManager, blocklist)
	interactSvc := interactservice.New(nil, userRepo, videoRepo, messageSvc, mocks.NewIncrementalIDGenerator(8000))
	worker := videotranscode.NewWorker(config.Config{
		TranscodeMaxRetry: 1,
		TranscodeLockTTL:  time.Minute,
	}, redisClient, videoService, videoOSS, feedSvc, messageSvc)
	videoService.SetTranscodeDispatcher(worker)

	cfg := config.Config{
		CORSAllowedOrigins: []string{"http://localhost:5173"},
	}

router := httprouter.NewEngine(app.NewForTest(cfg, userService, relationService, videoService, interactSvc, feedSvc, recommendSvc, messageSvc, nil, jwtManager, blocklist))
	cleanup := func() {
		_ = redisClient.Close()
		miniRedis.Close()
	}

	return router, videoOSS, videoService, worker, cleanup
}

func registerAndLogin(t *testing.T, router http.Handler, username string) (string, string) {
	t.Helper()

	registerResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/register", `{"username":"`+username+`","password":"password123"}`, "")
	if registerResp.Code != http.StatusOK {
		t.Fatalf("unexpected register status for %s: %d", username, registerResp.Code)
	}

	loginResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/login", `{"username":"`+username+`","password":"password123"}`, "")
	if loginResp.Code != http.StatusOK {
		t.Fatalf("unexpected login status for %s: %d", username, loginResp.Code)
	}

	var loginEnvelope struct {
		Data struct {
			Token string `json:"token"`
			User  struct {
				ID string `json:"id"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginResp.Body.Bytes(), &loginEnvelope); err != nil {
		t.Fatalf("decode login response for %s: %v", username, err)
	}
	return loginEnvelope.Data.Token, loginEnvelope.Data.User.ID
}

func publishVideoForTest(t *testing.T, router http.Handler, token, fileName, title string, videoOSS *mocks.MemoryOSSClient) int64 {
	t.Helper()

	credentialResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/video/upload-credential", `{"file_name":"`+fileName+`"}`, token)
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
		"title":"`+title+`",
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
	return videoID
}

func workerFeedService(worker *videotranscode.Worker) any {
	return worker.FeedServiceForTest()
}
