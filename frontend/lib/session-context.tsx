"use client";

import { createContext, useContext, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { me, type AuthUser } from "@/lib/auth-client";

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
  const { data, isLoading } = useQuery({
    queryKey: ["session", "me"],
    queryFn: me,
    staleTime: 60_000,
    retry: false,
  });

  const user = data ?? null;

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
