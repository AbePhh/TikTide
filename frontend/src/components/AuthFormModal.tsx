import { useState } from "react";

import { ApiError, login, register } from "../api/client";
import { useAuth } from "../stores/auth";

type AuthMode = "login" | "register";

interface AuthFormModalProps {
  defaultMode?: AuthMode;
  title?: string;
  subtitle?: string;
  onClose: () => void;
  onSuccess?: () => void;
}

export function AuthFormModal({
  defaultMode = "login",
  title,
  subtitle,
  onClose,
  onSuccess
}: AuthFormModalProps) {
  const { setSession } = useAuth();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [authMode, setAuthMode] = useState<AuthMode>(defaultMode);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");

  async function handleSubmit() {
    const trimmedUsername = username.trim();
    const trimmedPassword = password.trim();
    if (!trimmedUsername || !trimmedPassword) {
      setError("请输入用户名和密码");
      return;
    }

    setPending(true);
    setError(null);
    try {
      if (authMode === "register") {
        await register(trimmedUsername, trimmedPassword);
      }
      const session = await login(trimmedUsername, trimmedPassword);
      setSession(session);
      onSuccess?.();
      onClose();
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError(authMode === "register" ? "注册失败，请稍后重试" : "登录失败，请稍后重试");
      }
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="auth-modal-backdrop" onClick={onClose}>
      <section className="auth-modal panel panel-roomy" onClick={(event) => event.stopPropagation()}>
        <div className="auth-modal-header">
          <div>
            <h3>{title ?? (authMode === "login" ? "登录 TikTide" : "注册 TikTide")}</h3>
            <p>{subtitle ?? (authMode === "login" ? "输入用户名和密码登录当前系统" : "先注册，再自动登录进入系统")}</p>
          </div>
          <button className="auth-close-button" type="button" onClick={onClose}>
            ×
          </button>
        </div>

        <div className="auth-mode-switch">
          <button className={authMode === "login" ? "auth-tab auth-tab-active" : "auth-tab"} type="button" onClick={() => setAuthMode("login")}>
            登录
          </button>
          <button className={authMode === "register" ? "auth-tab auth-tab-active" : "auth-tab"} type="button" onClick={() => setAuthMode("register")}>
            注册
          </button>
        </div>

        <label className="form-field">
          <span>用户名</span>
          <input value={username} onChange={(event) => setUsername(event.target.value)} placeholder="3-32 位字母、数字或下划线" />
        </label>

        <label className="form-field">
          <span>密码</span>
          <input type="password" value={password} onChange={(event) => setPassword(event.target.value)} placeholder="至少 8 位密码" />
        </label>

        {error ? <div className="form-error">{error}</div> : null}

        <div className="auth-modal-actions">
          <button className="ghost-button" type="button" onClick={onClose} disabled={pending}>
            取消
          </button>
          <button className="primary-button" type="button" onClick={() => void handleSubmit()} disabled={pending}>
            {pending ? "提交中..." : authMode === "login" ? "立即登录" : "注册并登录"}
          </button>
        </div>
      </section>
    </div>
  );
}
