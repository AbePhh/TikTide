import { useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { searchHashtags, searchUsers, searchVideos } from "../api/client";
import { PageSectionHeader } from "../components/PageSectionHeader";
import { VideoStage } from "../components/VideoStage";
import { useAuth } from "../stores/auth";
import type { SearchHashtagData, SearchUserData, SearchVideoData } from "../types/api";
import type { VideoCardModel } from "../types/models";
import { formatRelativeTime } from "../utils/format";
import { formatCount } from "../utils/format";

type SearchTab = "videos" | "users" | "hashtags";

const SEARCH_LIMIT = 18;

function mapSearchVideoToCard(video: SearchVideoData): VideoCardModel {
  const authorName = video.author?.nickname || video.author?.username || "未知作者";
  const authorHandle = `@${video.author?.username || `user${video.user_id}`}`;

  return {
    id: video.video_id,
    authorId: video.author?.id,
    authorName,
    authorHandle,
    authorAvatar: (authorName || "U").slice(0, 1).toUpperCase(),
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
    duration: "00:00",
    publishedAt: formatRelativeTime(new Date().toISOString()),
    isFollowed: video.interact?.is_followed ?? false,
    isLiked: video.interact?.is_liked ?? false,
    isFavorited: video.interact?.is_favorited ?? false,
    sourceUrl: video.source_url || undefined,
    allowComment: true
  };
}

export function SearchPage() {
  const { isAuthenticated } = useAuth();
  const [searchParams] = useSearchParams();
  const keyword = searchParams.get("q")?.trim() ?? "";
  const initialTab = (searchParams.get("tab") as SearchTab | null) ?? "videos";
  const [activeTab, setActiveTab] = useState<SearchTab>(["videos", "users", "hashtags"].includes(initialTab) ? initialTab : "videos");
  const [videoItems, setVideoItems] = useState<SearchVideoData[]>([]);
  const [userItems, setUserItems] = useState<SearchUserData[]>([]);
  const [hashtagItems, setHashtagItems] = useState<SearchHashtagData[]>([]);
  const [activeVideo, setActiveVideo] = useState<VideoCardModel | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!isAuthenticated || keyword.length < 1) {
      setVideoItems([]);
      setUserItems([]);
      setHashtagItems([]);
      setLoading(false);
      setError(null);
      return;
    }

    let alive = true;
    setLoading(true);
    setError(null);

    Promise.all([
      searchVideos({ q: keyword, limit: SEARCH_LIMIT }),
      searchUsers({ q: keyword, limit: SEARCH_LIMIT }),
      searchHashtags({ q: keyword, limit: SEARCH_LIMIT })
    ])
      .then(([videoResult, userResult, hashtagResult]) => {
        if (!alive) {
          return;
        }
        setVideoItems(videoResult.items);
        setUserItems(userResult.items);
        setHashtagItems(hashtagResult.items);
      })
      .catch((err: Error) => {
        if (alive) {
          console.error("[search-page] load failed", { keyword, err });
          setError(err.message);
        }
      })
      .finally(() => {
        if (alive) {
          setLoading(false);
        }
      });

    return () => {
      alive = false;
    };
  }, [isAuthenticated, keyword]);

  const summary = useMemo(
    () => [
      { key: "videos" as const, label: "视频", count: videoItems.length },
      { key: "users" as const, label: "作者", count: userItems.length },
      { key: "hashtags" as const, label: "话题", count: hashtagItems.length }
    ],
    [hashtagItems.length, userItems.length, videoItems.length]
  );

  return (
    <>
      <section className="page-block search-page">
        <PageSectionHeader
          title={keyword ? `搜索 “${keyword}”` : "搜索"}
          subtitle="结果已接入真实后端 Elasticsearch 搜索接口，当前展示视频、作者、话题三类结果。"
        />

        {!isAuthenticated ? <div className="panel panel-roomy">请先登录后再使用搜索功能。</div> : null}
        {isAuthenticated && !keyword ? <div className="panel panel-roomy">请输入关键词后进行搜索。</div> : null}
        {loading ? <div className="panel panel-roomy">正在搜索...</div> : null}
        {error ? <div className="panel panel-roomy">{error}</div> : null}

        {!loading && !error && isAuthenticated && keyword ? (
          <div className="search-shell">
            <section className="panel panel-roomy search-summary-panel">
              <div className="search-summary-tabs">
                {summary.map((item) => (
                  <button
                    key={item.key}
                    className={`search-summary-tab ${activeTab === item.key ? "search-summary-tab-active" : ""}`}
                    type="button"
                    onClick={() => setActiveTab(item.key)}
                  >
                    <span>{item.label}</span>
                    <strong>{item.count}</strong>
                  </button>
                ))}
              </div>
            </section>

            {activeTab === "videos" ? (
              <section className="panel panel-roomy">
                <div className="search-video-grid">
                  {videoItems.length === 0 ? <div className="discover-empty">没有找到相关视频。</div> : null}
                  {videoItems.map((video) => (
                    <article key={video.video_id} className="discover-video-card discover-video-card-rich search-video-card">
                      <button
                        className="discover-video-link search-video-button"
                        type="button"
                        onClick={() => setActiveVideo(mapSearchVideoToCard(video))}
                      >
                        <div className="discover-video-cover-shell">
                          {video.cover_url ? (
                            <>
                              <div className="discover-video-cover-blur" style={{ backgroundImage: `url(${video.cover_url})` }} aria-hidden="true" />
                              <img className="discover-video-cover-image" src={video.cover_url} alt={video.title || "视频封面"} />
                            </>
                          ) : (
                            <div className="discover-video-cover discover-video-cover-fallback" />
                          )}
                          <div className="discover-video-cover-overlay">
                            <span>{formatCount(video.play_count)} 播放</span>
                            <span>{formatCount(video.like_count)} 点赞</span>
                          </div>
                        </div>
                      </button>
                      <div className="discover-video-body">
                        <div className="discover-video-title">{video.title || "未命名视频"}</div>
                        <div className="discover-video-meta">
                          <Link to={`/users/${video.user_id}`}>@{video.author?.username || `user${video.user_id}`}</Link>
                          <span>{formatCount(video.comment_count)} 评论</span>
                        </div>
                        <div className="discover-video-submeta">
                          {formatCount(video.favorite_count)} 收藏 · {formatCount(video.comment_count)} 评论
                        </div>
                      </div>
                    </article>
                  ))}
                </div>
              </section>
            ) : null}

            {activeTab === "users" ? (
              <section className="panel panel-roomy">
                <div className="search-user-grid">
                  {userItems.length === 0 ? <div className="discover-empty">没有找到相关作者。</div> : null}
                  {userItems.map((user) => (
                    <Link key={user.id} to={`/users/${user.id}`} className="search-user-card">
                      <div className="search-user-avatar">{(user.nickname || user.username || "U").slice(0, 1).toUpperCase()}</div>
                      <div className="search-user-main">
                        <div className="search-user-name">{user.nickname || user.username}</div>
                        <div className="search-user-handle">@{user.username}</div>
                        <div className="search-user-signature">{user.signature || "这个作者还没有填写签名。"}</div>
                      </div>
                      <div className="search-user-metrics">
                        <span>{formatCount(user.follower_count)} 粉丝</span>
                        <span>{formatCount(user.work_count)} 作品</span>
                      </div>
                    </Link>
                  ))}
                </div>
              </section>
            ) : null}

            {activeTab === "hashtags" ? (
              <section className="panel panel-roomy">
                <div className="discover-topic-grid discover-topic-grid-large">
                  {hashtagItems.length === 0 ? <div className="discover-empty">没有找到相关话题。</div> : null}
                  {hashtagItems.map((topic, index) => (
                    <Link key={topic.id} to={`/discover/topics/${topic.id}`} className="discover-topic-card discover-topic-card-link">
                      <div className="discover-topic-rank">TOP {index + 1}</div>
                      <div className="discover-topic-title">#{topic.name}</div>
                      <div className="discover-topic-subtitle">搜索命中的相关话题</div>
                      <div className="discover-topic-heat">{formatCount(topic.use_count)} 次使用</div>
                    </Link>
                  ))}
                </div>
              </section>
            ) : null}
          </div>
        ) : null}
      </section>

      {activeVideo ? (
        <div className="auth-modal-backdrop search-video-modal-backdrop" onClick={() => setActiveVideo(null)}>
          <div className="search-video-modal" onClick={(event) => event.stopPropagation()}>
            <button className="auth-close-button search-video-modal-close" type="button" onClick={() => setActiveVideo(null)}>
              ×
            </button>
            <VideoStage video={activeVideo} onChange={setActiveVideo} />
          </div>
        </div>
      ) : null}
    </>
  );
}
