"use client";

import * as React from "react";
import { usePathname, useRouter } from "next/navigation";
import { Search, Menu, Moon, Sun, PanelLeftClose, PanelLeftOpen } from "lucide-react";
import { useTheme } from "next-themes";
import { useQueryClient } from "@tanstack/react-query";
import { Sheet, SheetContent, SheetTrigger, SheetTitle } from "@/components/ui/sheet";
import Link from "next/link";
import { Wallet, LogOut } from "lucide-react";
import { USER_NAV_ITEMS, isNavItemActive } from "@/lib/nav";
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

const navItems = USER_NAV_ITEMS;

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
    if (pathname.startsWith("/user/settings/subscription")) return "Langganan";
    return "Dashboard";
  };

  return (
    <header className="h-14 border-b-2 border-foreground bg-card sticky top-0 z-40 flex items-center justify-between px-4 lg:px-6">
      <div className="flex items-center gap-4">
        <Sheet>
          <SheetTrigger asChild>
            <button aria-label="Buka menu navigasi" className="md:hidden p-2 -ml-2 text-muted-foreground hover:text-foreground cursor-pointer">
              <Menu className="w-5 h-5" />
            </button>
          </SheetTrigger>
          <SheetContent side="left" className="p-0 w-[260px] bg-sidebar text-sidebar-foreground border-r-2 border-foreground">
            <SheetTitle className="sr-only">Navigasi</SheetTitle>
            <div className="flex flex-col h-full">
              <div className="p-6 border-b-2 border-sidebar-border">
                <Link href="/user" className="flex items-center gap-2 text-xl font-black tracking-tight">
                  <Wallet className="w-6 h-6 text-primary" />
                  Fin<span className="text-primary">Track</span>
                </Link>
              </div>

              <nav className="flex-1 px-3 py-4 space-y-1 overflow-y-auto">
                {navItems.map((item) => {
                  const isActive = isNavItemActive(pathname, item.href);
                  return (
                    <Link
                      key={item.href}
                      href={item.href}
                      className={`flex items-center gap-3 px-3 py-2.5 text-sm font-bold transition-all cursor-pointer ${
                        isActive
                          ? "bg-primary text-primary-foreground border-2 border-foreground shadow-[2px_2px_0px_0px] shadow-foreground"
                          : "text-sidebar-foreground/70 hover:text-sidebar-foreground hover:bg-sidebar-accent"
                      }`}
                    >
                      <item.icon className="w-5 h-5" />
                      {item.label}
                    </Link>
                  );
                })}
              </nav>

              <div className="p-4 border-t-2 border-sidebar-border">
                {/* Opens the shared logout confirmation rendered by the header. */}
                <button
                  type="button"
                  onClick={() => setDialogOpen(true)}
                  className="flex items-center gap-3 px-3 py-2.5 w-full text-sm font-bold text-sidebar-foreground/70 hover:text-red-400 hover:bg-sidebar-accent transition-all cursor-pointer"
                >
                  <LogOut className="w-5 h-5" />
                  Logout
                </button>
              </div>
            </div>
          </SheetContent>
        </Sheet>

        {/* Desktop sidebar toggle: lives here, beside the page title, so it is
            never mistaken for a menu item inside the collapsed rail. */}
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
          className="hidden sm:flex items-center gap-2 w-48 lg:w-64 pl-3 pr-2 py-1.5 text-sm border-2 border-foreground bg-background hover:shadow-[2px_2px_0px_0px] hover:shadow-foreground transition-shadow cursor-pointer text-muted-foreground"
        >
          <Search className="w-4 h-4 shrink-0" aria-hidden="true" />
          <span className="flex-1 text-left truncate">Cari modul atau fitur…</span>
          <kbd className="hidden lg:inline-flex text-[10px] font-bold border-2 border-foreground/30 px-1">⌘K</kbd>
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
