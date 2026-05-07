package transcode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	messageservice "github.com/AbePhh/TikTide/backend/internal/message/service"
	videoservice "github.com/AbePhh/TikTide/backend/internal/video/service"
	"github.com/AbePhh/TikTide/backend/pkg/config"
	"github.com/AbePhh/TikTide/backend/pkg/rediskey"
)

type Worker struct {
	cfg          config.Config
	redis        *redis.Client
	videoService videoservice.VideoService
	oss          videoservice.OSSClient
	feed         FeedDistributor
	message      MessageNotifier

	queue chan int64
	wg    sync.WaitGroup
	once  sync.Once
}

func (w *Worker) FeedServiceForTest() any {
	return w.feed
}

type FeedDistributor interface {
	DistributeVideo(ctx context.Context, videoID, authorUserID int64, createdAt time.Time) error
}

type MessageNotifier interface {
	CreateVideoProcessResult(ctx context.Context, receiverID, relatedID int64, content string) error
}

type transcodeProfile struct {
	Resolution string
	Width      int
	Height     int
	BitrateK   int
}

type ffprobeFormat struct {
	Duration string `json:"duration"`
}

type ffprobeOutput struct {
	Format ffprobeFormat `json:"format"`
}

func NewWorker(cfg config.Config, redisClient *redis.Client, videoService videoservice.VideoService, oss videoservice.OSSClient, feed FeedDistributor, message MessageNotifier) *Worker {
	queueSize := cfg.TranscodeQueueSize
	if queueSize <= 0 {
		queueSize = 128
	}

	return &Worker{
		cfg:          cfg,
		redis:        redisClient,
		videoService: videoService,
		oss:          oss,
		feed:         feed,
		message:      message,
		queue:        make(chan int64, queueSize),
	}
}

func (w *Worker) Start(ctx context.Context) {
	workers := w.cfg.TranscodeWorkers
	if workers <= 0 {
		workers = 1
	}

	for i := 0; i < workers; i++ {
		w.wg.Add(1)
		go func(workerIndex int) {
			defer w.wg.Done()
			w.run(ctx, workerIndex+1)
		}(i)
	}
}

func (w *Worker) Stop() {
	w.once.Do(func() {
		close(w.queue)
	})
	w.wg.Wait()
}

func (w *Worker) Dispatch(ctx context.Context, videoID int64) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case w.queue <- videoID:
		return nil
	default:
		return errors.New("transcode queue is full")
	}
}

func (w *Worker) run(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return
		case videoID, ok := <-w.queue:
			if !ok {
				return
			}
			if err := w.handle(ctx, videoID); err != nil {
				log.Printf("transcode worker-%d handle video %d failed: %v", workerID, videoID, err)
			}
		}
	}
}

func (w *Worker) handle(ctx context.Context, videoID int64) error {
	lockAcquired, err := w.acquireLock(ctx, videoID)
	if err != nil {
		return err
	}
	if !lockAcquired {
		return nil
	}
	defer w.releaseLock(context.Background(), videoID)

	detail, err := w.videoService.GetVideoForTranscode(ctx, videoID)
	if err != nil {
		return err
	}

	if err := w.videoService.StartTranscode(ctx, videoID); err != nil {
		return err
	}

	maxAttempts := w.maxRetry()
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		lastErr = w.processAttempt(ctx, detail, videoID)
		if lastErr == nil {
			w.runPostSuccessHooks(ctx, detail, videoID)
			return nil
		}
		log.Printf("transcode video %d attempt %d/%d failed: %v", videoID, attempt, maxAttempts, lastErr)
	}

	_ = w.failVideo(ctx, detail.UserID, videoID, lastErr.Error())
	return lastErr
}

