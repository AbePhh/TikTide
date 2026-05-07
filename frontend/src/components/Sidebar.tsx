import { useState } from "react";
import { NavLink } from "react-router-dom";

import { changePassword, logout, updateProfile, updateUsername } from "../api/client";
import { creatorNavItems, mainNavItems } from "../constants/navigation";
import { useAuth } from "../stores/auth";
import { buildAvatarFallback } from "../utils/format";
import { AuthFormModal } from "./AuthFormModal";

function navClassName({ isActive }: { isActive: boolean }) {
  return isActive ? "nav-link nav-link-active" : "nav-link";
}

export function Sidebar() {
  const { user, updateUser, clearSession } = useAuth();
  const [profileOpen, setProfileOpen] = useState(false);
  const [switchAccountOpen, setSwitchAccountOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [username, setUsername] = useState(user?.username ?? "");
  const [nickname, setNickname] = useState(user?.nickname ?? "");
  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");

  const resolvedNickname = user?.nickname ?? "未登录用户";
  const resolvedUsername = user?.username ?? "guest";
  const avatar = buildAvatarFallback(resolvedNickname);

  function openProfile() {
    setUsername(user?.username ?? "");
    setNickname(user?.nickname ?? "");
    setOldPassword("");
    setNewPassword("");
    setError(null);
    setSuccess(null);
    setProfileOpen(true);
  }

  function closeProfile() {
    setProfileOpen(false);
    setError(null);
    setSuccess(null);
  }

  async function handleSaveProfile() {
    setBusy(true);
    setError(null);
    setSuccess(null);
    try {
      let nextUser = user;
      if (username.trim() && username.trim() !== user?.username) {
        nextUser = await updateUsername(username.trim());
      }
      if (nickname.trim() !== (nextUser?.nickname ?? "")) {
        nextUser = await updateProfile({ nickname: nickname.trim() });
      }
      if (nextUser) {
        updateUser(nextUser);
      }
      setSuccess("个人信息已更新。");
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("个人信息更新失败。");
      }
    } finally {
      setBusy(false);
    }
  }

  async function handleChangePassword() {
    if (!oldPassword.trim() || !newPassword.trim()) {
      setError("请输入旧密码和新密码。");
      return;
    }

    setBusy(true);
    setError(null);
    setSuccess(null);
    try {
      await changePassword({
        old_password: oldPassword,
        new_password: newPassword
      });
      setOldPassword("");
      setNewPassword("");
      setSuccess("密码修改成功。");
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("密码修改失败。");
      }
    } finally {
      setBusy(false);
    }
  }

  async function handleLogout() {
    try {
      await logout();
    } catch {
      // Keep local logout even if server-side blacklist write fails.
    }
    clearSession();
    closeProfile();
  }

  return (
    <>
      <aside className="sidebar">
        <div className="brand-block">
          <div className="brand-mark">T</div>
          <div>
            <div className="brand-name">TikTide</div>
            <div className="brand-subtitle">短视频网页端</div>
          </div>
        </div>

        <nav className="nav-section">
          <div className="nav-title">内容浏览</div>
          {mainNavItems.map((item) => (
            <NavLink key={item.to} to={item.to} className={navClassName}>
              <span>{item.label}</span>
            </NavLink>
          ))}
        </nav>

        <nav className="nav-section">
          <div className="nav-title">创作与管理</div>
          {creatorNavItems.map((item) => (
            <NavLink key={item.to} to={item.to} className={navClassName}>
              <span>{item.label}</span>
            </NavLink>
          ))}
        </nav>

        <div className="sidebar-footer">
          <button className="sidebar-user-card sidebar-user-card-button" type="button" onClick={openProfile}>
            <div className="sidebar-user-avatar">{avatar}</div>
            <div>
              <div className="sidebar-user-name">{resolvedNickname}</div>
              <div className="sidebar-user-handle">@{resolvedUsername}</div>
            </div>
          </button>
        </div>
      </aside>

      {profileOpen ? (
        <div className="auth-modal-backdrop" onClick={closeProfile}>
          <section className="auth-modal panel panel-roomy profile-settings-modal" onClick={(event) => event.stopPropagation()}>
            <div className="auth-modal-header">
              <div>
                <h3>个人信息</h3>
                <p>这里可以修改用户名、昵称、密码，也可以切换账号或退出登录。</p>
              </div>
              <button className="auth-close-button" type="button" onClick={closeProfile}>
                ×
              </button>
            </div>

            <div className="profile-settings-grid">
              <div className="profile-settings-section">
                <h4>基本资料</h4>
                <label className="form-field">
                  <span>用户名</span>
                  <input value={username} onChange={(event) => setUsername(event.target.value)} placeholder="3-32 位字母、数字或下划线" />
                </label>
                <label className="form-field">
                  <span>昵称</span>
                  <input value={nickname} onChange={(event) => setNickname(event.target.value)} placeholder="输入展示昵称" />
                </label>
                <button className="primary-button profile-settings-submit" type="button" onClick={() => void handleSaveProfile()} disabled={busy}>
                  保存资料
                </button>
              </div>

              <div className="profile-settings-section">
                <h4>安全设置</h4>
                <label className="form-field">
                  <span>旧密码</span>
                  <input type="password" value={oldPassword} onChange={(event) => setOldPassword(event.target.value)} placeholder="输入旧密码" />
                </label>
                <label className="form-field">
                  <span>新密码</span>
                  <input type="password" value={newPassword} onChange={(event) => setNewPassword(event.target.value)} placeholder="至少 8 位密码" />
                </label>
                <button className="ghost-button profile-settings-submit" type="button" onClick={() => void handleChangePassword()} disabled={busy}>
                  修改密码
                </button>
              </div>
            </div>

            {error ? <div className="form-error">{error}</div> : null}
            {success ? <div className="form-success">{success}</div> : null}

            <div className="profile-settings-actions">
              <button className="ghost-button" type="button" onClick={() => setSwitchAccountOpen(true)}>
                切换账号
              </button>
              <button className="primary-button" type="button" onClick={() => void handleLogout()}>
                退出登录
              </button>
            </div>
          </section>
        </div>
      ) : null}

      {switchAccountOpen ? (
        <AuthFormModal
          defaultMode="login"
          title="切换账号"
          subtitle="登录另一个账号后，当前会话会被新的登录结果覆盖。"
          onClose={() => setSwitchAccountOpen(false)}
          onSuccess={() => {
            setSwitchAccountOpen(false);
            closeProfile();
          }}
        />
      ) : null}
    </>
  );
}
