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

func TestInteractLikeFavoriteCommentFlow(t *testing.T) {
	t.Parallel()

	router, videoOSS, videoService, cleanup := newInteractTestRouter(t)
	defer cleanup()

	authorToken, authorID := registerAndLogin(t, router, "interact_author")
	viewerToken, _ := registerAndLogin(t, router, "interact_viewer")
	videoID := publishVideoForTestWithStatus(t, router, authorToken, "interact.mp4", "interact video", videoService, videoOSS)

	likeResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/interact/like", `{"video_id":`+strconv.FormatInt(videoID, 10)+`,"action_type":1}`, viewerToken)
	if likeResp.Code != http.StatusOK {
		t.Fatalf("unexpected like status: %d", likeResp.Code)
	}

	favoriteResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/interact/favorite", `{"video_id":`+strconv.FormatInt(videoID, 10)+`,"action_type":1}`, viewerToken)
	if favoriteResp.Code != http.StatusOK {
		t.Fatalf("unexpected favorite status: %d", favoriteResp.Code)
	}

	commentResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/interact/comment/publish", `{"video_id":`+strconv.FormatInt(videoID, 10)+`,"content":"nice video","parent_id":0,"root_id":0,"to_user_id":0}`, viewerToken)
	if commentResp.Code != http.StatusOK {
		t.Fatalf("unexpected comment publish status: %d", commentResp.Code)
	}

	var commentEnvelope struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(commentResp.Body.Bytes(), &commentEnvelope); err != nil {
		t.Fatalf("decode comment response: %v", err)
	}

	commentLikeResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/interact/comment/like", `{"comment_id":`+commentEnvelope.Data.ID+`,"action_type":1}`, authorToken)
	if commentLikeResp.Code != http.StatusOK {
		t.Fatalf("unexpected comment like status: %d", commentLikeResp.Code)
	}

	favoriteListResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/interact/favorite/list?limit=20", "", viewerToken)
	if favoriteListResp.Code != http.StatusOK {
		t.Fatalf("unexpected favorite list status: %d", favoriteListResp.Code)
	}

	commentListResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/interact/comment/list?video_id="+strconv.FormatInt(videoID, 10)+"&root_id=0&limit=20", "", viewerToken)
	if commentListResp.Code != http.StatusOK {
		t.Fatalf("unexpected comment list status: %d", commentListResp.Code)
	}

	videoDetailResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/video/"+strconv.FormatInt(videoID, 10), "", viewerToken)
	if videoDetailResp.Code != http.StatusOK {
		t.Fatalf("unexpected video detail status: %d", videoDetailResp.Code)
	}

	var videoEnvelope struct {
		Data struct {
			UserID        string `json:"user_id"`
			LikeCount     int64  `json:"like_count"`
			CommentCount  int64  `json:"comment_count"`
			FavoriteCount int64  `json:"favorite_count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(videoDetailResp.Body.Bytes(), &videoEnvelope); err != nil {
		t.Fatalf("decode video detail response: %v", err)
	}
	if videoEnvelope.Data.UserID != authorID || videoEnvelope.Data.LikeCount != 1 || videoEnvelope.Data.CommentCount != 1 || videoEnvelope.Data.FavoriteCount != 1 {
		t.Fatalf("unexpected video stats: %+v", videoEnvelope.Data)
	}
}

func TestInteractUnlikeUnfavoriteAndReply(t *testing.T) {
	t.Parallel()

	router, videoOSS, videoService, cleanup := newInteractTestRouter(t)
	defer cleanup()

	authorToken, _ := registerAndLogin(t, router, "reply_author")
	viewerToken, _ := registerAndLogin(t, router, "reply_viewer")
	videoID := publishVideoForTestWithStatus(t, router, authorToken, "reply.mp4", "reply video", videoService, videoOSS)

	commentResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/interact/comment/publish", `{"video_id":`+strconv.FormatInt(videoID, 10)+`,"content":"root comment","parent_id":0,"root_id":0,"to_user_id":0}`, viewerToken)
	if commentResp.Code != http.StatusOK {
		t.Fatalf("unexpected root comment status: %d", commentResp.Code)
	}

	var rootEnvelope struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(commentResp.Body.Bytes(), &rootEnvelope); err != nil {
		t.Fatalf("decode root comment response: %v", err)
	}

	replyResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/interact/comment/publish", `{"video_id":`+strconv.FormatInt(videoID, 10)+`,"content":"reply comment","parent_id":`+rootEnvelope.Data.ID+`,"root_id":`+rootEnvelope.Data.ID+`,"to_user_id":0}`, authorToken)
	if replyResp.Code != http.StatusOK {
		t.Fatalf("unexpected reply comment status: %d", replyResp.Code)
	}

	likeResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/interact/like", `{"video_id":`+strconv.FormatInt(videoID, 10)+`,"action_type":1}`, viewerToken)
	if likeResp.Code != http.StatusOK {
		t.Fatalf("unexpected like status: %d", likeResp.Code)
	}
	unlikeResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/interact/like", `{"video_id":`+strconv.FormatInt(videoID, 10)+`,"action_type":2}`, viewerToken)
	if unlikeResp.Code != http.StatusOK {
		t.Fatalf("unexpected unlike status: %d", unlikeResp.Code)
	}

	favoriteResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/interact/favorite", `{"video_id":`+strconv.FormatInt(videoID, 10)+`,"action_type":1}`, viewerToken)
	if favoriteResp.Code != http.StatusOK {
		t.Fatalf("unexpected favorite status: %d", favoriteResp.Code)
	}
	unfavoriteResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/interact/favorite", `{"video_id":`+strconv.FormatInt(videoID, 10)+`,"action_type":2}`, viewerToken)
	if unfavoriteResp.Code != http.StatusOK {
		t.Fatalf("unexpected unfavorite status: %d", unfavoriteResp.Code)
	}
}

func TestCommentDeleteAndVisibilityRules(t *testing.T) {
	t.Parallel()

	router, videoOSS, videoService, cleanup := newInteractTestRouter(t)
	defer cleanup()

	authorToken, _ := registerAndLogin(t, router, "delete_author")
	viewerToken, _ := registerAndLogin(t, router, "delete_viewer")
	videoID := publishVideoForTestWithStatus(t, router, authorToken, "delete.mp4", "delete video", videoService, videoOSS)

	rootResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/interact/comment/publish", `{"video_id":`+strconv.FormatInt(videoID, 10)+`,"content":"root","parent_id":0,"root_id":0,"to_user_id":0}`, viewerToken)
	if rootResp.Code != http.StatusOK {
		t.Fatalf("unexpected root publish status: %d", rootResp.Code)
	}

	var rootEnvelope struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rootResp.Body.Bytes(), &rootEnvelope); err != nil {
		t.Fatalf("decode root response: %v", err)
	}

	replyResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/interact/comment/publish", `{"video_id":`+strconv.FormatInt(videoID, 10)+`,"content":"reply","parent_id":`+rootEnvelope.Data.ID+`,"root_id":`+rootEnvelope.Data.ID+`,"to_user_id":0}`, authorToken)
	if replyResp.Code != http.StatusOK {
		t.Fatalf("unexpected reply publish status: %d", replyResp.Code)
	}

	deleteResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/interact/comment/delete", `{"comment_id":`+rootEnvelope.Data.ID+`}`, viewerToken)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("unexpected delete status: %d", deleteResp.Code)
	}

	rootListResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/interact/comment/list?video_id="+strconv.FormatInt(videoID, 10)+"&root_id=0&limit=20", "", viewerToken)
	if rootListResp.Code != http.StatusOK {
		t.Fatalf("unexpected root list status: %d", rootListResp.Code)
	}

	var rootListEnvelope struct {
		Data struct {
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rootListResp.Body.Bytes(), &rootListEnvelope); err != nil {
		t.Fatalf("decode root list response: %v", err)
	}
	if len(rootListEnvelope.Data.Items) != 0 {
		t.Fatalf("expected deleted root comment to be hidden, got %+v", rootListEnvelope.Data.Items)
	}

	replyListResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/interact/comment/list?video_id="+strconv.FormatInt(videoID, 10)+"&root_id="+rootEnvelope.Data.ID+"&limit=20", "", viewerToken)
	if replyListResp.Code != http.StatusOK {
		t.Fatalf("unexpected reply list status: %d", replyListResp.Code)
	}

	var replyListEnvelope struct {
		Data struct {
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(replyListResp.Body.Bytes(), &replyListEnvelope); err != nil {
		t.Fatalf("decode reply list response: %v", err)
	}
	if len(replyListEnvelope.Data.Items) != 0 {
		t.Fatalf("expected replies under deleted root to be hidden, got %+v", replyListEnvelope.Data.Items)
	}
}

func newInteractTestRouter(t *testing.T) (http.Handler, *mocks.MemoryOSSClient, *videoservice.Service, func()) {
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
	videoRepo := mocks.NewMemoryVideoRepository(11)
	videoOSS := mocks.NewMemoryOSSClient()
	messageRepo := mocks.NewMemoryMessageRepository()
	interactRepo := mocks.NewMemoryInteractRepository(videoRepo, userRepo)
	blocklist := mocks.NewMemoryTokenBlacklist()

	messageSvc := messageservice.New(messageRepo, redisClient)
	relationService := relationservice.New(relationRepo, userRepo, messageSvc)
	userService := userservice.New(userRepo, relationService, mocks.NewIncrementalIDGenerator(2000), jwtManager, blocklist)
	videoService := videoservice.New(videoRepo, videoOSS, mocks.NewIncrementalIDGenerator(5000), config.Config{
		OSSUploadExpire: 15 * time.Minute,
	})
	feedSvc := feedservice.New(redisClient, relationRepo, videoService, userRepo, relationService, interactRepo)
	recommendSvc := recommendservice.New(redisClient, videoRepo, interactRepo, videoService, userRepo, relationService)
	interactSvc := interactservice.New(interactRepo, userRepo, videoRepo, messageSvc, mocks.NewIncrementalIDGenerator(9000))

	cfg := config.Config{
		CORSAllowedOrigins: []string{"http://localhost:5173"},
	}
router := httprouter.NewEngine(app.NewForTest(cfg, userService, relationService, videoService, interactSvc, feedSvc, recommendSvc, messageSvc, nil, jwtManager, blocklist))
	cleanup := func() {
		_ = redisClient.Close()
		miniRedis.Close()
	}
	return router, videoOSS, videoService, cleanup
}

func publishVideoForTestWithStatus(t *testing.T, router http.Handler, token, fileName, title string, videoService *videoservice.Service, videoOSS *mocks.MemoryOSSClient) int64 {
	t.Helper()
	videoID := publishVideoForTest(t, router, token, fileName, title, videoOSS)
	if err := videoService.StartTranscode(t.Context(), videoID); err != nil {
		t.Fatalf("start transcode failed: %v", err)
	}
	if err := videoService.CompleteTranscode(t.Context(), videoservice.CompleteTranscodeRequest{
		VideoID:    videoID,
		CoverURL:   "https://example.com/interact-cover.jpg",
		DurationMS: 7000,
		Resources: []videoservice.TranscodedResource{
			{Resolution: "720p", FileURL: "https://example.com/interact-720.mp4", FileSize: 2048, Bitrate: 1800000},
		},
	}); err != nil {
		t.Fatalf("complete transcode failed: %v", err)
	}
	return videoID
}