func (w *Worker) processAttempt(ctx context.Context, detail *videoservice.VideoDetailResult, videoID int64) error {
	workDir, err := os.MkdirTemp(resolveWorkDir(w.cfg.TranscodeWorkDir), fmt.Sprintf("transcode-%d-", videoID))
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	sourcePath := filepath.Join(workDir, "source"+filepath.Ext(detail.ObjectKey))
	if err := w.downloadSource(ctx, detail.ObjectKey, sourcePath); err != nil {
		return fmt.Errorf("download source: %w", err)
	}

	durationMS, err := w.probeDuration(ctx, sourcePath)
	if err != nil {
		return fmt.Errorf("probe video: %w", err)
	}

	coverPath := filepath.Join(workDir, "cover.jpg")
	if err := w.generateCover(ctx, sourcePath, coverPath); err != nil {
		return fmt.Errorf("generate cover: %w", err)
	}

	outputs := []transcodeProfile{
		{Resolution: "480p", Width: 854, Height: 480, BitrateK: 900},
		{Resolution: "720p", Width: 1280, Height: 720, BitrateK: 1800},
		{Resolution: "1080p", Width: 1920, Height: 1080, BitrateK: 3000},
	}

	resources := make([]videoservice.TranscodedResource, 0, len(outputs))
	for _, profile := range outputs {
		outputPath := filepath.Join(workDir, profile.Resolution+".mp4")
		if err := w.generateResolution(ctx, sourcePath, outputPath, profile); err != nil {
			return fmt.Errorf("generate %s: %w", profile.Resolution, err)
		}

		objectKey := buildDerivedObjectKey(detail.ObjectKey, profile.Resolution+".mp4")
		if err := w.uploadFile(ctx, objectKey, outputPath); err != nil {
			return fmt.Errorf("upload %s: %w", profile.Resolution, err)
		}

		info, statErr := os.Stat(outputPath)
		if statErr != nil {
			return fmt.Errorf("stat %s: %w", profile.Resolution, statErr)
		}

		resources = append(resources, videoservice.TranscodedResource{
			Resolution: profile.Resolution,
			FileURL:    w.oss.ObjectURL(objectKey),
			FileSize:   info.Size(),
			Bitrate:    int32(profile.BitrateK * 1000),
		})
	}

	coverObjectKey := buildDerivedObjectKey(detail.ObjectKey, "cover.jpg")
	if err := w.uploadFile(ctx, coverObjectKey, coverPath); err != nil {
		return fmt.Errorf("upload cover: %w", err)
	}

	if err := w.videoService.CompleteTranscode(ctx, videoservice.CompleteTranscodeRequest{
		VideoID:    videoID,
		CoverURL:   w.oss.ObjectURL(coverObjectKey),
		DurationMS: durationMS,
		Resources:  resources,
	}); err != nil {
		return fmt.Errorf("complete transcode: %w", err)
	}

	return nil
}

func (w *Worker) runPostSuccessHooks(ctx context.Context, detail *videoservice.VideoDetailResult, videoID int64) {
	if w.feed != nil && detail.Visibility == 1 && detail.AuditStatus == 1 {
		if err := w.retrySideEffect(ctx, "feed distribute", func(runCtx context.Context) error {
			return w.feed.DistributeVideo(runCtx, videoID, detail.UserID, detail.CreatedAt)
		}); err != nil {
			log.Printf("transcode video %d feed distribute failed after retries: %v", videoID, err)
			w.enqueueDeadLetter(ctx, "feed_distribute", videoID, err.Error())
		}
	}

	if w.message != nil {
		if err := w.retrySideEffect(ctx, "success notify", func(runCtx context.Context) error {
			return w.message.CreateVideoProcessResult(runCtx, detail.UserID, videoID, "视频处理完成")
		}); err != nil {
			log.Printf("transcode video %d success notify failed after retries: %v", videoID, err)
			w.enqueueDeadLetter(ctx, "notify_success", videoID, err.Error())
		}
	}
}

// RunPostSuccessHooksForTest 暴露成功后置处理，便于跨包集成测试验证转码闭环。
func (w *Worker) RunPostSuccessHooksForTest(ctx context.Context, detail *videoservice.VideoDetailResult, videoID int64) {
	w.runPostSuccessHooks(ctx, detail, videoID)
}

func (w *Worker) downloadSource(ctx context.Context, objectKey, targetPath string) error {
	reader, err := w.oss.GetObject(ctx, objectKey)
	if err != nil {
		return err
	}
	defer reader.Close()

	file, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	return err
}

func (w *Worker) probeDuration(ctx context.Context, sourcePath string) (int32, error) {
	cmd := exec.CommandContext(ctx, w.cfg.FFprobePath, "-v", "error", "-print_format", "json", "-show_format", sourcePath)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	var result ffprobeOutput
	if err := json.Unmarshal(output, &result); err != nil {
		return 0, err
	}

	durationSeconds, err := strconv.ParseFloat(strings.TrimSpace(result.Format.Duration), 64)
	if err != nil {
		return 0, err
	}

	return int32(durationSeconds * 1000), nil
}

