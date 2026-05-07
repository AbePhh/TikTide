import { useEffect, useMemo, useState } from "react";

import { getProfile, listFollowerUsers, listFollowingUsers, listUserVideos } from "../api/client";
import { PageSectionHeader } from "../components/PageSectionHeader";
import { useAuth } from "../stores/auth";
import type { FeedVideoData, ProfileData, RelationUserData } from "../types/api";
import { buildAvatarFallback, formatCount } from "../utils/format";

let profileRequest: Promise<ProfileData> | null = null;

type RelationModalMode = "following" | "followers" | null;

function getProfileOnce() {
  if (!profileRequest) {
    profileRequest = getProfile().finally(() => {
      profileRequest = null;
    });
  }
  return profileRequest;
}

function getVideoStatusText(video: FeedVideoData) {
  if (video.transcode_status === 3) {
    return video.transcode_fail_reason || "转码失败";
  }
  if (video.transcode_status !== 2) {
    return "转码中";
  }
  if (video.audit_status === 2) {
    return video.audit_remark || "审核未通过";
  }
  if (video.audit_status !== 1) {
    return "审核中";
  }
  if (video.visibility === 0) {
    return "仅自己可见";
  }
  return "";
}

function isPlayableVideo(video: FeedVideoData) {
  return video.transcode_status === 2 && video.audit_status === 1 && Boolean(video.source_url);
}

