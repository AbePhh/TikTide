package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/AbePhh/TikTide/backend/internal/app"
	httprouter "github.com/AbePhh/TikTide/backend/internal/http/router"
	userservice "github.com/AbePhh/TikTide/backend/internal/user/service"
	videoservice "github.com/AbePhh/TikTide/backend/internal/video/service"
	"github.com/AbePhh/TikTide/backend/pkg/config"
	"github.com/AbePhh/TikTide/backend/pkg/jwt"
	"github.com/AbePhh/TikTide/backend/tests/mocks"
)

func TestVideoPublishFlow(t *testing.T) {
	t.Parallel()

	router, videoOSS := newVideoTestRouter(t)

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

	router, videoOSS := newVideoTestRouter(t)

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

	hashtagVideosResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/hashtag/11/videos?limit=20", "", token)
	if hashtagVideosResp.Code != http.StatusOK {
		t.Fatalf("unexpected hashtag videos status: %d", hashtagVideosResp.Code)
	}
}

func TestCreateHashtagEndpoint(t *testing.T) {
	t.Parallel()

	router, _ := newVideoTestRouter(t)

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

func newVideoTestRouter(t *testing.T) (http.Handler, *mocks.MemoryOSSClient) {
	t.Helper()

	jwtManager, err := jwt.NewManager("tiktide-system", "tiktide-test", "tiktide-web", 24*time.Hour)
	if err != nil {
		t.Fatalf("create jwt manager: %v", err)
	}

	userRepo := mocks.NewMemoryUserRepository()
	blocklist := mocks.NewMemoryTokenBlacklist()
	idGenerator := mocks.NewIncrementalIDGenerator(2000)
	userService := userservice.New(userRepo, idGenerator, jwtManager, blocklist)

	videoRepo := mocks.NewMemoryVideoRepository(11)
	videoOSS := mocks.NewMemoryOSSClient()
	videoService := videoservice.New(videoRepo, videoOSS, mocks.NewIncrementalIDGenerator(5000), config.Config{
		OSSUploadExpire: 15 * time.Minute,
	})

	cfg := config.Config{
		CORSAllowedOrigins: []string{"http://localhost:5173"},
	}

	return httprouter.NewEngine(app.NewForTest(cfg, userService, videoService, jwtManager, blocklist)), videoOSS
}
