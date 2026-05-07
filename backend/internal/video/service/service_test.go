package service

import (
	"context"
	"errors"
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
		FileName:    "demo.mp4",
		ContentType: "video/mp4",
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
	if result.ContentType != "video/mp4" {
		t.Fatalf("unexpected content type: %s", result.ContentType)
	}
}

func TestPublishVideoSuccess(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMemoryVideoRepository(11, 22)
	oss := mocks.NewMemoryOSSClient("video/source/1001/20260423/1.mp4")
	service := New(repo, oss, mocks.NewIncrementalIDGenerator(5000), config.Config{
		OSSUploadExpire: 15 * time.Minute,
		OSSReadExpire:   15 * time.Minute,
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
		OSSReadExpire:   15 * time.Minute,
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
		OSSReadExpire:   15 * time.Minute,
	})

	publishResult, err := service.PublishVideo(context.Background(), 1001, PublishVideoRequest{
		ObjectKey:    "video/source/1001/20260423/1.mp4",
		Title:        "hello world",
		HashtagIDs:   []int64{11},
		AllowComment: 1,
		Visibility:   model.VisibilityPublic,
	})
	if err != nil {
		t.Fatalf("publish video failed: %v", err)
	}

	if err := service.StartTranscode(context.Background(), publishResult.VideoID); err != nil {
		t.Fatalf("start transcode failed: %v", err)
	}
	if err := service.CompleteTranscode(context.Background(), CompleteTranscodeRequest{
		VideoID:    publishResult.VideoID,
		CoverURL:   "https://example.com/cover.jpg",
		DurationMS: 1000,
		Resources: []TranscodedResource{
			{Resolution: "720p", FileURL: "https://example.com/720.m3u8", FileSize: 100, Bitrate: 800},
		},
	}); err != nil {
		t.Fatalf("complete transcode failed: %v", err)
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
		OSSReadExpire:   15 * time.Minute,
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

func TestSaveAndListDrafts(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMemoryVideoRepository()
	oss := mocks.NewMemoryOSSClient()
	service := New(repo, oss, mocks.NewIncrementalIDGenerator(5000), config.Config{})

	draft, err := service.SaveDraft(context.Background(), 1001, SaveDraftRequest{
		ObjectKey:    "video/source/1001/20260424/draft.mp4",
		CoverURL:     "video/source/1001/20260424/draft/cover.jpg",
		Title:        "draft title",
		TagNames:     "travel,sunset",
		AllowComment: 1,
		Visibility:   model.VisibilityPrivate,
	})
	if err != nil {
		t.Fatalf("save draft failed: %v", err)
	}
	if draft.ID == 0 {
		t.Fatal("expected draft id")
	}
	storedDraft, err := repo.GetDraft(context.Background(), 1001, draft.ID)
	if err != nil {
		t.Fatalf("get draft from repo failed: %v", err)
	}
	if storedDraft.CoverURL != "https://example.com/object/video/source/1001/20260424/draft/cover.jpg" {
		t.Fatalf("expected full cover url stored in repo, got %s", storedDraft.CoverURL)
	}
	if !strings.Contains(draft.CoverURL, "video/source/1001/20260424/draft/cover.jpg?signature=demo") {
		t.Fatalf("expected signed draft cover url, got %s", draft.CoverURL)
	}

	listResult, err := service.ListDrafts(context.Background(), 1001)
	if err != nil {
		t.Fatalf("list drafts failed: %v", err)
	}
	if len(listResult.Items) != 1 {
		t.Fatalf("expected 1 draft, got %d", len(listResult.Items))
	}
	if !strings.Contains(listResult.Items[0].CoverURL, "video/source/1001/20260424/draft/cover.jpg?signature=demo") {
		t.Fatalf("expected signed cover url in draft list, got %s", listResult.Items[0].CoverURL)
	}
}

func TestSaveDraftAutoDerivesCoverObjectKey(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMemoryVideoRepository()
	oss := mocks.NewMemoryOSSClient()
	service := New(repo, oss, mocks.NewIncrementalIDGenerator(5000), config.Config{})

	draft, err := service.SaveDraft(context.Background(), 1001, SaveDraftRequest{
		ObjectKey:    "video/source/1001/20260424/draft.mp4",
		Title:        "draft title",
		AllowComment: 1,
		Visibility:   model.VisibilityPrivate,
	})
	if err != nil {
		t.Fatalf("save draft failed: %v", err)
	}
	storedDraft, err := repo.GetDraft(context.Background(), 1001, draft.ID)
	if err != nil {
		t.Fatalf("get draft from repo failed: %v", err)
	}
	if storedDraft.CoverURL != "https://example.com/object/video/source/1001/20260424/draft/cover.jpg" {
		t.Fatalf("expected derived full cover url stored in repo, got %s", storedDraft.CoverURL)
	}

	if !strings.Contains(draft.CoverURL, "video/source/1001/20260424/draft/cover.jpg?signature=demo") {
		t.Fatalf("expected derived signed cover url, got %s", draft.CoverURL)
	}
}

func TestBuildDraftResultSupportsStoredStorageURL(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMemoryVideoRepository()
	oss := mocks.NewMemoryOSSClient()
	service := New(repo, oss, mocks.NewIncrementalIDGenerator(5000), config.Config{
		OSSEndpoint: "https://tiktide.s3.cn-east-1.qiniucs.com",
		OSSBucket:   "tiktide",
	})

	draft, err := service.SaveDraft(context.Background(), 1001, SaveDraftRequest{
		ObjectKey:    "video/source/1001/20260424/draft.mp4",
		CoverURL:     "https://tiktide.s3.cn-east-1.qiniucs.com/video/source/1001/20260424/draft/cover.jpg",
		Title:        "draft title",
		AllowComment: 1,
		Visibility:   model.VisibilityPrivate,
	})
	if err != nil {
		t.Fatalf("save draft failed: %v", err)
	}

	if !strings.Contains(draft.CoverURL, "video/source/1001/20260424/draft/cover.jpg?signature=demo") {
		t.Fatalf("expected signed cover url for stored storage url, got %s", draft.CoverURL)
	}
}

func TestDeleteDraft(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMemoryVideoRepository()
	oss := mocks.NewMemoryOSSClient()
	service := New(repo, oss, mocks.NewIncrementalIDGenerator(5000), config.Config{})

	draft, err := service.SaveDraft(context.Background(), 1001, SaveDraftRequest{
		ObjectKey:    "video/source/1001/20260424/draft.mp4",
		Title:        "draft title",
		AllowComment: 1,
		Visibility:   model.VisibilityPrivate,
	})
	if err != nil {
		t.Fatalf("save draft failed: %v", err)
	}

	if err := service.DeleteDraft(context.Background(), 1001, draft.ID); err != nil {
		t.Fatalf("delete draft failed: %v", err)
	}

	listResult, err := service.ListDrafts(context.Background(), 1001)
	if err != nil {
		t.Fatalf("list drafts failed: %v", err)
	}
	if len(listResult.Items) != 0 {
		t.Fatalf("expected 0 drafts, got %d", len(listResult.Items))
	}
}

func TestGetVideoDetailForOwnerBeforeTranscodeFinished(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMemoryVideoRepository()
	oss := mocks.NewMemoryOSSClient("video/source/1001/20260423/1.mp4")
	service := New(repo, oss, mocks.NewIncrementalIDGenerator(5000), config.Config{
		OSSUploadExpire: 15 * time.Minute,
		OSSReadExpire:   15 * time.Minute,
	})

	publish, err := service.PublishVideo(context.Background(), 1001, PublishVideoRequest{
		ObjectKey:    "video/source/1001/20260423/1.mp4",
		Title:        "hello world",
		AllowComment: 1,
		Visibility:   model.VisibilityPrivate,
	})
	if err != nil {
		t.Fatalf("publish video failed: %v", err)
	}

	detail, err := service.GetVideoDetail(context.Background(), 1001, publish.VideoID)
	if err != nil {
		t.Fatalf("get detail failed: %v", err)
	}
	if detail.TranscodeStatus != model.TranscodePending {
		t.Fatalf("unexpected transcode status: %d", detail.TranscodeStatus)
	}
}

func TestGetVideoDetailForPublicViewerRequiresSuccessfulTranscode(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMemoryVideoRepository()
	oss := mocks.NewMemoryOSSClient("video/source/1001/20260423/1.mp4")
	service := New(repo, oss, mocks.NewIncrementalIDGenerator(5000), config.Config{
		OSSUploadExpire: 15 * time.Minute,
		OSSReadExpire:   15 * time.Minute,
	})

	publish, err := service.PublishVideo(context.Background(), 1001, PublishVideoRequest{
		ObjectKey:    "video/source/1001/20260423/1.mp4",
		Title:        "hello world",
		AllowComment: 1,
		Visibility:   model.VisibilityPublic,
	})
	if err != nil {
		t.Fatalf("publish video failed: %v", err)
	}

	_, err = service.GetVideoDetail(context.Background(), 2001, publish.VideoID)
	if !errno.IsCode(err, errno.ErrVideoTranscoding.Code) {
		t.Fatalf("expected video transcoding error, got: %v", err)
	}
}

func TestTranscodeFlowAndResourceQuery(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMemoryVideoRepository()
	oss := mocks.NewMemoryOSSClient("video/source/1001/20260423/1.mp4")
	service := New(repo, oss, mocks.NewIncrementalIDGenerator(5000), config.Config{
		OSSUploadExpire: 15 * time.Minute,
	})

	publish, err := service.PublishVideo(context.Background(), 1001, PublishVideoRequest{
		ObjectKey:    "video/source/1001/20260423/1.mp4",
		Title:        "hello world",
		AllowComment: 1,
		Visibility:   model.VisibilityPublic,
	})
	if err != nil {
		t.Fatalf("publish video failed: %v", err)
	}

	if err := service.StartTranscode(context.Background(), publish.VideoID); err != nil {
		t.Fatalf("start transcode failed: %v", err)
	}

	err = service.CompleteTranscode(context.Background(), CompleteTranscodeRequest{
		VideoID:    publish.VideoID,
		CoverURL:   "https://example.com/cover.jpg",
		DurationMS: 12345,
		Resources: []TranscodedResource{
			{Resolution: "720p", FileURL: "https://example.com/720.m3u8", FileSize: 2048, Bitrate: 1500},
			{Resolution: "1080p", FileURL: "https://example.com/1080.m3u8", FileSize: 4096, Bitrate: 2400},
		},
	})
	if err != nil {
		t.Fatalf("complete transcode failed: %v", err)
	}

	detail, err := service.GetVideoDetail(context.Background(), 2001, publish.VideoID)
	if err != nil {
		t.Fatalf("public get detail failed: %v", err)
	}
	if detail.CoverURL == "" || detail.DurationMS != 12345 || detail.TranscodeStatus != model.TranscodeSuccess {
		t.Fatalf("unexpected detail: %+v", detail)
	}
	if !strings.Contains(detail.CoverURL, "?signature=demo") {
		t.Fatalf("expected signed cover url, got %s", detail.CoverURL)
	}
	if !strings.Contains(detail.SourceURL, "?signature=demo") {
		t.Fatalf("expected signed source url, got %s", detail.SourceURL)
	}

	resources, err := service.GetVideoResources(context.Background(), 2001, publish.VideoID)
	if err != nil {
		t.Fatalf("get resources failed: %v", err)
	}
	if len(resources.Items) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(resources.Items))
	}
	if resources.Items[0].Resolution != "1080p" {
		t.Fatalf("expected highest resolution first, got %s", resources.Items[0].Resolution)
	}
	if !strings.Contains(resources.Items[0].FileURL, "?signature=demo") {
		t.Fatalf("expected signed resource url, got %s", resources.Items[0].FileURL)
	}
}

