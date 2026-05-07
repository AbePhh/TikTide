import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";

import { listHotHashtags, listRecommendFeed } from "../api/client";
import { PageSectionHeader } from "../components/PageSectionHeader";
import { useAuth } from "../stores/auth";
import type { FeedVideoData, HashtagData } from "../types/api";
import { formatCount, formatRelativeTime } from "../utils/format";

const DISCOVER_RECOMMEND_LIMIT = 12;
const DISCOVER_HASHTAG_LIMIT = 10;

type CreatorSummary = {
  id: string;
  username: string;
  nickname: string;
  avatar: string;
  videoCount: number;
  totalLikes: number;
};

function buildCreatorSummary(videos: FeedVideoData[]): CreatorSummary[] {
  const creatorMap = new Map<string, CreatorSummary>();

  for (const video of videos) {
    const author = video.author;
    if (!author?.id) {
      continue;
    }

    const current = creatorMap.get(author.id) ?? {
      id: author.id,
      username: author.username,
      nickname: author.nickname || author.username,
      avatar: (author.nickname || author.username || "U").slice(0, 1).toUpperCase(),
      videoCount: 0,
      totalLikes: 0
    };

    current.videoCount += 1;
    current.totalLikes += video.like_count;
    creatorMap.set(author.id, current);
  }

  return Array.from(creatorMap.values())
    .sort((left, right) => {
      if (left.totalLikes === right.totalLikes) {
        return right.videoCount - left.videoCount;
      }
      return right.totalLikes - left.totalLikes;
    })
    .slice(0, 6);
}

export function DiscoverPage() {
  const { isAuthenticated } = useAuth();
  const [hotTopics, setHotTopics] = useState<HashtagData[]>([]);
  const [hotVideos, setHotVideos] = useState<FeedVideoData[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!isAuthenticated) {
      setHotTopics([]);
      setHotVideos([]);
      setLoading(false);
      setError(null);
      return;
    }

    let alive = true;
    setLoading(true);
    setError(null);

    Promise.all([listHotHashtags({ limit: DISCOVER_HASHTAG_LIMIT }), listRecommendFeed({ limit: DISCOVER_RECOMMEND_LIMIT })])
      .then(([topicResult, videoResult]) => {
        if (!alive) {
          return;
        }
        setHotTopics(topicResult.items);
        setHotVideos(videoResult.items);
      })
      .catch((err: Error) => {
        if (alive) {
          console.error("[discover] load failed", err);
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
  }, [isAuthenticated]);

  const hotCreators = useMemo(() => buildCreatorSummary(hotVideos), [hotVideos]);
  const rankingTopics = hotTopics.slice(0, 5);

  return (
    <section className="page-block">
      <PageSectionHeader title="发现" subtitle="围绕热搜话题、热门视频和创作者推荐构建的探索页。当前仅接入后端已存在的真实接口。 " />

      {!isAuthenticated ? <div className="panel panel-roomy">请先登录后查看发现页内容。</div> : null}
      {loading ? <div className="panel panel-roomy">正在加载发现内容...</div> : null}
      {error ? <div className="panel panel-roomy">{error}</div> : null}

      {!loading && !error && isAuthenticated ? (
        <div className="discover-shell">
          <div className="discover-main">
            <section className="panel panel-roomy">
              <div className="panel-title-row">
                <h3>热门话题</h3>
                <span>真实数据</span>
              </div>
              <div className="discover-topic-grid discover-topic-grid-large">
                {hotTopics.length === 0 ? <div className="discover-empty">当前暂无热门话题数据。</div> : null}
                {hotTopics.map((topic, index) => (
                  <Link key={topic.id} to={`/discover/topics/${topic.id}`} className="discover-topic-card discover-topic-card-link">
                    <div className="discover-topic-rank">TOP {index + 1}</div>
                    <div className="discover-topic-title">#{topic.name}</div>
                    <div className="discover-topic-subtitle">当前按话题使用次数排序</div>
                    <div className="discover-topic-heat">{formatCount(topic.use_count)} 次使用</div>
                  </Link>
                ))}
              </div>
            </section>

            <section className="panel panel-roomy">
              <div className="panel-title-row">
                <h3>热门视频</h3>
                <span>来自推荐流</span>
              </div>
              <div className="discover-video-grid discover-video-grid-rich">
                {hotVideos.length === 0 ? <div className="discover-empty">当前暂无可展示的视频。</div> : null}
                {hotVideos.map((video) => (
                  <article key={video.video_id} className="discover-video-card discover-video-card-rich">
                    <Link to="/" className="discover-video-link" title="当前点击后进入推荐流观看完整视频">
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
                          <span>{video.duration_ms > 0 ? `${Math.max(1, Math.round(video.duration_ms / 1000))}s` : "视频"}</span>
                        </div>
                      </div>
                    </Link>
                    <div className="discover-video-body">
                      <div className="discover-video-title">{video.title || "未命名视频"}</div>
                      <div className="discover-video-meta">
                        <span>@{video.author?.username || `user${video.user_id}`}</span>
                        <span>{formatCount(video.like_count)} 点赞</span>
                      </div>
                      <div className="discover-video-submeta">{formatRelativeTime(video.created_at)}</div>
                    </div>
                  </article>
                ))}
              </div>
            </section>

            <section className="panel panel-roomy">
              <div className="panel-title-row">
                <h3>热门创作者</h3>
                <span>由推荐流作者聚合</span>
              </div>
              <div className="discover-creator-grid">
                {hotCreators.length === 0 ? <div className="discover-empty">当前暂无创作者数据。</div> : null}
                {hotCreators.map((creator) => (
                  <Link key={creator.id} to={`/users/${creator.id}`} className="discover-creator-card">
                    <div className="discover-creator-avatar">{creator.avatar}</div>
                    <div className="discover-creator-name">{creator.nickname}</div>
                    <div className="discover-creator-handle">@{creator.username}</div>
                    <div className="discover-creator-meta">
                      <span>{creator.videoCount} 条作品</span>
                      <span>{formatCount(creator.totalLikes)} 总点赞</span>
                    </div>
                  </Link>
                ))}
              </div>
            </section>
          </div>

          <aside className="discover-side">
            <section className="panel panel-roomy">
              <div className="panel-title-row">
                <h3>热度排行</h3>
                <span>话题榜</span>
              </div>
              <div className="discover-ranking-list">
                {rankingTopics.length === 0 ? <div className="discover-empty">暂无排行数据。</div> : null}
                {rankingTopics.map((topic, index) => (
                  <Link key={topic.id} to={`/discover/topics/${topic.id}`} className="discover-ranking-item">
                    <div className="discover-ranking-index">{index + 1}</div>
                    <div className="discover-ranking-main">
                      <div className="discover-ranking-title">#{topic.name}</div>
                      <div className="discover-ranking-subtitle">{formatCount(topic.use_count)} 次使用</div>
                    </div>
                  </Link>
                ))}
              </div>
            </section>
          </aside>
        </div>
      ) : null}
    </section>
  );
}