func (w *Worker) generateCover(ctx context.Context, sourcePath, coverPath string) error {
	cmd := exec.CommandContext(
		ctx,
		w.cfg.FFmpegPath,
		"-y",
		"-i", sourcePath,
		"-ss", "00:00:01",
		"-frames:v", "1",
		coverPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (w *Worker) generateResolution(ctx context.Context, sourcePath, outputPath string, profile transcodeProfile) error {
	scale := buildScaleFilter(profile)
	cmd := exec.CommandContext(
		ctx,
		w.cfg.FFmpegPath,
		"-y",
		"-i", sourcePath,
		"-vf", scale,
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-b:v", fmt.Sprintf("%dk", profile.BitrateK),
		"-maxrate", fmt.Sprintf("%dk", profile.BitrateK),
		"-bufsize", fmt.Sprintf("%dk", profile.BitrateK*2),
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func buildScaleFilter(profile transcodeProfile) string {
	return fmt.Sprintf(
		"scale=%d:%d:force_original_aspect_ratio=decrease:force_divisible_by=2",
		profile.Width,
		profile.Height,
	)
}

func (w *Worker) uploadFile(ctx context.Context, objectKey, localPath string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return w.oss.PutObject(ctx, objectKey, file)
}

func (w *Worker) failVideo(ctx context.Context, userID, videoID int64, reason string) error {
	if strings.TrimSpace(reason) == "" {
		reason = "unknown transcode error"
	}
	reason = truncateReason(reason)

	err := w.videoService.FailTranscode(ctx, videoservice.FailTranscodeRequest{
		VideoID:    videoID,
		FailReason: reason,
	})

	if w.message != nil {
		if notifyErr := w.retrySideEffect(ctx, "fail notify", func(runCtx context.Context) error {
			return w.message.CreateVideoProcessResult(runCtx, userID, videoID, "视频处理失败: "+reason)
		}); notifyErr != nil {
			log.Printf("transcode video %d fail notify failed after retries: %v", videoID, notifyErr)
		}
	}

	w.enqueueDeadLetter(ctx, "transcode_failed", videoID, reason)
	return err
}

// FailVideoForTest 暴露失败处理，便于跨包集成测试验证失败通知与状态更新。
func (w *Worker) FailVideoForTest(ctx context.Context, userID, videoID int64, reason string) error {
	return w.failVideo(ctx, userID, videoID, reason)
}

func resolveWorkDir(configured string) string {
	if strings.TrimSpace(configured) != "" {
		return configured
	}
	return os.TempDir()
}

func buildDerivedObjectKey(sourceObjectKey, fileName string) string {
	trimmed := strings.Trim(sourceObjectKey, "/")
	if index := strings.LastIndex(trimmed, "."); index >= 0 {
		trimmed = trimmed[:index]
	}
	return trimmed + "/" + fileName
}

func truncateReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if len(reason) <= 255 {
		return reason
	}
	return reason[:255]
}

func (w *Worker) acquireLock(ctx context.Context, videoID int64) (bool, error) {
	if w.redis == nil {
		return true, nil
	}

	ttl := w.cfg.TranscodeLockTTL
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return w.redis.SetNX(ctx, rediskey.TranscodeLock(videoID), "1", ttl).Result()
}

func (w *Worker) releaseLock(ctx context.Context, videoID int64) {
	if w.redis == nil {
		return
	}
	_ = w.redis.Del(ctx, rediskey.TranscodeLock(videoID)).Err()
}

func (w *Worker) retrySideEffect(ctx context.Context, name string, fn func(context.Context) error) error {
	maxAttempts := w.maxRetry()
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		lastErr = fn(ctx)
		if lastErr == nil {
			return nil
		}
		log.Printf("%s attempt %d/%d failed: %v", name, attempt, maxAttempts, lastErr)
	}
	return lastErr
}

func (w *Worker) enqueueDeadLetter(ctx context.Context, eventType string, videoID int64, reason string) {
	if w.redis == nil {
		return
	}

	payload, err := json.Marshal(map[string]any{
		"event_type": eventType,
		"video_id":   videoID,
		"reason":     truncateReason(reason),
		"created_at": time.Now().Format(time.RFC3339),
	})
	if err != nil {
		return
	}

	_ = w.redis.LPush(ctx, rediskey.TranscodeDeadLetter(), payload).Err()
}

func (w *Worker) maxRetry() int {
	if w.cfg.TranscodeMaxRetry <= 0 {
		return 1
	}
	return w.cfg.TranscodeMaxRetry
}

var _ MessageNotifier = messageservice.MessageService(nil)
