"use client";

import * as React from "react";
import { usePathname, useRouter } from "next/navigation";
import { Search, Moon, Sun, Laptop, PanelLeftClose, PanelLeftOpen } from "lucide-react";
import { useTheme } from "@/providers/theme-provider";
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
  const { theme, setTheme } = useTheme();
  const [mounted, setMounted] = React.useState(false);
  React.useEffect(() => {
    const id = requestAnimationFrame(() => setMounted(true));
    return () => cancelAnimationFrame(id);
  }, []);

  const cycleTheme = () => {
    if (theme === "light") setTheme("dark");
    else if (theme === "dark") setTheme("system");
    else setTheme("light");
  };

  const getThemeInfo = () => {
    if (!mounted) return { icon: <Sun className="w-5 h-5" />, label: "Tema: Terang" };
    if (theme === "dark") return { icon: <Moon className="w-5 h-5 text-blue-400" />, label: "Tema: Gelap (Klik untuk Sistem)" };
    if (theme === "system") return { icon: <Laptop className="w-5 h-5 text-primary" />, label: "Tema: Sistem (Klik untuk Terang)" };
    return { icon: <Sun className="w-5 h-5 text-amber-500" />, label: "Tema: Terang (Klik untuk Gelap)" };
  };

  const { icon, label } = getThemeInfo();

  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      onClick={cycleTheme}
      className="p-2 text-muted-foreground hover:text-foreground cursor-pointer transition-colors"
    >
      {icon}
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
    // User routes
    if (pathname === "/user") return "Dashboard";
    if (pathname.startsWith("/user/transactions")) return "Transaksi";
    if (pathname.startsWith("/user/reports")) return "Laporan";
    if (pathname.startsWith("/user/ai")) return "AI Chat";
    if (pathname.startsWith("/user/profile")) return "Profil & Pengaturan";
    if (pathname.startsWith("/user/usage")) return "Penggunaan";
    if (pathname.startsWith("/user/subscription")) return "Langganan";
    if (pathname.startsWith("/user/notifications")) return "Notifikasi";

    // Admin routes
    if (pathname === "/admin") return "Dashboard Admin";
    if (pathname.startsWith("/admin/users")) return "Manajemen Pengguna";
    if (pathname.startsWith("/admin/providers")) return "Monitoring Provider";
    if (pathname.startsWith("/admin/categories/expense")) return "Kategori Pengeluaran";
    if (pathname.startsWith("/admin/categories/income")) return "Kategori Pemasukan";
    if (pathname.startsWith("/admin/categories/savings")) return "Kategori Tabungan";

    // Super Admin routes
    if (pathname === "/super-admin/tokens") return "Token & Kuota AI";
    if (pathname.startsWith("/super-admin/subscriptions")) return "Manajemen Langganan";
    if (pathname.startsWith("/super-admin/payments/review")) return "Review Pembayaran Manual";
    if (pathname.startsWith("/super-admin/payments")) return "Pembayaran & Transaksi";
    if (pathname.startsWith("/super-admin/plans")) return "Katalog & Manajemen Paket";
    if (pathname.startsWith("/super-admin/audit")) return "Jejak Audit (Audit Log)";
    if (pathname.startsWith("/super-admin/ai-gateway")) return "Operasional AI Gateway";
    if (pathname.startsWith("/super-admin/settings")) return "Pengaturan Platform Global";
    if (pathname.startsWith("/super-admin/metrics")) return "Metrik & Performa Sistem";
    if (pathname.startsWith("/super-admin/kuota") || pathname.startsWith("/super-admin/quota-monitor")) return "Monitoring Kuota AI";
    if (pathname.startsWith("/super-admin/pricing")) return "Konfigurasi Harga & Token";
    if (pathname.startsWith("/super-admin/spk-metrics")) return "Metrik SPK";
    if (pathname === "/super-admin") return "Dashboard Super Admin";

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

        <div className="flex items-center gap-1.5 text-xs sm:text-sm font-black uppercase tracking-wide">
          {pathname.startsWith("/super-admin") && pathname !== "/super-admin" ? (
            <>
              <span className="text-muted-foreground hidden sm:inline">Super Admin</span>
              <span className="text-muted-foreground/60 hidden sm:inline">/</span>
              <h1 className="text-foreground inline truncate">{getPageTitle()}</h1>
            </>
          ) : pathname.startsWith("/admin") && pathname !== "/admin" ? (
            <>
              <span className="text-muted-foreground hidden sm:inline">Admin</span>
              <span className="text-muted-foreground/60 hidden sm:inline">/</span>
              <h1 className="text-foreground inline truncate">{getPageTitle()}</h1>
            </>
          ) : (
            <h1 className="text-foreground truncate">{getPageTitle()}</h1>
          )}
        </div>
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
