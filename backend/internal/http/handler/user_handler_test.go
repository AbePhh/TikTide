package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	"github.com/AbePhh/TikTide/backend/pkg/config"
	"github.com/AbePhh/TikTide/backend/pkg/jwt"
	"github.com/AbePhh/TikTide/backend/tests/mocks"
)

func TestUserAuthFlow(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)

	registerBody := `{"username":"web_user","password":"password123"}`
	registerResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/register", registerBody, "")
	if registerResp.Code != http.StatusOK {
		t.Fatalf("unexpected register status: %d", registerResp.Code)
	}

	loginBody := `{"username":"web_user","password":"password123"}`
	loginResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/login", loginBody, "")
	if loginResp.Code != http.StatusOK {
		t.Fatalf("unexpected login status: %d", loginResp.Code)
	}

	var loginEnvelope struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginResp.Body.Bytes(), &loginEnvelope); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if loginEnvelope.Code != 0 || loginEnvelope.Data.Token == "" {
		t.Fatalf("unexpected login envelope: %+v", loginEnvelope)
	}

	token := loginEnvelope.Data.Token
	profileResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/user/profile", "", token)
	if profileResp.Code != http.StatusOK {
		t.Fatalf("unexpected profile status: %d", profileResp.Code)
	}

	logoutResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/logout", "{}", token)
	if logoutResp.Code != http.StatusOK {
		t.Fatalf("unexpected logout status: %d", logoutResp.Code)
	}

	profileAfterLogout := performJSONRequest(t, router, http.MethodGet, "/api/v1/user/profile", "", token)
	if profileAfterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized after logout, got: %d", profileAfterLogout.Code)
	}
}

func newTestRouter(t *testing.T) http.Handler {
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

	repo := mocks.NewMemoryUserRepository()
	relationRepo := mocks.NewMemoryRelationRepository(repo)
	blocklist := mocks.NewMemoryTokenBlacklist()
	idGenerator := mocks.NewIncrementalIDGenerator(2000)
	messageRepo := mocks.NewMemoryMessageRepository()
	messageSvc := messageservice.New(messageRepo, redisClient)
	relationService := relationservice.New(relationRepo, repo, messageSvc)
	userService := userservice.New(repo, relationService, idGenerator, jwtManager, blocklist)
	interactSvc := interactservice.New(nil, nil, nil, messageSvc, nil)
	feedSvc := feedservice.New(redisClient, relationRepo, nil, repo, relationService, nil)
	recommendSvc := recommendservice.New(redisClient, mocks.NewMemoryVideoRepository(), mocks.NewMemoryInteractRepository(mocks.NewMemoryVideoRepository(), repo), nil, repo, relationService)
	cfg := config.Config{
		CORSAllowedOrigins: []string{"http://localhost:5173"},
	}

	t.Cleanup(func() {
		_ = redisClient.Close()
		miniRedis.Close()
	})

return httprouter.NewEngine(app.NewForTest(cfg, userService, relationService, nil, interactSvc, feedSvc, recommendSvc, messageSvc, nil, jwtManager, blocklist))
}

func performJSONRequest(t *testing.T, router http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}

	request := httptest.NewRequest(method, path, reader)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func int64ToString(value int64) string {
	return strconv.FormatInt(value, 10)
}
