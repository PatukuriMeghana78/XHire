import { ApiError, api, USE_MOCKS } from "@/lib/api/client";
import type { DetectedEmail, GmailStatus } from "@/types";

/**
 * Gmail transport for the candidate dashboard. Every call is scoped to the
 * authenticated user by the backend, so no user id is ever sent from here.
 *
 * Mock fixtures are deliberately not used: an unread email list or a "connected"
 * badge would claim access to a mailbox that was never authorized.
 */
const routes = {
  status: "/api/v1/candidate/email/status",
  messages: "/api/v1/candidate/email/messages",
  sync: "/api/v1/candidate/email/sync",
  connectUrl: "/api/v1/candidate/email/connect-url",
  disconnect: "/api/v1/candidate/email/disconnect",
};

const NO_BACKEND =
  "Gmail needs a running XHire backend. Set VITE_API_BASE_URL and sign in again.";

/** The status the dashboard shows when Gmail simply is not available. */
function unavailableStatus(reason: string): GmailStatus {
  return { connected: false, needsReconnect: false, hasRefreshToken: false, reason };
}

export const emailService = {
  async getStatus(): Promise<GmailStatus> {
    if (USE_MOCKS) return unavailableStatus(NO_BACKEND);
    return api.get<GmailStatus>(routes.status);
  },

  async listEmails(): Promise<DetectedEmail[]> {
    if (USE_MOCKS) return [];
    return api.get<DetectedEmail[]>(routes.messages);
  },

  /** Triggers a Gmail fetch. Throws with a `gmailCode` when the answer is actionable. */
  async sync(): Promise<{ processed: number }> {
    if (USE_MOCKS) throw new ApiError(NO_BACKEND, 503);
    return api.post<{ processed: number }>(routes.sync);
  },

  async getConnectUrl(): Promise<string> {
    if (USE_MOCKS) throw new ApiError(NO_BACKEND, 503);
    const { url } = await api.get<{ url: string }>(routes.connectUrl);
    return url;
  },

  async disconnect(): Promise<void> {
    if (USE_MOCKS) throw new ApiError(NO_BACKEND, 503);
    await api.post(routes.disconnect);
  },
};

/**
 * Machine-readable reason a Gmail action failed, so the UI can offer the right fix:
 * `not_connected` and `reconnect_required` are solved by the user authorizing Gmail,
 * `misconfigured` is a server deployment problem no user action can repair, and
 * `unavailable` covers everything else (transient Google or network failures).
 */
export type GmailFailure =
  | "not_connected"
  | "reconnect_required"
  | "misconfigured"
  | "unavailable";

export function gmailFailureOf(error: unknown): GmailFailure {
  const code = (error as { details?: { error?: string } })?.details?.error;
  switch (code) {
    case "gmail_not_connected":
      return "not_connected";
    case "gmail_reconnect_required":
      return "reconnect_required";
    case "gmail_misconfigured":
      return "misconfigured";
    default:
      return "unavailable";
  }
}

/** Human-readable message for a Gmail failure. */
export function gmailFailureMessage(error: unknown): string {
  const details = (error as { details?: { message?: string } })?.details;
  if (typeof details?.message === "string" && details.message.length > 0) {
    return details.message;
  }
  if (error instanceof Error && error.message) return error.message;
  return "Gmail could not be reached right now.";
}
