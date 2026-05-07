package service

import (
	"context"
	"testing"
	"time"

	"github.com/AbePhh/TikTide/backend/internal/video/model"
	"github.com/AbePhh/TikTide/backend/pkg/config"
	"github.com/AbePhh/TikTide/backend/tests/mocks"
)

type stubDispatcher struct {
	dispatched []int64
}

func (s *stubDispatcher) Dispatch(_ context.Context, videoID int64) error {
	s.dispatched = append(s.dispatched, videoID)
	return nil
}

func TestPublishVideoDispatchesTranscodeTask(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMemoryVideoRepository()
	oss := mocks.NewMemoryOSSClient("video/source/1001/20260423/1.mp4")
	service := New(repo, oss, mocks.NewIncrementalIDGenerator(5000), config.Config{
		OSSUploadExpire: 15 * time.Minute,
	})

	dispatcher := &stubDispatcher{}
	service.SetTranscodeDispatcher(dispatcher)

	result, err := service.PublishVideo(context.Background(), 1001, PublishVideoRequest{
		ObjectKey:    "video/source/1001/20260423/1.mp4",
		Title:        "hello world",
		AllowComment: 1,
		Visibility:   model.VisibilityPublic,
	})
	if err != nil {
		t.Fatalf("publish video failed: %v", err)
	}

	if len(dispatcher.dispatched) != 1 {
		t.Fatalf("expected 1 dispatch, got %d", len(dispatcher.dispatched))
	}
	if dispatcher.dispatched[0] != result.VideoID {
		t.Fatalf("expected dispatched video id %d, got %d", result.VideoID, dispatcher.dispatched[0])
	}
}
