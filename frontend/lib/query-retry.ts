import type { AxiosError } from "axios";

/**
 * TanStack Query retry policy: never retry a client error (4xx).
 *
 * A 401/402/403 is a definitive answer — the request is unauthorised, over
 * quota, or the plan lacks the feature. Retrying only doubles the request
 * (visible as a duplicated call in the network tab) and delays the UI, since
 * the second attempt fails the same way. Server/network errors (5xx, timeouts)
 * are worth exactly one retry.
 */
export function shouldRetry(failureCount: number, error: unknown): boolean {
  const status = (error as AxiosError | undefined)?.response?.status;
  if (typeof status === "number" && status >= 400 && status < 500) {
    return false;
  }
  return failureCount < 1;
}
