package service

import (
	"context"
	"testing"

	usermodel "github.com/AbePhh/TikTide/backend/internal/user/model"
	"github.com/AbePhh/TikTide/backend/pkg/errno"
	"github.com/AbePhh/TikTide/backend/tests/mocks"
)

func TestFollowAndUnfollow(t *testing.T) {
	t.Parallel()

	relationService, userRepo := newTestService()
	createUser(t, userRepo, 1001, "alice_01")
	createUser(t, userRepo, 1002, "bob_01")

	result, err := relationService.Action(context.Background(), 1001, ActionRequest{
		ToUserID:   1002,
		ActionType: ActionFollow,
	})
	if err != nil {
		t.Fatalf("follow failed: %v", err)
	}
	if !result.IsFollowed {
		t.Fatal("expected followed state")
	}

	aliceStats, err := userRepo.GetStatsByID(context.Background(), 1001)
	if err != nil {
		t.Fatalf("load alice stats: %v", err)
	}
	if aliceStats.FollowCount != 1 {
		t.Fatalf("unexpected alice follow count: %d", aliceStats.FollowCount)
	}

	bobStats, err := userRepo.GetStatsByID(context.Background(), 1002)
	if err != nil {
		t.Fatalf("load bob stats: %v", err)
	}
	if bobStats.FollowerCount != 1 {
		t.Fatalf("unexpected bob follower count: %d", bobStats.FollowerCount)
	}

	_, err = relationService.Action(context.Background(), 1001, ActionRequest{
		ToUserID:   1002,
		ActionType: ActionUnfollow,
	})
	if err != nil {
		t.Fatalf("unfollow failed: %v", err)
	}
}

func TestMutualState(t *testing.T) {
	t.Parallel()

	relationService, userRepo := newTestService()
	createUser(t, userRepo, 1001, "alice_01")
	createUser(t, userRepo, 1002, "bob_01")

	_, err := relationService.Action(context.Background(), 1001, ActionRequest{
		ToUserID:   1002,
		ActionType: ActionFollow,
	})
	if err != nil {
		t.Fatalf("alice follow failed: %v", err)
	}

	result, err := relationService.Action(context.Background(), 1002, ActionRequest{
		ToUserID:   1001,
		ActionType: ActionFollow,
	})
	if err != nil {
		t.Fatalf("bob follow failed: %v", err)
	}
	if !result.IsMutual {
		t.Fatal("expected mutual relation")
	}

	state, err := relationService.GetRelationState(context.Background(), 1001, 1002)
	if err != nil {
		t.Fatalf("get relation state failed: %v", err)
	}
	if !state.IsFollowed || !state.IsMutual {
		t.Fatalf("unexpected relation state: %+v", state)
	}
}

func TestListFollowersAndFollowing(t *testing.T) {
	t.Parallel()

	relationService, userRepo := newTestService()
	createUser(t, userRepo, 1001, "alice_01")
	createUser(t, userRepo, 1002, "bob_01")
	createUser(t, userRepo, 1003, "charlie_01")

	_, _ = relationService.Action(context.Background(), 1001, ActionRequest{ToUserID: 1002, ActionType: ActionFollow})
	_, _ = relationService.Action(context.Background(), 1001, ActionRequest{ToUserID: 1003, ActionType: ActionFollow})
	_, _ = relationService.Action(context.Background(), 1002, ActionRequest{ToUserID: 1001, ActionType: ActionFollow})

	following, err := relationService.ListFollowing(context.Background(), 1001, 1001, ListRequest{Limit: 20})
	if err != nil {
		t.Fatalf("list following failed: %v", err)
	}
	if len(following.Items) != 2 {
		t.Fatalf("expected 2 following users, got %d", len(following.Items))
	}

	followers, err := relationService.ListFollowers(context.Background(), 1001, 1001, ListRequest{Limit: 20})
	if err != nil {
		t.Fatalf("list followers failed: %v", err)
	}
	if len(followers.Items) != 1 {
		t.Fatalf("expected 1 follower, got %d", len(followers.Items))
	}
	if !followers.Items[0].IsMutual {
		t.Fatal("expected follower to be mutual")
	}
}

func TestDuplicateFollowRejected(t *testing.T) {
	t.Parallel()

	relationService, userRepo := newTestService()
	createUser(t, userRepo, 1001, "alice_01")
	createUser(t, userRepo, 1002, "bob_01")

	_, _ = relationService.Action(context.Background(), 1001, ActionRequest{
		ToUserID:   1002,
		ActionType: ActionFollow,
	})

	_, err := relationService.Action(context.Background(), 1001, ActionRequest{
		ToUserID:   1002,
		ActionType: ActionFollow,
	})
	if !errno.IsCode(err, errno.ErrDuplicateFollow.Code) {
		t.Fatalf("expected duplicate follow error, got: %v", err)
	}
}

func newTestService() (*Service, *mocks.MemoryUserRepository) {
	userRepo := mocks.NewMemoryUserRepository()
	relationRepo := mocks.NewMemoryRelationRepository(userRepo)
	relationService := New(relationRepo, userRepo, nil)
	return relationService, userRepo
}

func createUser(t *testing.T, repo *mocks.MemoryUserRepository, id int64, username string) {
	t.Helper()

	err := repo.Create(context.Background(), &usermodel.User{
		ID:       id,
		Username: username,
		Nickname: username,
		Status:   usermodel.UserStatusActive,
	}, &usermodel.UserStats{ID: id})
	if err != nil {
		t.Fatalf("create user %s: %v", username, err)
	}
}
