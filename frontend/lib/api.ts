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
let refreshPromise: Promise<void> | null = null;

function refreshSession(): Promise<void> {
  refreshPromise ??= axios
    .post("/api/v1/auth/refresh", undefined, { withCredentials: true })
    .then(() => undefined)
    .finally(() => {
      refreshPromise = null;
    });
  return refreshPromise;
}

api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const original = error.config as (InternalAxiosRequestConfig & { _retried?: boolean }) | undefined;

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