func TestFailTranscode(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMemoryVideoRepository()
	oss := mocks.NewMemoryOSSClient("video/source/1001/20260423/1.mp4")
	service := New(repo, oss, mocks.NewIncrementalIDGenerator(5000), config.Config{
		OSSUploadExpire: 15 * time.Minute,
		OSSReadExpire:   15 * time.Minute,
	})

	publish, err := service.PublishVideo(context.Background(), 1001, PublishVideoRequest{
		ObjectKey:    "video/source/1001/20260423/1.mp4",
		Title:        "hello world",
		AllowComment: 1,
		Visibility:   model.VisibilityPublic,
	})
	if err != nil {
		t.Fatalf("publish video failed: %v", err)
	}

	if err := service.StartTranscode(context.Background(), publish.VideoID); err != nil {
		t.Fatalf("start transcode failed: %v", err)
	}
	if err := service.FailTranscode(context.Background(), FailTranscodeRequest{
		VideoID:    publish.VideoID,
		FailReason: "ffmpeg exited with code 1",
	}); err != nil {
		t.Fatalf("fail transcode failed: %v", err)
	}

	_, err = service.GetVideoDetail(context.Background(), 2001, publish.VideoID)
	if !errno.IsCode(err, errno.ErrVideoTranscodeFailed.Code) {
		t.Fatalf("expected transcode failed error, got: %v", err)
	}
}

