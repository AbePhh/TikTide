import { useEffect, useRef, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";

import { searchAll } from "../api/client";
import type { SearchAllResponseData } from "../types/api";
import { AuthFormModal } from "./AuthFormModal";
import { useAuth } from "../stores/auth";

type AuthMode = "login" | "register";

const SEARCH_SUGGEST_LIMIT = 3;

export function Topbar() {
  const { isAuthenticated } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [authOpen, setAuthOpen] = useState(false);
  const [authMode, setAuthMode] = useState<AuthMode>("login");
  const [keyword, setKeyword] = useState(() => new URLSearchParams(location.search).get("q") ?? "");
  const [suggestions, setSuggestions] = useState<SearchAllResponseData | null>(null);
  const [loading, setLoading] = useState(false);
  const [searchError, setSearchError] = useState<string | null>(null);
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const routeKeyword = new URLSearchParams(location.search).get("q") ?? "";
    if (location.pathname === "/search") {
      setKeyword(routeKeyword);
    }
  }, [location.pathname, location.search]);

  useEffect(() => {
    if (!isAuthenticated) {
      setSuggestions(null);
      setSearchError(null);
      setLoading(false);
      setOpen(false);
      return;
    }

    const trimmed = keyword.trim();
    if (trimmed.length < 2) {
      setSuggestions(null);
      setSearchError(null);
      setLoading(false);
      return;
    }

    let alive = true;
    const timer = window.setTimeout(() => {
      setLoading(true);
      setSearchError(null);
      searchAll({ q: trimmed, limit: SEARCH_SUGGEST_LIMIT })
        .then((result) => {
          if (!alive) {
            return;
          }
          setSuggestions(result);
          setOpen(true);
        })
        .catch((error: Error) => {
          if (!alive) {
            return;
          }
          console.error("[topbar-search] suggest failed", { keyword: trimmed, error });
          setSuggestions(null);
          setSearchError(error.message);
          setOpen(true);
        })
        .finally(() => {
          if (alive) {
            setLoading(false);
          }
        });
    }, 320);

    return () => {
      alive = false;
      window.clearTimeout(timer);
    };
  }, [isAuthenticated, keyword]);

  useEffect(() => {
    function handleClick(event: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => {
      document.removeEventListener("mousedown", handleClick);
    };
  }, []);

  function openAuth(mode: AuthMode) {
    setAuthMode(mode);
    setAuthOpen(true);
  }

  function closeAuth() {
    setAuthOpen(false);
  }

  function submitSearch(rawKeyword?: string) {
    const nextKeyword = (rawKeyword ?? keyword).trim();
    if (nextKeyword.length < 1) {
      return;
    }
    setOpen(false);
    navigate(`/search?q=${encodeURIComponent(nextKeyword)}`);
  }

  return (
    <>
      <header className="topbar">
        <div className="topbar-main">
          <div className="topbar-search-wrap" ref={containerRef}>
            <form
              className="topbar-search"
              onSubmit={(event) => {
                event.preventDefault();
                submitSearch();
              }}
            >
              <span className="topbar-search-icon">⌕</span>
              <input
                className="topbar-search-input"
                placeholder="搜索作者、话题、视频"
                aria-label="搜索"
                value={keyword}
                onChange={(event) => setKeyword(event.currentTarget.value)}
                onFocus={() => {
                  if (suggestions || searchError) {
                    setOpen(true);
                  }
                }}
              />
            </form>

            {isAuthenticated && open ? (
              <div className="topbar-search-popover">
                {loading ? <div className="topbar-search-empty">搜索中...</div> : null}
                {!loading && searchError ? <div className="topbar-search-empty">{searchError}</div> : null}
                {!loading && !searchError && suggestions ? (
                  <>
                    <div className="topbar-search-header">
                      <span>搜索建议</span>
                      <button className="topbar-search-link" type="button" onClick={() => submitSearch()}>
                        查看全部
                      </button>
                    </div>

                    {suggestions.users.length > 0 ? (
                      <div className="topbar-search-group">
                        <div className="topbar-search-group-title">作者</div>
                        {suggestions.users.map((item) => (
                          <Link
                            key={`user-${item.id}`}
                            to={`/users/${item.id}`}
                            className="topbar-search-item"
                            onClick={() => setOpen(false)}
                          >
                            <div className="topbar-search-avatar">{(item.nickname || item.username || "U").slice(0, 1).toUpperCase()}</div>
                            <div className="topbar-search-item-main">
                              <div className="topbar-search-item-title">{item.nickname || item.username}</div>
                              <div className="topbar-search-item-subtitle">@{item.username}</div>
                            </div>
                          </Link>
                        ))}
                      </div>
                    ) : null}

                    {suggestions.hashtags.length > 0 ? (
                      <div className="topbar-search-group">
                        <div className="topbar-search-group-title">话题</div>
                        {suggestions.hashtags.map((item) => (
                          <Link
                            key={`hashtag-${item.id}`}
                            to={`/discover/topics/${item.id}`}
                            className="topbar-search-item"
                            onClick={() => setOpen(false)}
                          >
                            <div className="topbar-search-tag">#</div>
                            <div className="topbar-search-item-main">
                              <div className="topbar-search-item-title">#{item.name}</div>
                              <div className="topbar-search-item-subtitle">{item.use_count} 次使用</div>
                            </div>
                          </Link>
                        ))}
                      </div>
                    ) : null}

                    {suggestions.videos.length > 0 ? (
                      <div className="topbar-search-group">
                        <div className="topbar-search-group-title">视频</div>
                        {suggestions.videos.map((item) => (
                          <button
                            key={`video-${item.video_id}`}
                            className="topbar-search-item topbar-search-item-button"
                            type="button"
                            onClick={() => submitSearch(item.title)}
                          >
                            <div className="topbar-search-cover">
                              {item.cover_url ? <img src={item.cover_url} alt={item.title || "视频封面"} /> : <span>▶</span>}
                            </div>
                            <div className="topbar-search-item-main">
                              <div className="topbar-search-item-title">{item.title || "未命名视频"}</div>
                              <div className="topbar-search-item-subtitle">@{item.author?.username || `user${item.user_id}`}</div>
                            </div>
                          </button>
                        ))}
                      </div>
                    ) : null}

                    {!suggestions.users.length && !suggestions.hashtags.length && !suggestions.videos.length ? (
                      <div className="topbar-search-empty">没有找到相关内容</div>
                    ) : null}
                  </>
                ) : null}
              </div>
            ) : null}
          </div>
        </div>

        {!isAuthenticated ? (
          <div className="topbar-actions">
            <div className="topbar-auth-buttons">
              <button className="ghost-button" type="button" onClick={() => openAuth("register")}>
                注册
              </button>
              <button className="primary-button" type="button" onClick={() => openAuth("login")}>
                登录
              </button>
            </div>
          </div>
        ) : null}
      </header>

      {authOpen ? <AuthFormModal defaultMode={authMode} onClose={closeAuth} /> : null}
    </>
  );
}
