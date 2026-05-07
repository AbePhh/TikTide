import { useEffect, useRef, useState } from "react";

import { listRecommendFeed } from "../api/client";
import { VideoStage } from "../components/VideoStage";
import { useAuth } from "../stores/auth";
import type { FeedVideoData } from "../types/api";
import type { VideoCardModel } from "../types/models";
import { mapFeedVideoToCard } from "../utils/feed";

const INITIAL_BATCH_SIZE = 4;
const NEXT_BATCH_SIZE = 4;
const PREFETCH_THRESHOLD = 2;
const MAX_RENDERED_ITEMS = 20;

type FeedCursorState = {
  nextCursor: string | null;
  hasMore: boolean;
};

function mergeUniqueVideos(current: VideoCardModel[], incoming: VideoCardModel[]) {
  if (incoming.length === 0) {
    return current;
  }

  const seen = new Set(current.map((item) => item.id));
  const merged = [...current];
  for (const item of incoming) {
    if (seen.has(item.id)) {
      continue;
    }
    seen.add(item.id);
    merged.push(item);
  }
  return merged;
}

function mapVideos(items: FeedVideoData[]) {
  return items.map(mapFeedVideoToCard);
}

export function FeedPage() {
  const { isAuthenticated } = useAuth();
  const [items, setItems] = useState<VideoCardModel[]>([]);
  const [loading, setLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const cursorRef = useRef<FeedCursorState>({ nextCursor: null, hasMore: true });
  const requestTokenRef = useRef(0);
  const activeVideoIDRef = useRef<string | null>(null);
  const isLoadingMoreRef = useRef(false);
  const itemsRef = useRef<VideoCardModel[]>([]);

  async function fetchBatch(limit: number, cursor?: string | null) {
    const result = await listRecommendFeed({
      limit,
      cursor: cursor || undefined
    });
    return {
      items: mapVideos(result.items),
      nextCursor: result.next_cursor || null,
      hasMore: Boolean(result.next_cursor) || result.items.length === limit
    };
  }

  useEffect(() => {
    if (!isAuthenticated) {
      setItems([]);
      itemsRef.current = [];
      setError(null);
      setLoading(false);
      setLoadingMore(false);
      cursorRef.current = { nextCursor: null, hasMore: true };
      activeVideoIDRef.current = null;
      requestTokenRef.current += 1;
      isLoadingMoreRef.current = false;
      return;
    }

    const requestToken = requestTokenRef.current + 1;
    requestTokenRef.current = requestToken;

    let alive = true;
    setLoading(true);
    setLoadingMore(false);
    setError(null);
    setItems([]);
    itemsRef.current = [];
    cursorRef.current = { nextCursor: null, hasMore: true };
    activeVideoIDRef.current = null;
    isLoadingMoreRef.current = false;

    fetchBatch(INITIAL_BATCH_SIZE, null)
      .then((result) => {
        if (!alive || requestTokenRef.current !== requestToken) {
          return;
        }
        itemsRef.current = result.items;
        setItems(result.items);
        cursorRef.current = {
          nextCursor: result.nextCursor,
          hasMore: result.hasMore
        };
      })
      .catch((err: Error) => {
        if (!alive || requestTokenRef.current !== requestToken) {
          return;
        }
        setError(err.message);
      })
      .finally(() => {
        if (!alive || requestTokenRef.current !== requestToken) {
          return;
        }
        setLoading(false);
      });

    return () => {
      alive = false;
    };
  }, [isAuthenticated]);

  function handleVideoChange(nextVideo: VideoCardModel) {
    setItems((current) => {
      const nextItems = current.map((item) => (item.id === nextVideo.id ? nextVideo : item));
      itemsRef.current = nextItems;
      return nextItems;
    });
  }

  async function loadMoreIfNeeded(activeVideoID: string) {
    const currentItems = itemsRef.current;
    const activeIndex = currentItems.findIndex((item) => item.id === activeVideoID);
    if (activeIndex < 0) {
      return;
    }

    const remaining = currentItems.length - activeIndex - 1;
    if (remaining >= PREFETCH_THRESHOLD) {
      return;
    }

    if (isLoadingMoreRef.current || !cursorRef.current.hasMore) {
      return;
    }

    isLoadingMoreRef.current = true;
    setLoadingMore(true);
    setError(null);

    try {
      const result = await fetchBatch(NEXT_BATCH_SIZE, cursorRef.current.nextCursor);
      cursorRef.current = {
        nextCursor: result.nextCursor,
        hasMore: result.hasMore
      };

      const merged = mergeUniqueVideos(itemsRef.current, result.items);
      const limited = merged.slice(0, MAX_RENDERED_ITEMS);
      itemsRef.current = limited;
      setItems(limited);
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("推荐流加载失败");
      }
    } finally {
      isLoadingMoreRef.current = false;
      setLoadingMore(false);
    }
  }

  function handleVideoActivate(videoID: string) {
    activeVideoIDRef.current = videoID;
    void loadMoreIfNeeded(videoID);
  }

  return (
    <section className="page-block">
      {!isAuthenticated ? <div className="panel panel-roomy">请先登录后查看推荐流。</div> : null}
      {loading ? <div className="panel panel-roomy">正在为你加载推荐内容...</div> : null}
      {error ? <div className="panel panel-roomy">{error}</div> : null}
      <div className="feed-stack">
        {items.map((video) => (
          <VideoStage key={video.id} video={video} onChange={handleVideoChange} onActivate={handleVideoActivate} />
        ))}
        {loadingMore && items.length > 0 ? <div className="panel panel-roomy">正在预加载下一批视频...</div> : null}
        {!loading && !loadingMore && isAuthenticated && items.length === 0 ? <div className="panel panel-roomy">当前暂无可推荐内容，稍后再来看看。</div> : null}
      </div>
    </section>
  );
}
