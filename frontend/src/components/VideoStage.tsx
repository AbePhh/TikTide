import { type MouseEvent, useEffect, useMemo, useRef, useState } from "react";
import { Link } from "react-router-dom";

import { favoriteVideo, getVideoResources, likeComment, likeVideo, listComments, publishComment, reportVideoPlay } from "../api/client";
import type { CommentData } from "../types/api";
import type { VideoCardModel } from "../types/models";
import { countVisibleComments, updateVideoCardInteract } from "../utils/feed";
import { formatCount, formatRelativeTime } from "../utils/format";

interface VideoStageProps {
  video: VideoCardModel;
  onChange?: (next: VideoCardModel) => void;
  onActivate?: (videoID: string) => void;
}

interface IconProps {
  filled?: boolean;
}

const PLAYBACK_RATES = [1, 1.25, 1.5, 2];

function formatClock(seconds: number) {
  const safeSeconds = Math.max(0, Math.floor(seconds));
  const minutes = Math.floor(safeSeconds / 60);
  const remains = safeSeconds % 60;
  return `${String(minutes).padStart(2, "0")}:${String(remains).padStart(2, "0")}`;
}

function formatPlaybackRate(rate: number) {
  return `${Number(rate.toFixed(2)).toString()}x`;
}

function HeartIcon({ filled = false }: IconProps) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path
        d="M12 21s-6.716-4.35-9.192-8.17C1.14 10.26 1.55 6.91 4.352 5.1c2.214-1.43 5.108-.89 6.77 1.05L12 7.24l.878-1.09c1.662-1.94 4.556-2.48 6.77-1.05 2.802 1.81 3.211 5.16 1.544 7.73C18.716 16.65 12 21 12 21Z"
        fill={filled ? "currentColor" : "none"}
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function CommentIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path
        d="M7 18.5 3.5 21v-4.35A7.5 7.5 0 1 1 19 16.5H7Z"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function BookmarkIcon({ filled = false }: IconProps) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path
        d="M7 4.75h10a1.25 1.25 0 0 1 1.25 1.25v13.5L12 16.1 5.75 19.5V6A1.25 1.25 0 0 1 7 4.75Z"
        fill={filled ? "currentColor" : "none"}
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function ShareIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path
        d="m13.5 4.5 6 6-6 6"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M19 10.5h-6.5a6 6 0 0 0-6 6V18"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function PlayIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path d="m8 6 10 6-10 6V6Z" fill="currentColor" />
    </svg>
  );
}

function PauseIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path d="M8 5.5h3.5v13H8zM12.5 5.5H16v13h-3.5z" fill="currentColor" />
    </svg>
  );
}

function VolumeOnIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path
        d="M4.5 9.5h4L13 5.75v12.5L8.5 14.5h-4Z"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinejoin="round"
      />
      <path
        d="M16 9a4.5 4.5 0 0 1 0 6"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
      />
      <path
        d="M18.5 6.5a8 8 0 0 1 0 11"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
      />
    </svg>
  );
}

function VolumeOffIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path
        d="M4.5 9.5h4L13 5.75v12.5L8.5 14.5h-4Z"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinejoin="round"
      />
      <path
        d="m16.5 9 5 6m0-6-5 6"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
      />
    </svg>
  );
}

function SpeedIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path
        d="M5 16.5a7 7 0 1 1 14 0"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
      />
      <path
        d="m12 12 4-3"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
      />
      <circle cx="12" cy="12" r="1.4" fill="currentColor" />
    </svg>
  );
}

function FullscreenIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path
        d="M8 4.75H4.75V8M16 4.75h3.25V8M8 19.25H4.75V16M16 19.25h3.25V16"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function ChevronUpIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path d="m6 14 6-6 6 6" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

function ChevronDownIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path d="m6 10 6 6 6-6" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

