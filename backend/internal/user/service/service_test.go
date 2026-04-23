package service

import (
	"context"
	"testing"
	"time"

	"github.com/AbePhh/TikTide/backend/pkg/errno"
	"github.com/AbePhh/TikTide/backend/pkg/jwt"
	"github.com/AbePhh/TikTide/backend/tests/mocks"
)

func TestRegisterAndLogin(t *testing.T) {
	t.Parallel()

	service, _ := newTestService(t)

	registerResult, err := service.Register(context.Background(), RegisterRequest{
		Username: "alice_01",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if registerResult.User.Username != "alice_01" {
		t.Fatalf("unexpected username: %s", registerResult.User.Username)
	}

	loginResult, err := service.Login(context.Background(), LoginRequest{
		Username: "alice_01",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if loginResult.Token == "" {
		t.Fatal("expected token to be issued")
	}
}

func TestChangePassword(t *testing.T) {
	t.Parallel()

	service, _ := newTestService(t)

	registerResult, err := service.Register(context.Background(), RegisterRequest{
		Username: "bob_01",
		Password: "oldpass123",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	err = service.ChangePassword(context.Background(), registerResult.User.ID, ChangePasswordRequest{
		OldPassword: "oldpass123",
		NewPassword: "newpass456",
	})
	if err != nil {
		t.Fatalf("change password failed: %v", err)
	}

	_, err = service.Login(context.Background(), LoginRequest{
		Username: "bob_01",
		Password: "oldpass123",
	})
	if !errno.IsCode(err, errno.ErrInvalidCredential.Code) {
		t.Fatalf("expected invalid credential for old password, got: %v", err)
	}

	_, err = service.Login(context.Background(), LoginRequest{
		Username: "bob_01",
		Password: "newpass456",
	})
	if err != nil {
		t.Fatalf("login with new password failed: %v", err)
	}
}

func TestBannedUserCannotLogin(t *testing.T) {
	t.Parallel()

	service, repo := newTestService(t)

	registerResult, err := service.Register(context.Background(), RegisterRequest{
		Username: "charlie_01",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	repo.BanUser(registerResult.User.ID)

	_, err = service.Login(context.Background(), LoginRequest{
		Username: "charlie_01",
		Password: "password123",
	})
	if !errno.IsCode(err, errno.ErrUserBanned.Code) {
		t.Fatalf("expected banned user error, got: %v", err)
	}
}

func newTestService(t *testing.T) (*Service, *mocks.MemoryUserRepository) {
	t.Helper()

	jwtManager, err := jwt.NewManager("tiktide-system", "tiktide-test", "tiktide-web", 24*time.Hour)
	if err != nil {
		t.Fatalf("create jwt manager: %v", err)
	}

	repo := mocks.NewMemoryUserRepository()
	blocklist := mocks.NewMemoryTokenBlacklist()
	idGenerator := mocks.NewIncrementalIDGenerator(1000)

	return New(repo, idGenerator, jwtManager, blocklist), repo
}
