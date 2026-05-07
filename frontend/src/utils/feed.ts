import type { CommentData, FeedVideoData } from "../types/api";
import type { VideoCardModel } from "../types/models";
import { buildAvatarFallback, formatCount, formatRelativeTime } from "./format";

function formatDuration(durationMs: number): string {
  if (!durationMs || durationMs < 0) {
    return "00:00";
  }

  const totalSeconds = Math.floor(durationMs / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
}

export function mapFeedVideoToCard(video: FeedVideoData): VideoCardModel {
  const authorName = video.author?.nickname || video.author?.username || "未知作者";
  const authorHandle = `@${video.author?.username || `user${video.user_id}`}`;

  return {
    id: video.video_id,
    authorId: video.author?.id,
    authorName,
    authorHandle,
    authorAvatar: buildAvatarFallback(authorName),
    caption: video.title || "未命名视频",
    music: "",
    likes: formatCount(video.like_count),
    comments: formatCount(video.comment_count),
    favorites: formatCount(video.favorite_count),
    shares: formatCount(video.play_count),
    likeCount: video.like_count,
    commentCount: video.comment_count,
    favoriteCount: video.favorite_count,
    shareCount: video.play_count,
    cover: video.cover_url
      ? `linear-gradient(180deg, rgba(17,22,32,0.18) 0%, rgba(10,10,12,0.74) 100%), url(${video.cover_url}) center / cover`
      : "linear-gradient(180deg, rgba(18,22,33,0.18) 0%, rgba(10,10,12,0.72) 100%), radial-gradient(circle at top, #4f647f 0%, #1d2430 45%, #09090b 100%)",
    coverUrl: video.cover_url || undefined,
    tag: "",
    duration: formatDuration(video.duration_ms),
    publishedAt: formatRelativeTime(video.created_at),
    isFollowed: video.interact?.is_followed ?? false,
    isLiked: video.interact?.is_liked ?? false,
    isFavorited: video.interact?.is_favorited ?? false,
    sourceUrl: video.source_url || undefined,
    allowComment: video.allow_comment === 1
  };
}

export function updateVideoCardInteract(
  video: VideoCardModel,
  patch: Partial<Pick<VideoCardModel, "likeCount" | "commentCount" | "favoriteCount" | "isLiked" | "isFavorited">>
): VideoCardModel {
  const next = {
    ...video,
    ...patch
  };

  return {
    ...next,
    likes: formatCount(next.likeCount),
    comments: formatCount(next.commentCount),
    favorites: formatCount(next.favoriteCount)
  };
}

export function countVisibleComments(items: CommentData[]): number {
  return items.filter((item) => !item.is_deleted).length;
}
