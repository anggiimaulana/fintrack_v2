"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { Header } from "@/components/layout/Header";
import { useSession } from "@/lib/session-context";

export interface DashboardShellProps {
  sidebar: React.ReactNode;
  children: React.ReactNode;
  banner?: React.ReactNode;
  bottomNav?: React.ReactNode;
  role?: "user" | "admin" | "super_admin";
  requiredRole?: "admin" | "super_admin";
}

/**
 * Unified application shell for all dashboard views (user, admin, super-admin).
 * Provides a consistent sticky header, responsive main content scrolling container,
 * seamless sidebar integration with local storage persistence, and dynamic RBAC guards.
 */
export function DashboardShell({
  sidebar,
  children,
  banner,
  bottomNav,
  role,
  requiredRole,
}: DashboardShellProps) {
  const router = useRouter();
  const { user, isLoading, isAdmin, isSuperAdmin } = useSession();

  const effectiveRequiredRole = requiredRole || (role === "admin" || role === "super_admin" ? role : undefined);

  // Dynamic RBAC & suspension guard: redirects immediately if role or active flag changes in real-time
  React.useEffect(() => {
    if (isLoading || !user) return;

    if (user.is_active === false) {
      toast.error("Sesi Anda ditangguhkan: Akun telah dinonaktifkan.");
      router.replace("/login");
      return;
    }

    if (effectiveRequiredRole === "super_admin" && !isSuperAdmin) {
      toast.error("Akses dicabut: Anda tidak memiliki izin Super Admin.");
      router.replace("/user");
      return;
    }

    if (effectiveRequiredRole === "admin" && !isAdmin) {
      toast.error("Akses dicabut: Anda tidak memiliki izin Admin.");
      router.replace("/user");
      return;
    }
  }, [user, isLoading, isAdmin, isSuperAdmin, effectiveRequiredRole, router]);

  const hasBottomNav = Boolean(bottomNav || role === "user");

  return (
    <div className="flex h-screen w-full bg-background overflow-hidden text-foreground font-sans">
      {sidebar}
      <div className="flex flex-col flex-1 min-w-0 h-full overflow-hidden">
        <Header />
        {banner ? <div className="shrink-0">{banner}</div> : null}
        <main
          className={`flex-1 overflow-y-auto p-4 lg:p-6 ${
            hasBottomNav
              ? "pb-[calc(5rem+env(safe-area-inset-bottom))] md:pb-6"
              : "md:pb-6"
          }`}
        >
          {children}
        </main>
      </div>
      {bottomNav}
    </div>
  );
}
