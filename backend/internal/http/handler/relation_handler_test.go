package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestRelationEndpoints(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)

	registerAlice := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/register", `{"username":"alice_rel","password":"password123"}`, "")
	if registerAlice.Code != http.StatusOK {
		t.Fatalf("unexpected register status for alice: %d", registerAlice.Code)
	}

	registerBob := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/register", `{"username":"bob_rel","password":"password123"}`, "")
	if registerBob.Code != http.StatusOK {
		t.Fatalf("unexpected register status for bob: %d", registerBob.Code)
	}

	loginAlice := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/login", `{"username":"alice_rel","password":"password123"}`, "")
	loginBob := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/login", `{"username":"bob_rel","password":"password123"}`, "")

	var aliceEnvelope struct {
		Data struct {
			Token string `json:"token"`
			User  struct {
				ID string `json:"id"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginAlice.Body.Bytes(), &aliceEnvelope); err != nil {
		t.Fatalf("decode alice login response: %v", err)
	}

	var bobEnvelope struct {
		Data struct {
			Token string `json:"token"`
			User  struct {
				ID string `json:"id"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginBob.Body.Bytes(), &bobEnvelope); err != nil {
		t.Fatalf("decode bob login response: %v", err)
	}

	followResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/relation/action", `{
		"to_user_id":`+bobEnvelope.Data.User.ID+`,
		"action_type":1
	}`, aliceEnvelope.Data.Token)
	if followResp.Code != http.StatusOK {
		t.Fatalf("unexpected follow status: %d", followResp.Code)
	}

	homepageResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/user/"+bobEnvelope.Data.User.ID, "", aliceEnvelope.Data.Token)
	if homepageResp.Code != http.StatusOK {
		t.Fatalf("unexpected homepage status: %d", homepageResp.Code)
	}

	var homepageEnvelope struct {
		Data struct {
			IsFollowed bool `json:"is_followed"`
			IsMutual   bool `json:"is_mutual"`
		} `json:"data"`
	}
	if err := json.Unmarshal(homepageResp.Body.Bytes(), &homepageEnvelope); err != nil {
		t.Fatalf("decode homepage response: %v", err)
	}
	if !homepageEnvelope.Data.IsFollowed {
		t.Fatal("expected alice to follow bob")
	}

	followersResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/relation/followers/"+bobEnvelope.Data.User.ID, "", aliceEnvelope.Data.Token)
	if followersResp.Code != http.StatusOK {
		t.Fatalf("unexpected followers status: %d", followersResp.Code)
	}

	followingResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/relation/following/"+aliceEnvelope.Data.User.ID, "", aliceEnvelope.Data.Token)
	if followingResp.Code != http.StatusOK {
		t.Fatalf("unexpected following status: %d", followingResp.Code)
	}

	reverseFollowResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/relation/action", `{
		"to_user_id":`+aliceEnvelope.Data.User.ID+`,
		"action_type":1
	}`, bobEnvelope.Data.Token)
	if reverseFollowResp.Code != http.StatusOK {
		t.Fatalf("unexpected reverse follow status: %d", reverseFollowResp.Code)
	}

	mutualHomepageResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/user/"+aliceEnvelope.Data.User.ID, "", bobEnvelope.Data.Token)
	if mutualHomepageResp.Code != http.StatusOK {
		t.Fatalf("unexpected mutual homepage status: %d", mutualHomepageResp.Code)
	}

	if err := json.Unmarshal(mutualHomepageResp.Body.Bytes(), &homepageEnvelope); err != nil {
		t.Fatalf("decode mutual homepage response: %v", err)
	}
	if !homepageEnvelope.Data.IsMutual {
		t.Fatal("expected mutual follow state")
	}

	unfollowResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/relation/action", `{
		"to_user_id":`+bobEnvelope.Data.User.ID+`,
		"action_type":2
	}`, aliceEnvelope.Data.Token)
	if unfollowResp.Code != http.StatusOK {
		t.Fatalf("unexpected unfollow status: %d", unfollowResp.Code)
	}
}

func TestFollowCreatesNotification(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)

	registerAlice := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/register", `{"username":"alice_notify","password":"password123"}`, "")
	if registerAlice.Code != http.StatusOK {
		t.Fatalf("unexpected register status for alice: %d", registerAlice.Code)
	}

	registerBob := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/register", `{"username":"bob_notify","password":"password123"}`, "")
	if registerBob.Code != http.StatusOK {
		t.Fatalf("unexpected register status for bob: %d", registerBob.Code)
	}

	loginAlice := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/login", `{"username":"alice_notify","password":"password123"}`, "")
	loginBob := performJSONRequest(t, router, http.MethodPost, "/api/v1/user/login", `{"username":"bob_notify","password":"password123"}`, "")

	var aliceEnvelope struct {
		Data struct {
			Token string `json:"token"`
			User  struct {
				ID string `json:"id"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginAlice.Body.Bytes(), &aliceEnvelope); err != nil {
		t.Fatalf("decode alice login response: %v", err)
	}

	var bobEnvelope struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginBob.Body.Bytes(), &bobEnvelope); err != nil {
		t.Fatalf("decode bob login response: %v", err)
	}

	followResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/relation/action", `{
		"to_user_id":`+aliceEnvelope.Data.User.ID+`,
		"action_type":1
	}`, bobEnvelope.Data.Token)
	if followResp.Code != http.StatusOK {
		t.Fatalf("unexpected follow status: %d", followResp.Code)
	}

	unreadResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/message/unread-count", "", aliceEnvelope.Data.Token)
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
		t.Fatalf("expected follower unread count to be 1, got %+v", unreadEnvelope.Data.Items)
	}
}
