import { cookies } from "next/headers";
import { jwtVerify } from "jose";
import { normalizeRole, type Role } from "@/lib/roles";

export type SessionUser = {
  id: string;
  role: Role;
  email: string;
};

const ACCESS_COOKIE = "ft_at";

/**
 * Read and verify the access-token cookie on the server. Returns null when the
 * cookie is missing or invalid. The frontend never holds the JWT secret of the
 * backend beyond this shared value; it only verifies — the backend is the
 * authority for issuing tokens.
 */
export async function getSession(): Promise<SessionUser | null> {
  const secret = process.env.JWT_SECRET;
  if (!secret) return null;

  const store = await cookies();
  const token = store.get(ACCESS_COOKIE)?.value;
  if (!token) return null;

  try {
    const { payload } = await jwtVerify(token, new TextEncoder().encode(secret), {
      issuer: "fintrack",
      audience: "fintrack-web",
    });
    const sub = payload.sub as string | undefined;
    const role = payload.role as string | undefined;
    const email = payload.email as string | undefined;
    if (!sub || !role) return null;
    return { id: sub, role: normalizeRole(role), email: email ?? "" };
  } catch {
    return null;
  }
}
