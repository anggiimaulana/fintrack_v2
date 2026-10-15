"use client";

import * as React from "react";
import { usePathname, useRouter } from "next/navigation";
import { Search, Moon, Sun, PanelLeftClose, PanelLeftOpen } from "lucide-react";
import { useTheme } from "next-themes";
import { useQueryClient } from "@tanstack/react-query";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { logout } from "@/lib/auth-client";
import { useSidebarCollapsed } from "@/hooks/use-sidebar-collapsed";
import { SearchCommand } from "@/components/layout/SearchCommand";
import { NotificationBell } from "@/components/layout/NotificationBell";
import { ProfileMenu } from "@/components/layout/ProfileMenu";

function ThemeToggle() {
  const { resolvedTheme, setTheme } = useTheme();
  const [mounted, setMounted] = React.useState(false);
  React.useEffect(() => {
    // next-themes only knows the resolved theme after hydration; flipping this
    // once avoids a server/client icon mismatch without a render-cascading write.
    const id = requestAnimationFrame(() => setMounted(true));
    return () => cancelAnimationFrame(id);
  }, []);

  return (
    <button
      type="button"
      aria-label="Ganti tema terang/gelap"
      onClick={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")}
      className="p-2 text-muted-foreground hover:text-foreground cursor-pointer"
    >
      {mounted && resolvedTheme === "dark" ? <Sun className="w-5 h-5" /> : <Moon className="w-5 h-5" />}
    </button>
  );
}

export function Header() {
  const pathname = usePathname();
  const router = useRouter();
  const queryClient = useQueryClient();
  const { collapsed, toggle } = useSidebarCollapsed();
  const [dialogOpen, setDialogOpen] = React.useState(false);
  const [searchOpen, setSearchOpen] = React.useState(false);

  // Ctrl/Cmd-K opens the search palette from anywhere in the dashboard.
  React.useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setSearchOpen((v) => !v);
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);

  const handleSignOut = async () => {
    try {
      await logout();
    } finally {
      queryClient.clear();
      router.push("/login");
      router.refresh();
    }
  };

  const getPageTitle = () => {
    if (pathname === "/user") return "Dashboard";
    if (pathname.startsWith("/user/transactions")) return "Transaksi";
    if (pathname.startsWith("/user/reports")) return "Laporan";
    if (pathname.startsWith("/user/ai")) return "AI Chat";
    if (pathname.startsWith("/user/profile")) return "Profil & Pengaturan";
    if (pathname.startsWith("/user/usage")) return "Penggunaan";
    if (pathname.startsWith("/user/subscription")) return "Langganan";
    return "Dashboard";
  };

  return (
    <header className="h-14 border-b-2 border-foreground bg-card sticky top-0 z-40 flex items-center justify-between px-4 lg:px-6">
      <div className="flex items-center gap-4">
        {/* Desktop sidebar toggle: lives here, beside the page title, so it is
            never mistaken for a menu item inside the collapsed rail. Mobile
            navigation lives in the bottom bar instead (MobileBottomNav). */}
        <button
          type="button"
          aria-label={collapsed ? "Buka sidebar" : "Tutup sidebar"}
          aria-expanded={!collapsed}
          onClick={toggle}
          className="hidden md:inline-flex p-2 -ml-2 text-muted-foreground hover:text-foreground cursor-pointer"
        >
          {collapsed ? <PanelLeftOpen className="w-5 h-5" /> : <PanelLeftClose className="w-5 h-5" />}
        </button>

        <h1 className="text-base font-black uppercase tracking-wide text-foreground">{getPageTitle()}</h1>
      </div>

      <div className="flex items-center gap-2 sm:gap-4">
        {/* Search opens the command palette: it finds modules/features and
            navigates on click. A read-only input would look interactive but do
            nothing, which is exactly what users reported. */}
        <button
          type="button"
          onClick={() => setSearchOpen(true)}
          aria-label="Cari modul atau fitur"
          className="hidden sm:flex items-center gap-2 w-48 lg:w-64 pl-3 pr-2 py-1.5 text-sm border-2 rounded-md border-foreground bg-background hover:shadow-[2px_2px_0px_0px] hover:shadow-foreground transition-shadow cursor-pointer text-muted-foreground"
        >
          <Search className="w-4 h-4 shrink-0" aria-hidden="true" />
          <span className="flex-1 text-left truncate">Cari modul atau fitur…</span>
          <kbd className="hidden lg:inline-flex text-[10px] font-bold border-2 rounded-md border-foreground/30 px-1">⌘K</kbd>
        </button>
        {/* Compact icon-only search on small screens. */}
        <button
          type="button"
          onClick={() => setSearchOpen(true)}
          aria-label="Cari modul atau fitur"
          className="sm:hidden p-2 text-muted-foreground hover:text-foreground cursor-pointer"
        >
          <Search className="w-5 h-5" />
        </button>
        <ThemeToggle />
        <NotificationBell />
        <ProfileMenu onLogout={() => setDialogOpen(true)} />
      </div>

      {/* Logout confirmation: shared by the profile menu and the mobile sheet. */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-sm">
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

      <SearchCommand open={searchOpen} onOpenChange={setSearchOpen} />
    </header>
  );
}
