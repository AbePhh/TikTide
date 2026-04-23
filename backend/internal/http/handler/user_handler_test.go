package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AbePhh/TikTide/backend/internal/app"
	httprouter "github.com/AbePhh/TikTide/backend/internal/http/router"
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

	repo := mocks.NewMemoryUserRepository()
	blocklist := mocks.NewMemoryTokenBlacklist()
	idGenerator := mocks.NewIncrementalIDGenerator(2000)
	userService := userservice.New(repo, idGenerator, jwtManager, blocklist)
	cfg := config.Config{
		CORSAllowedOrigins: []string{"http://localhost:5173"},
	}

	return httprouter.NewEngine(app.NewForTest(cfg, userService, jwtManager, blocklist))
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