func TestStartTranscodeRejectsInvalidState(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMemoryVideoRepository()
	oss := mocks.NewMemoryOSSClient("video/source/1001/20260423/1.mp4")
	service := New(repo, oss, mocks.NewIncrementalIDGenerator(5000), config.Config{
		OSSUploadExpire: 15 * time.Minute,
		OSSReadExpire:   15 * time.Minute,
	})

	publish, err := service.PublishVideo(context.Background(), 1001, PublishVideoRequest{
		ObjectKey:    "video/source/1001/20260423/1.mp4",
		Title:        "hello world",
		AllowComment: 1,
		Visibility:   model.VisibilityPublic,
	})
	if err != nil {
		t.Fatalf("publish video failed: %v", err)
	}

	if err := service.StartTranscode(context.Background(), publish.VideoID); err != nil {
		t.Fatalf("start transcode failed: %v", err)
	}
	err = service.StartTranscode(context.Background(), publish.VideoID)
	if !errors.Is(err, model.ErrInvalidTranscodeState) {
		t.Fatalf("expected invalid transcode state, got: %v", err)
	}
}

func newTestVideoService() *Service {
	repo := mocks.NewMemoryVideoRepository(11, 22)
	oss := mocks.NewMemoryOSSClient("video/source/1001/20260423/1.mp4")
	return New(repo, oss, mocks.NewIncrementalIDGenerator(5000), config.Config{
		OSSUploadExpire: 15 * time.Minute,
		OSSReadExpire:   15 * time.Minute,
	})
}
