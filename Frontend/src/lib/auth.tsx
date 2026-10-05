import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";

import { tokenStore } from "@/lib/api/client";
import { authService } from "@/services";
import type { LoginPayload, RegisterPayload, User, UserRole } from "@/types";

const USER_KEY = "xhire.user";

interface AuthContextValue {
  user: User | null;
  /** False until localStorage has been read on the client. */
  ready: boolean;
  login: (payload: LoginPayload & { role?: UserRole }) => Promise<User>;
  register: (payload: RegisterPayload) => Promise<User>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function homeForRole(role: UserRole) {
  return role === "recruiter" ? "/recruiter" : "/candidate";
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    try {
      const raw = window.localStorage.getItem(USER_KEY);
      if (raw && tokenStore.get()) setUser(JSON.parse(raw) as User);
    } catch {
      /* ignore corrupted storage */
    }
    setReady(true);
  }, []);

  const persist = useCallback((token: string, nextUser: User) => {
    tokenStore.set(token);
    window.localStorage.setItem(USER_KEY, JSON.stringify(nextUser));
    setUser(nextUser);
  }, []);

  const login = useCallback<AuthContextValue["login"]>(
    async (payload) => {
      const { token, user: nextUser } = await authService.login(payload);
      persist(token, nextUser);
      return nextUser;
    },
    [persist],
  );

  const register = useCallback<AuthContextValue["register"]>(
    async (payload) => {
      const { token, user: nextUser } = await authService.register(payload);
      persist(token, nextUser);
      return nextUser;
    },
    [persist],
  );

  const logout = useCallback(async () => {
    await authService.logout();
    window.localStorage.removeItem(USER_KEY);
    setUser(null);
  }, []);

  const value = useMemo(
    () => ({ user, ready, login, register, logout }),
    [user, ready, login, register, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside <AuthProvider>");
  return ctx;
}
