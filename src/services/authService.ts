import { api, USE_MOCKS, tokenStore } from "@/lib/api/client";
import { mockCandidateUser, mockDelay, mockRecruiterUser } from "@/mocks/fixtures";
import type { AuthResponse, LoginPayload, RegisterPayload, User } from "@/types";

/** Endpoints expected from the Go + Gin backend. */
const routes = {
  login: "/api/v1/auth/login",
  register: "/api/v1/auth/register",
  me: "/api/v1/auth/me",
  logout: "/api/v1/auth/logout",
};

export const authService = {
  async login(payload: LoginPayload & { role?: User["role"] }): Promise<AuthResponse> {
    if (USE_MOCKS) {
      const user = payload.role === "recruiter" ? mockRecruiterUser : mockCandidateUser;
      return mockDelay({ token: "mock-jwt-token", user: { ...user, email: payload.email } });
    }
    return api.post<AuthResponse>(routes.login, payload);
  },

  async register(payload: RegisterPayload): Promise<AuthResponse> {
    if (USE_MOCKS) {
      const base = payload.role === "recruiter" ? mockRecruiterUser : mockCandidateUser;
      return mockDelay({
        token: "mock-jwt-token",
        user: { ...base, email: payload.email, fullName: payload.fullName, role: payload.role },
      });
    }
    return api.post<AuthResponse>(routes.register, payload);
  },

  async me(): Promise<User> {
    if (USE_MOCKS) return mockDelay(mockCandidateUser);
    return api.get<User>(routes.me);
  },

  async logout(): Promise<void> {
    if (!USE_MOCKS) await api.post<void>(routes.logout).catch(() => undefined);
    tokenStore.clear();
  },
};