export function ProfilePage() {
  const { isAuthenticated, updateUser, user } = useAuth();
  const [profile, setProfile] = useState<ProfileData | null>(user);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [relationModalMode, setRelationModalMode] = useState<RelationModalMode>(null);
  const [relationLoading, setRelationLoading] = useState(false);
  const [relationError, setRelationError] = useState<string | null>(null);
  const [relationUsers, setRelationUsers] = useState<RelationUserData[]>([]);

  const [videos, setVideos] = useState<FeedVideoData[]>([]);
  const [videoLoading, setVideoLoading] = useState(false);
  const [videoError, setVideoError] = useState<string | null>(null);

  const relationModalTitle = useMemo(() => {
    if (relationModalMode === "following") {
      return "关注列表";
    }
    if (relationModalMode === "followers") {
      return "粉丝列表";
    }
    return "";
  }, [relationModalMode]);

  useEffect(() => {
    if (!isAuthenticated) {
      setProfile(null);
      setVideos([]);
      return;
    }

    let alive = true;
    setLoading(true);
    setError(null);

    getProfileOnce()
      .then((result) => {
        if (!alive) {
          return;
        }
        setProfile(result);
        updateUser(result);
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
  }, [isAuthenticated, updateUser]);

  useEffect(() => {
    if (user) {
      setProfile(user);
    }
  }, [user]);

  useEffect(() => {
    if (!profile?.id || !isAuthenticated) {
      setVideos([]);
      return;
    }

    let alive = true;
    setVideoLoading(true);
    setVideoError(null);

    listUserVideos(profile.id, { limit: 24 })
      .then((result) => {
        if (alive) {
          setVideos(result.items);
        }
      })
      .catch((err: Error) => {
        if (alive) {
          setVideoError(err.message);
        }
      })
      .finally(() => {
        if (alive) {
          setVideoLoading(false);
        }
      });

    return () => {
      alive = false;
    };
  }, [isAuthenticated, profile?.id]);

  useEffect(() => {
    if (!relationModalMode || !profile?.id) {
      return;
    }

    let alive = true;
    setRelationLoading(true);
    setRelationError(null);
    setRelationUsers([]);

    const request =
      relationModalMode === "following" ? listFollowingUsers(profile.id, { limit: 50 }) : listFollowerUsers(profile.id, { limit: 50 });

    request
      .then((result) => {
        if (alive) {
          setRelationUsers(result.items);
        }
      })
      .catch((err: Error) => {
        if (alive) {
          setRelationError(err.message);
        }
      })
      .finally(() => {
        if (alive) {
          setRelationLoading(false);
        }
      });

    return () => {
      alive = false;
    };
  }, [profile?.id, relationModalMode]);

  return (
    <>
      <section className="page-block">
        <PageSectionHeader title="我的主页" subtitle="展示当前登录用户资料、关注关系和本人已发布作品。" />
        {!isAuthenticated ? <div className="panel panel-roomy">请先登录后查看个人主页。</div> : null}
        {loading ? <div className="panel panel-roomy">正在加载个人资料...</div> : null}
        {error ? <div className="panel panel-roomy">{error}</div> : null}

        <section className="profile-hero panel panel-roomy">
          <div className="profile-hero-avatar">{buildAvatarFallback(profile?.nickname ?? "T")}</div>
          <div className="profile-hero-main">
            <h2>{profile?.nickname ?? "未登录用户"}</h2>
            <div className="profile-handle">@{profile?.username ?? "guest"}</div>
            <p className="profile-signature">{profile?.signature || "这个用户还没有填写个性签名。"}</p>
            <div className="profile-stats">
              <span className="profile-stat-chip">
                <strong>{formatCount(profile?.work_count ?? 0)}</strong> 作品
              </span>
              <button className="profile-stat-chip profile-stat-button" type="button" onClick={() => setRelationModalMode("following")}>
                <strong>{formatCount(profile?.follow_count ?? 0)}</strong> 关注
              </button>
              <button className="profile-stat-chip profile-stat-button" type="button" onClick={() => setRelationModalMode("followers")}>
                <strong>{formatCount(profile?.follower_count ?? 0)}</strong> 粉丝
              </button>
              <span className="profile-stat-chip">
                <strong>{formatCount(profile?.total_liked_count ?? 0)}</strong> 获赞
              </span>
            </div>
          </div>
        </section>

        <section className="works-grid works-grid-profile">
          {videoLoading ? <div className="panel panel-roomy works-empty">正在加载作品...</div> : null}
          {videoError ? <div className="panel panel-roomy works-empty">{videoError}</div> : null}
          {!videoLoading && !videoError && videos.length === 0 ? <div className="panel panel-roomy works-empty">你还没有发布作品。</div> : null}

          {!videoLoading && !videoError
            ? videos.map((video) => {
                const statusText = getVideoStatusText(video);
                const playable = isPlayableVideo(video);
                const mediaPoster = video.cover_url || undefined;
                return (
                  <article key={video.video_id} className="work-card work-card-video">
                    <div className="work-cover work-cover-video">
                      {video.cover_url ? (
                        <div className="work-cover-background" style={{ backgroundImage: `url(${video.cover_url})` }} aria-hidden="true" />
                      ) : (
                        <div className="work-cover-background work-cover-background-fallback" aria-hidden="true" />
                      )}

                      {playable ? (
                        <video
                          className="work-video"
                          src={video.source_url}
                          poster={mediaPoster}
                          muted
                          loop
                          playsInline
                          preload="metadata"
                          onMouseEnter={(event) => {
                            event.currentTarget.play().catch(() => undefined);
                          }}
                          onMouseLeave={(event) => {
                            event.currentTarget.pause();
                            event.currentTarget.currentTime = 0;
                          }}
                        />
                      ) : video.cover_url ? (
                        <img className="work-cover-image" src={video.cover_url} alt={video.title || "视频封面"} />
                      ) : (
                        <div className="work-cover-empty">暂无封面</div>
                      )}

                      {statusText ? <div className="work-status-badge">{statusText}</div> : null}
                    </div>
                    <div className="work-title work-title-video">{video.title || "未命名视频"}</div>
                    <div className="work-metrics">
                      <span>{formatCount(video.play_count)} 播放</span>
                      <span>{formatCount(video.like_count)} 点赞</span>
                      <span>{formatCount(video.comment_count)} 评论</span>
                    </div>
                  </article>
                );
              })
            : null}
        </section>
      </section>

      {relationModalMode ? (
        <div className="auth-modal-backdrop" onClick={() => setRelationModalMode(null)}>
          <div className="auth-modal panel panel-roomy relation-modal" onClick={(event) => event.stopPropagation()}>
            <div className="relation-modal-header">
              <div>
                <h3>{relationModalTitle}</h3>
                <p>展示该用户的头像、用户名和昵称。</p>
              </div>
              <button className="auth-close-button" type="button" onClick={() => setRelationModalMode(null)}>
                X
              </button>
            </div>

            {relationLoading ? <div className="relation-modal-empty">正在加载列表...</div> : null}
            {relationError ? <div className="form-error">{relationError}</div> : null}
            {!relationLoading && !relationError && relationUsers.length === 0 ? <div className="relation-modal-empty">暂时还没有数据。</div> : null}

            {!relationLoading && relationUsers.length > 0 ? (
              <div className="relation-user-list">
                {relationUsers.map((item) => (
                  <article key={item.id} className="relation-user-card">
                    <div className="relation-user-avatar">{buildAvatarFallback(item.nickname || item.username || "U")}</div>
                    <div className="relation-user-main">
                      <div className="relation-user-name">{item.nickname || item.username}</div>
                      <div className="relation-user-handle">@{item.username}</div>
                    </div>
                  </article>
                ))}
              </div>
            ) : null}
          </div>
        </div>
      ) : null}
    </>
  );
}
