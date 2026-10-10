/**
 * Client-side auth helpers. All calls hit the same-origin `/api/v1/auth/*`
 * proxy, so the httpOnly auth cookies are sent/received automatically.
 */

export type AuthUser = {
  id: string;
  email: string;
  full_name: string;
  role: "admin" | "user";
  avatar_url?: string | null;
  is_active: boolean;
};

type ApiEnvelope<T> = { success: boolean; message: string; data?: T; error?: string };

async function post<T>(path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
    credentials: "same-origin",
  });
  const json = (await res.json().catch(() => ({}))) as ApiEnvelope<T>;
  if (!res.ok) {
    throw new Error(json.message || json.error || `Request failed (${res.status})`);
  }
  return json.data as T;
}

export function login(email: string, password: string): Promise<AuthUser> {
  return post<AuthUser>("/api/v1/auth/login", { email, password });
}

export function register(email: string, password: string, fullName: string): Promise<AuthUser> {
  return post<AuthUser>("/api/v1/auth/register", {
    email,
    password,
    full_name: fullName,
  });
}

export function refresh(): Promise<AuthUser> {
  return post<AuthUser>("/api/v1/auth/refresh");
}

export function logout(): Promise<void> {
  return post<void>("/api/v1/auth/logout");
}

export async function me(): Promise<AuthUser | null> {
  const res = await fetch("/api/v1/me", { credentials: "same-origin" });
  if (!res.ok) return null;
  const json = (await res.json()) as ApiEnvelope<AuthUser>;
  return json.data ?? null;
}
