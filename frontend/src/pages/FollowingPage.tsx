import { useEffect, useState } from "react";

import { listFollowingFeed } from "../api/client";
import { VideoStage } from "../components/VideoStage";
import { useAuth } from "../stores/auth";
import type { VideoCardModel } from "../types/models";
import { mapFeedVideoToCard } from "../utils/feed";

export function FollowingPage() {
  const { isAuthenticated } = useAuth();
  const [items, setItems] = useState<VideoCardModel[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!isAuthenticated) {
      setItems([]);
      return;
    }

    let alive = true;
    setLoading(true);
    setError(null);

    listFollowingFeed({ limit: 10 })
      .then((result) => {
        if (alive) {
          setItems(result.items.map(mapFeedVideoToCard));
        }
      })
      .catch((err: Error) => {
        if (alive) {
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

  function handleVideoChange(nextVideo: VideoCardModel) {
    setItems((current) => current.map((item) => (item.id === nextVideo.id ? nextVideo : item)));
  }

  return (
    <section className="page-block">
      {!isAuthenticated ? <div className="panel panel-roomy">请先登录后查看关注流。</div> : null}
      {loading ? <div className="panel panel-roomy">正在加载关注流...</div> : null}
      {error ? <div className="panel panel-roomy">{error}</div> : null}
      <div className="feed-stack">
        {items.map((video) => (
          <VideoStage key={video.id} video={video} onChange={handleVideoChange} />
        ))}
        {!loading && isAuthenticated && items.length === 0 ? <div className="panel panel-roomy">当前关注流为空，先去关注一些作者吧。</div> : null}
      </div>
    </section>
  );
}
