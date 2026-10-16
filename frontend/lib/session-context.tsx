"use client";

import { createContext, useContext, useEffect, type ReactNode } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { me, type AuthUser } from "@/lib/auth-client";
import { subscribeDomainEvents } from "@/lib/realtime-bus";

type SessionContextValue = {
  user: AuthUser | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  isAdmin: boolean;
  isSuperAdmin: boolean;
};

const SessionContext = createContext<SessionContextValue>({
  user: null,
  isLoading: true,
  isAuthenticated: false,
  isAdmin: false,
  isSuperAdmin: false,
});

export function SessionProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ["session", "me"],
    queryFn: me,
    staleTime: 60_000,
    retry: false,
  });

  const user = data ?? null;

  // Realtime dynamic RBAC sync: refetch session when role or status updates
  useEffect(() => {
    return subscribeDomainEvents((event) => {
      if (
        event.event === "role_updated" ||
        event.event === "user_status_updated" ||
        event.event === "user_suspended"
      ) {
        queryClient.invalidateQueries({ queryKey: ["session", "me"] });
      }
    });
  }, [queryClient]);

  return (
    <SessionContext.Provider
      value={{
        user,
        isLoading,
        isAuthenticated: Boolean(user),
        isAdmin: user?.role === "admin" || user?.role === "super_admin",
        isSuperAdmin: user?.role === "super_admin",
      }}
    >
      {children}
    </SessionContext.Provider>
  );
}

export function useSession() {
  return useContext(SessionContext);
}
