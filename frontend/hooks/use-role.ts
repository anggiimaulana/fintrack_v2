"use client";

import { useSession } from "@/lib/session-context";
import { normalizeRole, type Role } from "@/lib/roles";

/**
 * Typed access to the signed-in user's platform role.
 *
 * Wraps the existing session query rather than issuing its own `/me` request —
 * role and profile come from the same payload, so a second fetch would be
 * redundant. This is a UX mirror: route guards live in `proxy.ts` and the
 * backend middleware is authoritative.
 */
export function useRole(): {
  role: Role;
  isLoading: boolean;
  isUser: boolean;
  isAdmin: boolean;
  isSuperAdmin: boolean;
} {
  const { user, isLoading } = useSession();
  const role = normalizeRole(user?.role);

  return {
    role,
    isLoading,
    isUser: role === "user",
    isAdmin: role === "admin",
    isSuperAdmin: role === "super_admin",
  };
}
