package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/AbePhh/TikTide/backend/internal/video/model"
	"github.com/AbePhh/TikTide/backend/pkg/config"
	"github.com/AbePhh/TikTide/backend/pkg/errno"
	"github.com/AbePhh/TikTide/backend/tests/mocks"
)

func TestCreateUploadCredential(t *testing.T) {
	t.Parallel()

	service := newTestVideoService()
	result, err := service.CreateUploadCredential(context.Background(), 1001, CreateUploadCredentialRequest{
		FileName: "demo.mp4",
	})
	if err != nil {
		t.Fatalf("create upload credential failed: %v", err)
	}
	if result.ObjectKey == "" || !strings.HasSuffix(result.ObjectKey, ".mp4") {
		t.Fatalf("unexpected object key: %s", result.ObjectKey)
	}
	if result.UploadURL == "" {
		t.Fatal("expected signed upload url")
	}
}

func TestPublishVideoSuccess(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMemoryVideoRepository(11, 22)
	oss := mocks.NewMemoryOSSClient("video/source/1001/20260423/1.mp4")
	service := New(repo, oss, mocks.NewIncrementalIDGenerator(5000), config.Config{
		OSSUploadExpire: 15 * time.Minute,
	})

	result, err := service.PublishVideo(context.Background(), 1001, PublishVideoRequest{
		ObjectKey:    "video/source/1001/20260423/1.mp4",
		Title:        "hello world",
		HashtagIDs:   []int64{11, 22},
		AllowComment: 1,
		Visibility:   model.VisibilityPublic,
	})
	if err != nil {
		t.Fatalf("publish video failed: %v", err)
	}
	if result.VideoID == 0 {
		t.Fatal("expected video id")
	}
	if result.TranscodeStatus != model.TranscodePending {
		t.Fatalf("unexpected transcode status: %d", result.TranscodeStatus)
	}
}

func TestPublishVideoWithoutObject(t *testing.T) {
	t.Parallel()

	service := newTestVideoService()
	_, err := service.PublishVideo(context.Background(), 1001, PublishVideoRequest{
		ObjectKey:    "video/source/1001/20260423/not-found.mp4",
		Title:        "hello world",
		AllowComment: 1,
		Visibility:   model.VisibilityPublic,
	})
	if !errno.IsCode(err, errno.ErrUploadObjectNotFound.Code) {
		t.Fatalf("expected upload object not found, got: %v", err)
	}
}

func TestPublishVideoWithUnknownHashtag(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMemoryVideoRepository(11)
	oss := mocks.NewMemoryOSSClient("video/source/1001/20260423/1.mp4")
	service := New(repo, oss, mocks.NewIncrementalIDGenerator(5000), config.Config{
		OSSUploadExpire: 15 * time.Minute,
	})

	_, err := service.PublishVideo(context.Background(), 1001, PublishVideoRequest{
		ObjectKey:    "video/source/1001/20260423/1.mp4",
		Title:        "hello world",
		HashtagIDs:   []int64{11, 99},
		AllowComment: 1,
		Visibility:   model.VisibilityPublic,
	})
	if !errno.IsCode(err, errno.ErrHashtagNotFound.Code) {
		t.Fatalf("expected hashtag not found, got: %v", err)
	}
}

func TestGetHashtag(t *testing.T) {
	t.Parallel()

	service := newTestVideoService()
	result, err := service.GetHashtag(context.Background(), 11)
	if err != nil {
		t.Fatalf("get hashtag failed: %v", err)
	}
	if result.ID != 11 {
		t.Fatalf("unexpected hashtag id: %d", result.ID)
	}
}

func TestCreateHashtag(t *testing.T) {
	t.Parallel()

	service := newTestVideoService()
	result, err := service.CreateHashtag(context.Background(), 1001, CreateHashtagRequest{
		Name: "travel",
	})
	if err != nil {
		t.Fatalf("create hashtag failed: %v", err)
	}
	if result.Name != "travel" {
		t.Fatalf("unexpected hashtag name: %s", result.Name)
	}
}

func TestListHashtagVideos(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMemoryVideoRepository(11)
	oss := mocks.NewMemoryOSSClient("video/source/1001/20260423/1.mp4")
	service := New(repo, oss, mocks.NewIncrementalIDGenerator(5000), config.Config{
		OSSUploadExpire: 15 * time.Minute,
	})

	_, err := service.PublishVideo(context.Background(), 1001, PublishVideoRequest{
		ObjectKey:    "video/source/1001/20260423/1.mp4",
		Title:        "hello world",
		HashtagIDs:   []int64{11},
		AllowComment: 1,
		Visibility:   model.VisibilityPublic,
	})
	if err != nil {
		t.Fatalf("publish video failed: %v", err)
	}

	listResult, err := service.ListHashtagVideos(context.Background(), 11, ListHashtagVideosRequest{
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("list hashtag videos failed: %v", err)
	}
	if len(listResult.Items) != 1 {
		t.Fatalf("expected 1 video, got: %d", len(listResult.Items))
	}
}

func TestPublishVideoAutoCreateHashtags(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMemoryVideoRepository()
	oss := mocks.NewMemoryOSSClient("video/source/1001/20260423/1.mp4")
	service := New(repo, oss, mocks.NewIncrementalIDGenerator(5000), config.Config{
		OSSUploadExpire: 15 * time.Minute,
	})

	result, err := service.PublishVideo(context.Background(), 1001, PublishVideoRequest{
		ObjectKey:    "video/source/1001/20260423/1.mp4",
		Title:        "hello world",
		HashtagNames: []string{"travel", "#sunset"},
		AllowComment: 1,
		Visibility:   model.VisibilityPublic,
	})
	if err != nil {
		t.Fatalf("publish video with auto hashtags failed: %v", err)
	}
	if result.VideoID == 0 {
		t.Fatal("expected video id")
	}
}

func newTestVideoService() *Service {
	repo := mocks.NewMemoryVideoRepository(11, 22)
	oss := mocks.NewMemoryOSSClient("video/source/1001/20260423/1.mp4")
	return New(repo, oss, mocks.NewIncrementalIDGenerator(5000), config.Config{
		OSSUploadExpire: 15 * time.Minute,
	})
}
