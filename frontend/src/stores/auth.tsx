import {
  useCallback,
  createContext,
  useContext,
  useMemo,
  useState,
  type PropsWithChildren
} from "react";

import type { LoginData, ProfileData } from "../types/api";

const TOKEN_KEY = "tiktide_token";
const USER_KEY = "tiktide_user";

function persistToken(token: string | null) {
  if (token) {
    window.localStorage.setItem(TOKEN_KEY, token);
    return;
  }
  window.localStorage.removeItem(TOKEN_KEY);
}

function persistUser(user: ProfileData | null) {
  if (user) {
    window.localStorage.setItem(USER_KEY, JSON.stringify(user));
    return;
  }
  window.localStorage.removeItem(USER_KEY);
}

interface AuthContextValue {
  token: string | null;
  user: ProfileData | null;
  isAuthenticated: boolean;
  setSession: (session: LoginData) => void;
  updateUser: (user: ProfileData) => void;
  clearSession: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

function readStoredUser(): ProfileData | null {
  const raw = window.localStorage.getItem(USER_KEY);
  if (!raw) {
    return null;
  }
  try {
    return JSON.parse(raw) as ProfileData;
  } catch {
    return null;
  }
}

export function AuthProvider({ children }: PropsWithChildren) {
  const [token, setToken] = useState<string | null>(() => window.localStorage.getItem(TOKEN_KEY));
  const [user, setUser] = useState<ProfileData | null>(() => readStoredUser());

  const setSession = useCallback((session: LoginData) => {
    persistToken(session.token);
    persistUser(session.user);
    setToken(session.token);
    setUser(session.user);
  }, []);

  const updateUser = useCallback((nextUser: ProfileData) => {
    persistUser(nextUser);
    setUser(nextUser);
  }, []);

  const clearSession = useCallback(() => {
    persistToken(null);
    persistUser(null);
    setToken(null);
    setUser(null);
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      token,
      user,
      isAuthenticated: Boolean(token),
      setSession,
      updateUser,
      clearSession
    }),
    [clearSession, setSession, token, updateUser, user]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return context;
}
