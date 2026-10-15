/**
 * Extracts a human-readable message from an API error without an unchecked cast.
 *
 * The error shape is unknown (Axios, fetch, or a thrown string), so each level
 * is narrowed with `in`/`typeof` before the field is read. Returns `fallback`
 * when no server message is present.
 */
export function apiErrorMessage(err: unknown, fallback = ''): string {
  if (typeof err !== 'object' || err === null || !('response' in err)) return fallback
  const { response } = err
  if (typeof response !== 'object' || response === null || !('data' in response)) return fallback
  const { data } = response
  if (typeof data !== 'object' || data === null || !('message' in data)) return fallback
  const { message } = data
  return typeof message === 'string' && message ? message : fallback
}
