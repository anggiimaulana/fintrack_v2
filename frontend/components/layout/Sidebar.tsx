"use client";

import * as React from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { LogOut, Wallet } from "lucide-react";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { logout } from "@/lib/auth-client";
import { useSession } from "@/lib/session-context";
import { USER_NAV_ITEMS, isNavItemActive } from "@/lib/nav";
import { useSidebarCollapsed } from "@/hooks/use-sidebar-collapsed";
import { useEntitlements } from "@/hooks/use-entitlements";
import { cn } from "@/lib/utils";

const navItems = USER_NAV_ITEMS;

export function Sidebar() {
  const pathname = usePathname();
  const router = useRouter();
  const { user } = useSession();
  const { planName, planCode } = useEntitlements();
  const { collapsed } = useSidebarCollapsed();
  const activePlan = user?.role && user.role !== "user" ? user.role : (planName || planCode || "Free");
  const [dialogOpen, setDialogOpen] = React.useState(false);

  const handleSignOut = async () => {
    await logout();
    router.push("/login");
    router.refresh();
  };

  const displayName = user?.full_name || "User";
  const initials = displayName
    .split(" ")
    .map((w) => w[0])
    .filter(Boolean)
    .slice(0, 2)
    .join("")
    .toUpperCase();

  return (
    <aside
      data-collapsed={collapsed}
      className={cn(
        "hidden md:flex flex-col bg-sidebar text-sidebar-foreground h-screen sticky top-0 border-r-2 border-foreground shrink-0",
        "transition-[width] duration-300 ease-in-out",
        collapsed ? "w-[76px]" : "w-[240px]"
      )}
    >
      {/* Brand. The collapse toggle lives in the Header, beside the page title,
          so it never sits in the nav column (which made it look like a menu
          item when collapsed). */}
      <div className={cn("border-b-2 border-sidebar-border flex items-center", collapsed ? "p-3 justify-center" : "p-4")}>
        <Link href="/user" className="flex items-center gap-2 text-xl font-black tracking-tight overflow-hidden">
          <Wallet className="w-6 h-6 text-primary shrink-0" />
          <span
            className={cn(
              "whitespace-nowrap transition-all duration-200",
              collapsed ? "w-0 opacity-0" : "w-auto opacity-100"
            )}
          >
            Fin<span className="text-primary">Track</span>
          </span>
        </Link>
      </div>

      <nav className="flex-1 px-3 py-4 space-y-1 overflow-y-auto overflow-x-hidden no-scrollbar">
        {navItems.map((item) => {
          const isActive = isNavItemActive(pathname, item.href);
          return (
            <Link
              key={item.href}
              href={item.href}
              title={collapsed ? item.label : undefined}
              aria-current={isActive ? "page" : undefined}
              className={cn(
                "group relative flex items-center gap-3 px-3 py-2.5 text-sm font-bold transition-all cursor-pointer",
                collapsed && "justify-center px-0",
                isActive
                  ? "bg-primary text-primary-foreground border-2 rounded-md border-foreground shadow-[2px_2px_0px_0px] shadow-foreground"
                  : "text-sidebar-foreground/70 hover:text-sidebar-foreground hover:bg-sidebar-accent"
              )}
            >
              <item.icon className="w-5 h-5 shrink-0" />
              <span
                className={cn(
                  "whitespace-nowrap transition-all duration-200",
                  collapsed ? "w-0 opacity-0 overflow-hidden" : "w-auto opacity-100"
                )}
              >
                {item.label}
              </span>
              {/* Tooltip when collapsed */}
              {collapsed && (
                <span className="pointer-events-none absolute left-full ml-3 z-50 hidden group-hover:block whitespace-nowrap bg-foreground text-background text-xs font-bold px-2.5 py-1.5 border-2 rounded-md border-foreground shadow-[2px_2px_0px_0px] shadow-foreground/40">
                  {item.label}
                </span>
              )}
            </Link>
          );
        })}
      </nav>

      <div className={cn("border-t-2 border-sidebar-border", collapsed ? "p-2" : "p-4")}>
        {collapsed ? (
          <div className="flex justify-center mb-2" title={displayName}>
            <Avatar className="w-9 h-9 rounded-full">
              {user?.avatar_url ? <AvatarImage src={user.avatar_url} alt={displayName} /> : null}
              <AvatarFallback className="bg-primary text-primary-foreground text-xs font-bold">{initials}</AvatarFallback>
            </Avatar>
          </div>
        ) : (
          <div className="flex items-center gap-3 px-3 py-2 border-2 rounded-md border-sidebar-border bg-sidebar-accent mb-2">
            <Avatar className="w-8 h-8 rounded-full">
              {user?.avatar_url ? <AvatarImage src={user.avatar_url} alt={displayName} /> : null}
              <AvatarFallback className="bg-primary text-primary-foreground text-xs font-bold">{initials}</AvatarFallback>
            </Avatar>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-bold text-sidebar-foreground truncate">{displayName}</p>
              <p className="text-xs text-sidebar-foreground/60 font-bold truncate uppercase">{activePlan}</p>
            </div>
          </div>
        )}

        <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
          <DialogTrigger asChild>
            <button
              aria-label="Logout"
              title={collapsed ? "Logout" : undefined}
              className={cn(
                "flex items-center gap-3 py-2.5 w-full text-sm font-bold text-sidebar-foreground/70 hover:text-red-400 hover:bg-sidebar-accent transition-all cursor-pointer",
                collapsed ? "justify-center px-0" : "px-3"
              )}
            >
              <LogOut className="w-5 h-5 shrink-0" />
              <span className={cn("whitespace-nowrap transition-all duration-200", collapsed ? "w-0 opacity-0 overflow-hidden" : "w-auto opacity-100")}>
                Logout
              </span>
            </button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Konfirmasi Logout</DialogTitle>
              <DialogDescription>Apakah Anda yakin ingin logout?</DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <Button variant="outline" onClick={() => setDialogOpen(false)}>
                Batal
              </Button>
              <Button variant="destructive" onClick={handleSignOut}>
                Logout
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </aside>
  );
}
