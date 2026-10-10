import axios, { type AxiosError, type InternalAxiosRequestConfig } from "axios";

/**
 * API client. Requests go to the same-origin `/api/v1` proxy (see
 * next.config.ts rewrites), so the httpOnly auth cookies are sent
 * automatically — no Authorization header is needed.
 */
const api = axios.create({
  baseURL: "/api/v1",
  withCredentials: true,
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
      .post("/api/v1/auth/refresh", undefined, { withCredentials: true })
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