export function VideoStage({ video, onChange, onActivate }: VideoStageProps) {
  const containerRef = useRef<HTMLElement | null>(null);
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const feedStackRef = useRef<HTMLElement | null>(null);
  const fullscreenScrollTopRef = useRef(0);
  const wasFullscreenRef = useRef(false);
  const playReportedRef = useRef(false);

  const [playbackUrl, setPlaybackUrl] = useState<string | null>(null);
  const [playbackLoading, setPlaybackLoading] = useState(false);
  const [isVisible, setIsVisible] = useState(false);
  const [busyAction, setBusyAction] = useState<"" | "like" | "favorite" | "comment">("");
  const [commentsOpen, setCommentsOpen] = useState(false);
  const [comments, setComments] = useState<CommentData[]>([]);
  const [commentInput, setCommentInput] = useState("");
  const [commentError, setCommentError] = useState<string | null>(null);
  const [commentLoading, setCommentLoading] = useState(false);
  const [isMuted, setIsMuted] = useState(true);
  const [isPaused, setIsPaused] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [expanded, setExpanded] = useState(false);
  const [playbackRate, setPlaybackRate] = useState(1);
  const [isFullscreen, setIsFullscreen] = useState(false);

  const visibleComments = useMemo(() => comments.filter((item) => !item.is_deleted), [comments]);
  const shouldTruncateCaption = video.caption.length > 34;
  const displayCaption = shouldTruncateCaption && !expanded ? `${video.caption.slice(0, 34)}...` : video.caption;
  const progress = duration > 0 ? (currentTime / duration) * 100 : 0;

  useEffect(() => {
    setPlaybackUrl(null);
    setPlaybackLoading(false);
    setCommentsOpen(false);
    setCommentError(null);
    setCommentInput("");
    setComments([]);
    setCommentLoading(false);
    setIsPaused(false);
    setCurrentTime(0);
    setDuration(0);
    setExpanded(false);
    setPlaybackRate(1);
    playReportedRef.current = false;
  }, [video.id]);

  useEffect(() => {
    const element = containerRef.current;
    if (!element) {
      return;
    }
    feedStackRef.current = element.closest(".feed-stack");

    const observer = new IntersectionObserver(
      (entries) => {
        const entry = entries[0];
        if (isFullscreen) {
          setIsVisible(true);
          return;
        }
        setIsVisible(entry?.isIntersecting === true && entry.intersectionRatio >= 0.7);
      },
      { threshold: [0.4, 0.7, 0.9] }
    );

    observer.observe(element);
    return () => observer.disconnect();
  }, [isFullscreen]);

  useEffect(() => {
    if (isVisible) {
      onActivate?.(video.id);
    }
  }, [isVisible, onActivate, video.id]);

  useEffect(() => {
    const feedStack = feedStackRef.current;
    if (!feedStack) {
      return;
    }

    if (isFullscreen) {
      feedStack.classList.add("feed-stack-fullscreen-lock");
      document.body.classList.add("app-body-fullscreen-lock");
      setIsVisible(true);
      wasFullscreenRef.current = true;
      return () => {
        feedStack.classList.remove("feed-stack-fullscreen-lock");
        document.body.classList.remove("app-body-fullscreen-lock");
      };
    }

    feedStack.classList.remove("feed-stack-fullscreen-lock");
    document.body.classList.remove("app-body-fullscreen-lock");
    if (wasFullscreenRef.current) {
      feedStack.scrollTop = fullscreenScrollTopRef.current;
      wasFullscreenRef.current = false;
    }
  }, [isFullscreen]);

  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setIsFullscreen(false);
      }
    }

    window.addEventListener("keydown", handleKeyDown);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
      document.body.classList.remove("app-body-fullscreen-lock");
      feedStackRef.current?.classList.remove("feed-stack-fullscreen-lock");
    };
  }, []);

  useEffect(() => {
    if (!isVisible && !isFullscreen) {
      videoRef.current?.pause();
      return;
    }

    if (playbackUrl) {
      if (!isPaused) {
        void videoRef.current?.play().catch((error) => {
          console.error("[video-stage] autoplay failed", {
            videoId: video.id,
            playbackUrl,
            error
          });
        });
      }
      return;
    }

    let cancelled = false;
    setPlaybackLoading(true);
    setCommentError(null);

    getVideoResources(video.id)
      .then((result) => {
        if (cancelled) {
          return;
        }

        const signedResourceUrl = result.items[0]?.file_url || "";
        if (!signedResourceUrl) {
          throw new Error(`当前视频没有可播放资源，videoId=${video.id}`);
        }

        setPlaybackUrl(signedResourceUrl);
      })
      .catch((error: Error) => {
        if (!cancelled) {
          console.error("[video-stage] load resource failed", {
            videoId: video.id,
            error
          });
          setCommentError(error.message);
        }
      })
      .finally(() => {
        if (!cancelled) {
          setPlaybackLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [isVisible, isPaused, playbackUrl, video.id, isFullscreen]);

  useEffect(() => {
    const node = videoRef.current;
    if (!node) {
      return;
    }
    node.muted = isMuted;
  }, [isMuted]);

  useEffect(() => {
    const node = videoRef.current;
    if (!node) {
      return;
    }
    node.playbackRate = playbackRate;
  }, [playbackRate]);

  useEffect(() => {
    if (playReportedRef.current) {
      return;
    }
    if (!isVisible && !isFullscreen) {
      return;
    }
    if (currentTime < 2) {
      return;
    }

    playReportedRef.current = true;
    reportVideoPlay(video.id).catch((error: Error) => {
      playReportedRef.current = false;
      console.error("[video-stage] report play failed", {
        videoId: video.id,
        currentTime,
        error
      });
    });
  }, [currentTime, isFullscreen, isVisible, video.id]);

  useEffect(() => {
    if (!commentsOpen) {
      return;
    }

    let alive = true;
    setCommentLoading(true);
    setCommentError(null);

    listComments({ video_id: video.id, root_id: "0", limit: 20 })
      .then((result) => {
        if (alive) {
          setComments(result.items);
        }
      })
      .catch((error: Error) => {
        if (alive) {
          setCommentError(error.message);
        }
      })
      .finally(() => {
        if (alive) {
          setCommentLoading(false);
        }
      });

    return () => {
      alive = false;
    };
  }, [commentsOpen, video.id]);

  async function handleLike() {
    if (busyAction) {
      return;
    }

    const nextLiked = !video.isLiked;
    const nextCount = Math.max(0, video.likeCount + (nextLiked ? 1 : -1));
    const optimistic = updateVideoCardInteract(video, {
      isLiked: nextLiked,
      likeCount: nextCount
    });

    onChange?.(optimistic);
    setBusyAction("like");
    try {
      await likeVideo(video.id, nextLiked ? 1 : 2);
    } catch (error) {
      onChange?.(video);
      setCommentError(error instanceof Error ? error.message : "点赞操作失败");
    } finally {
      setBusyAction("");
    }
  }

  async function handleFavorite() {
    if (busyAction) {
      return;
    }

    const nextFavorited = !video.isFavorited;
    const nextCount = Math.max(0, video.favoriteCount + (nextFavorited ? 1 : -1));
    const optimistic = updateVideoCardInteract(video, {
      isFavorited: nextFavorited,
      favoriteCount: nextCount
    });

    onChange?.(optimistic);
    setBusyAction("favorite");
    try {
      await favoriteVideo(video.id, nextFavorited ? 1 : 2);
    } catch (error) {
      onChange?.(video);
      setCommentError(error instanceof Error ? error.message : "收藏操作失败");
    } finally {
      setBusyAction("");
    }
  }

  async function handlePublishComment() {
    const content = commentInput.trim();
    if (!content || busyAction) {
      return;
    }

    setBusyAction("comment");
    setCommentError(null);
    try {
      const created = await publishComment({
        video_id: video.id,
        content,
        parent_id: "0",
        root_id: "0",
        to_user_id: "0"
      });
      const nextComments = [created, ...comments];
      setComments(nextComments);
      setCommentInput("");
      onChange?.(
        updateVideoCardInteract(video, {
          commentCount: countVisibleComments(nextComments)
        })
      );
    } catch (error) {
      setCommentError(error instanceof Error ? error.message : "发表评论失败");
    } finally {
      setBusyAction("");
    }
  }

  async function handleLikeComment(commentId: string, liked: boolean) {
    try {
      await likeComment(commentId, liked ? 2 : 1);
      setComments((current) =>
        current.map((item) =>
          item.id === commentId
            ? {
                ...item,
                like_count: Math.max(0, item.like_count + (liked ? -1 : 1))
              }
            : item
        )
      );
    } catch (error) {
      setCommentError(error instanceof Error ? error.message : "评论点赞失败");
    }
  }

  function togglePause() {
    const node = videoRef.current;
    if (!node) {
      return;
    }

    if (node.paused) {
      void node.play().catch(() => undefined);
      setIsPaused(false);
      return;
    }

    node.pause();
    setIsPaused(true);
  }

  function handleStageClick(event: MouseEvent<HTMLDivElement>) {
    const target = event.target as HTMLElement | null;
    if (
      target?.closest(
        "button, textarea, input, select, a, .comment-drawer, .video-control-dock, .caption-expand-button, .video-stage-switchers, .video-stage-actions"
      )
    ) {
      return;
    }
    togglePause();
  }

  function toggleMute() {
    setIsMuted((current) => !current);
  }

  function handleSeek(nextValue: number) {
    const node = videoRef.current;
    if (!node || !duration) {
      return;
    }
    const nextTime = (nextValue / 100) * duration;
    node.currentTime = nextTime;
    setCurrentTime(nextTime);
  }

  function cyclePlaybackRate() {
    setPlaybackRate((current) => {
      const currentIndex = PLAYBACK_RATES.indexOf(current);
      return PLAYBACK_RATES[(currentIndex + 1) % PLAYBACK_RATES.length];
    });
  }

  function toggleFullscreen() {
    const feedStack = feedStackRef.current;
    if (!isFullscreen && feedStack) {
      fullscreenScrollTopRef.current = feedStack.scrollTop;
    }
    setIsFullscreen((current) => !current);
  }

  function scrollToSibling(direction: "prev" | "next") {
    const current = containerRef.current;
    if (!current) {
      return;
    }
    const target = direction === "next" ? current.nextElementSibling : current.previousElementSibling;
    if (target instanceof HTMLElement) {
      target.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  }

  return (
    <article ref={containerRef} data-video-id={video.id} className={`video-stage-card ${isFullscreen ? "video-stage-card-fullscreen" : ""}`}>
      <div className={`video-stage-frame ${commentsOpen ? "video-stage-frame-comments-open" : ""}`}>
        <div className="video-stage-shell">
          <div className="video-stage-player-wrap">
            <div className="video-stage-main">
              <div className="video-stage" onClick={handleStageClick}>
                {playbackUrl ? (
                  <video
                    ref={videoRef}
                    className="video-player"
                    src={playbackUrl}
                    autoPlay
                    muted={isMuted}
                    loop
                    playsInline
                    onTimeUpdate={(event) => setCurrentTime(event.currentTarget.currentTime)}
                    onLoadedMetadata={(event) => setDuration(event.currentTarget.duration)}
                    onError={(event) => {
                      const media = event.currentTarget;
                      console.error("[video-stage] video element error", {
                        videoId: video.id,
                        playbackUrl,
                        currentSrc: media.currentSrc,
                        networkState: media.networkState,
                        readyState: media.readyState,
                        mediaError: media.error
                      });
                      setCommentError(`视频加载失败，videoId=${video.id}`);
                    }}
                  />
                ) : video.coverUrl ? (
                  <img
                    className="video-cover-image"
                    src={video.coverUrl}
                    alt={video.caption}
                    loading="lazy"
                    onError={(event) => {
                      console.error("[video-stage] cover load failed", {
                        videoId: video.id,
                        coverUrl: event.currentTarget.currentSrc || video.coverUrl
                      });
                    }}
                  />
                ) : null}

                <div className="video-stage-gradient" />

                <div className="video-stage-actions">
                  <button
                    className={`action-bubble ${video.isLiked ? "action-bubble-active" : ""}`}
                    type="button"
                    onClick={() => void handleLike()}
                    disabled={busyAction !== "" && busyAction !== "like"}
                    aria-label={video.isLiked ? "取消点赞" : "点赞"}
                    title={video.isLiked ? "取消点赞" : "点赞"}
                  >
                    <span className="action-bubble-icon">
                      <HeartIcon filled={video.isLiked} />
                    </span>
                    <strong>{formatCount(video.likeCount)}</strong>
                  </button>
                  <button
                    className={`action-bubble ${commentsOpen ? "action-bubble-active" : ""}`}
                    type="button"
                    onClick={() => setCommentsOpen((current) => !current)}
                    aria-label={commentsOpen ? "收起评论" : "打开评论"}
                    title={commentsOpen ? "收起评论" : "打开评论"}
                  >
                    <span className="action-bubble-icon">
                      <CommentIcon />
                    </span>
                    <strong>{formatCount(video.commentCount)}</strong>
                  </button>
                  <button
                    className={`action-bubble ${video.isFavorited ? "action-bubble-active" : ""}`}
                    type="button"
                    onClick={() => void handleFavorite()}
                    disabled={busyAction !== "" && busyAction !== "favorite"}
                    aria-label={video.isFavorited ? "取消收藏" : "收藏"}
                    title={video.isFavorited ? "取消收藏" : "收藏"}
                  >
                    <span className="action-bubble-icon">
                      <BookmarkIcon filled={video.isFavorited} />
                    </span>
                    <strong>{formatCount(video.favoriteCount)}</strong>
                  </button>
                  <button className="action-bubble" type="button" aria-label="转发" title="转发">
                    <span className="action-bubble-icon">
                      <ShareIcon />
                    </span>
                    <strong>{formatCount(video.shareCount)}</strong>
                  </button>
                </div>

                <div className="video-stage-switchers">
                  <button className="video-switch-button" type="button" onClick={() => scrollToSibling("prev")} aria-label="上一条视频" title="上一条视频">
                    <ChevronUpIcon />
                  </button>
                  <button className="video-switch-button" type="button" onClick={() => scrollToSibling("next")} aria-label="下一条视频" title="下一条视频">
                    <ChevronDownIcon />
                  </button>
                </div>

                <div className="video-stage-overlay video-stage-overlay-bottom">
                  <div className="video-overlay-copy">
                    {video.authorId ? (
                      <Link
                        className="video-author-line video-author-link"
                        to={`/users/${video.authorId}`}
                        onClick={(event) => {
                          event.stopPropagation();
                        }}
                      >
                        {video.authorHandle}
                      </Link>
                    ) : (
                      <div className="video-author-line">{video.authorHandle}</div>
                    )}
                    <div className="video-caption">
                      {displayCaption}
                      {shouldTruncateCaption ? (
                        <button className="caption-expand-button" type="button" onClick={() => setExpanded((current) => !current)}>
                          {expanded ? "收起" : "展开"}
                        </button>
                      ) : null}
                    </div>
                    <div className="video-published-at">{video.publishedAt}</div>
                  </div>
                </div>

                {isPaused ? (
                  <div className="video-pause-indicator" aria-label="已暂停">
                    <PlayIcon />
                  </div>
                ) : null}
                {playbackLoading ? <div className="video-loading-badge">加载中...</div> : null}
              </div>

              <div className="video-control-dock">
                <input
                  className="video-progress"
                  type="range"
                  min="0"
                  max="100"
                  step="0.1"
                  value={progress}
                  onChange={(event) => handleSeek(Number(event.target.value))}
                />
                <div className="video-control-bar">
                  <div className="video-control-left">
                    <button
                      className="video-control-button video-control-button-primary"
                      type="button"
                      onClick={togglePause}
                      aria-label={isPaused ? "播放" : "暂停"}
                      title={isPaused ? "播放" : "暂停"}
                    >
                      {isPaused ? <PlayIcon /> : <PauseIcon />}
                    </button>
                    <div className="video-time">
                      {formatClock(currentTime)} / {formatClock(duration)}
                    </div>
                  </div>
                  <div className="video-control-right">
                    <button className="video-control-button video-control-button-icon" type="button" onClick={toggleMute} aria-label={isMuted ? "开启声音" : "静音"} title={isMuted ? "开启声音" : "静音"}>
                      {isMuted ? <VolumeOffIcon /> : <VolumeOnIcon />}
                    </button>
                    <button className="video-control-button video-control-speed" type="button" onClick={cyclePlaybackRate} aria-label={`切换倍速，当前 ${formatPlaybackRate(playbackRate)}`} title={`当前倍速 ${formatPlaybackRate(playbackRate)}`}>
                      <SpeedIcon />
                      <span>{formatPlaybackRate(playbackRate)}</span>
                    </button>
                    <button className="video-control-button video-control-button-icon" type="button" onClick={toggleFullscreen} aria-label="全屏" title="全屏">
                      <FullscreenIcon />
                    </button>
                  </div>
                </div>
              </div>
            </div>

            <aside className={`comment-drawer ${commentsOpen ? "comment-drawer-open" : ""}`}>
              <div className="comment-drawer-header">
                <div>
                  <h4>评论</h4>
                  <span>{formatCount(countVisibleComments(comments))} 条</span>
                </div>
                <button className="comment-drawer-close" type="button" onClick={() => setCommentsOpen(false)} aria-label="关闭评论">
                  ×
                </button>
              </div>

              {commentError ? <div className="form-error">{commentError}</div> : null}
              {video.allowComment === false ? <div className="comment-empty">作者已关闭评论。</div> : null}

              <div className="comment-drawer-body">
                {commentLoading ? <div className="comment-empty">正在加载评论...</div> : null}
                {!commentLoading && visibleComments.length === 0 ? <div className="comment-empty">还没有评论，来抢沙发吧。</div> : null}
                <div className="comment-list">
                  {visibleComments.map((comment) => (
                    <article key={comment.id} className="comment-card">
                      <div className="comment-card-main">
                        <div className="comment-card-meta">
                          <span>用户 {comment.user_id}</span>
                          <span>{formatRelativeTime(comment.created_at)}</span>
                        </div>
                        <div className="comment-card-content">{comment.content}</div>
                      </div>
                      <button className="comment-like-button" type="button" onClick={() => void handleLikeComment(comment.id, false)}>
                        <HeartIcon />
                        <span>{formatCount(comment.like_count)}</span>
                      </button>
                    </article>
                  ))}
                </div>
              </div>

              {video.allowComment !== false ? (
                <div className="comment-drawer-composer">
                  <textarea value={commentInput} onChange={(event) => setCommentInput(event.target.value)} placeholder="写下你的评论" rows={3} />
                  <div className="comment-drawer-actions">
                    <button
                      className="primary-button"
                      type="button"
                      onClick={() => void handlePublishComment()}
                      disabled={!commentInput.trim() || busyAction === "comment"}
                    >
                      发送
                    </button>
                  </div>
                </div>
              ) : null}
            </aside>
          </div>
        </div>
      </div>
    </article>
  );
}
