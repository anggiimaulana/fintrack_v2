import axios, { type AxiosError, type InternalAxiosRequestConfig } from "axios";

/**
 * Fallback budget for long-running AI calls (receipt vision, voice
 * transcription, chat streaming) used until the server-reported value arrives.
 * It deliberately exceeds axios' default because the upstream model can take a
 * minute or more to answer; cutting the request short would surface as a bogus
 * "timeout" to the user.
 */
export const AI_REQUEST_TIMEOUT_MS_FALLBACK = 180_000;

/**
 * The live AI request budget, shared across the app.
 *
 * A Super Admin can retune the server-side timeout at runtime; the client
 * mirrors it via GET /me/ai-config so the browser never aborts before the server
 * would. Exposing it as a mutable object (rather than a constant baked into each
 * call site) lets the value change without a rebuild: call sites read
 * `aiRequestTimeout.ms` at request time.
 */
export const aiRequestTimeout = { ms: AI_REQUEST_TIMEOUT_MS_FALLBACK };

/**
 * Applies a server-reported budget, ignoring non-positive/invalid values so the
 * fallback always stands as a floor.
 */
export function applyAIRequestTimeout(ms: number): void {
  if (Number.isFinite(ms) && ms > 0) {
    aiRequestTimeout.ms = ms;
  }
}

/**
 * API client. Requests go to the same-origin `/api/v1` proxy (see
 * next.config.ts rewrites), so the httpOnly auth cookies are sent
 * automatically — no Authorization header is needed.
 */
const api = axios.create({
  baseURL: "/api/v1",
  withCredentials: true,
  headers: {
    "X-Requested-With": "XMLHttpRequest",
  },
});

// Single-flight refresh: concurrent 401s trigger at most one refresh call.
let inFlightRefresh: Promise<void> | null = null;

/**
 * Refresh the session, coalescing concurrent callers into a single request.
 * Exported for testing.
 */
export function refreshSession(): Promise<void> {
  if (!inFlightRefresh) {
    inFlightRefresh = axios
      .post("/api/v1/auth/refresh", undefined, {
        withCredentials: true,
        headers: { "X-Requested-With": "XMLHttpRequest" },
      })
      .then(() => undefined)
      .finally(() => {
        inFlightRefresh = null;
      });
  }
  return inFlightRefresh;
}

type RetriableConfig = InternalAxiosRequestConfig & { _retried?: boolean };

api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const original = error.config as RetriableConfig | undefined;

    if (error.response?.status === 401 && original && !original._retried) {
      original._retried = true;
      try {
        await refreshSession();
        return api.request(original);
      } catch {
        if (typeof window !== "undefined") {
          window.location.href = "/login";
        }
        return Promise.reject(error);
      }
    }

    return Promise.reject(error);
  }
);

export default api;
