package rediskey

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// JWTBlacklist returns the JWT blacklist key.
func JWTBlacklist(token string) string {
	sum := sha256.Sum256([]byte(token))
	return "jwt:blacklist:" + hex.EncodeToString(sum[:])
}

// FeedInbox returns the following-feed inbox key.
func FeedInbox(userID int64) string {
	return fmt.Sprintf("feed:inbox:%d", userID)
}

// FeedOutbox returns the author outbox key for following-feed merge.
func FeedOutbox(userID int64) string {
	return fmt.Sprintf("feed:outbox:%d", userID)
}

// FeedRecommend returns the recommend-feed cache key.
func FeedRecommend(userID int64) string {
	return fmt.Sprintf("feed:recommend:%d", userID)
}

// FeedRecommendSeen returns the recommend-feed recent seen set key.
func FeedRecommendSeen(userID int64) string {
	return fmt.Sprintf("feed:recommend:seen:%d", userID)
}

// MessageUnread returns the unread-count key for messages.
func MessageUnread(userID int64) string {
	return fmt.Sprintf("msg:unread:%d", userID)
}

// VideoStats returns the video stats cache key.
func VideoStats(videoID int64) string {
	return fmt.Sprintf("video:stats:%d", videoID)
}

// VideoLiked returns the video-like dedupe set key.
func VideoLiked(videoID int64) string {
	return fmt.Sprintf("video:liked:%d", videoID)
}

// VideoFavorited returns the video-favorite dedupe set key.
func VideoFavorited(videoID int64) string {
	return fmt.Sprintf("video:favorited:%d", videoID)
}

// VideoPlayReported returns the video-play dedupe key for a user and video.
func VideoPlayReported(userID, videoID int64) string {
	return fmt.Sprintf("video:play:reported:%d:%d", userID, videoID)
}

// CommentLiked returns the comment-like dedupe set key.
func CommentLiked(commentID int64) string {
	return fmt.Sprintf("comment:liked:%d", commentID)
}

// TranscodeLock returns the transcode lock key.
func TranscodeLock(videoID int64) string {
	return fmt.Sprintf("transcode:lock:%d", videoID)
}

// TranscodeDeadLetter returns the transcode dead-letter queue key.
func TranscodeDeadLetter() string {
	return "transcode:dead-letter"
}
