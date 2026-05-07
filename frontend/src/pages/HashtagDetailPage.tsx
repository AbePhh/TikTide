import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";

import { getHashtag, listHashtagVideos } from "../api/client";
import { PageSectionHeader } from "../components/PageSectionHeader";
import { useAuth } from "../stores/auth";
import type { FeedVideoData, HashtagData } from "../types/api";
import { formatCount, formatRelativeTime } from "../utils/format";

export function HashtagDetailPage() {
  const { hid } = useParams();
  const { isAuthenticated } = useAuth();
  const [hashtag, setHashtag] = useState<HashtagData | null>(null);
  const [videos, setVideos] = useState<FeedVideoData[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!isAuthenticated || !hid) {
      setHashtag(null);
      setVideos([]);
      return;
    }

    let alive = true;
    setLoading(true);
    setError(null);

    Promise.all([getHashtag(hid), listHashtagVideos(hid, { limit: 18 })])
      .then(([hashtagResult, videoResult]) => {
        if (!alive) {
          return;
        }
        setHashtag(hashtagResult);
        setVideos(videoResult.items);
      })
      .catch((err: Error) => {
        if (alive) {
          console.error("[hashtag-detail] load failed", { hid, err });
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
  }, [hid, isAuthenticated]);

  return (
    <section className="page-block">
      <PageSectionHeader
        title={hashtag ? `#${hashtag.name}` : "话题详情"}
        subtitle={hashtag ? `当前话题共使用 ${formatCount(hashtag.use_count)} 次` : "浏览指定话题下的公开视频"}
      />

      {!isAuthenticated ? <div className="panel panel-roomy">请先登录后查看话题内容。</div> : null}
      {loading ? <div className="panel panel-roomy">正在加载话题内容...</div> : null}
      {error ? <div className="panel panel-roomy">{error}</div> : null}

      {!loading && !error && isAuthenticated ? (
        <section className="panel panel-roomy">
          <div className="discover-video-grid discover-video-grid-rich">
            {videos.length === 0 ? <div className="discover-empty">该话题下暂时还没有公开视频。</div> : null}
            {videos.map((video) => (
              <article key={video.video_id} className="discover-video-card discover-video-card-rich">
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
                <div className="discover-video-body">
                  <div className="discover-video-title">{video.title || "未命名视频"}</div>
                  <div className="discover-video-meta">
                    <span>@{video.author?.username || `user${video.user_id}`}</span>
                    <span>{formatCount(video.comment_count)} 评论</span>
                  </div>
                  <div className="discover-video-submeta">{formatRelativeTime(video.created_at)}</div>
                </div>
              </article>
            ))}
          </div>
        </section>
      ) : null}
    </section>
  );
}
