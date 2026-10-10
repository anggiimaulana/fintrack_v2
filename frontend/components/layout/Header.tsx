"use client";

import * as React from "react";
import { usePathname, useRouter } from "next/navigation";
import { Bell, Search, Menu, Moon, Sun } from "lucide-react";
import { useTheme } from "next-themes";
import { useQueryClient } from "@tanstack/react-query";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Sheet, SheetContent, SheetTrigger, SheetTitle } from "@/components/ui/sheet";
import Link from "next/link";
import { Home, ArrowLeftRight, FileText, MessageSquare, Key, Settings, Wallet, LogOut } from "lucide-react";
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

const navItems = [
  { href: "/user", icon: Home, label: "Dashboard" },
  { href: "/user/transactions", icon: ArrowLeftRight, label: "Transaksi" },
  { href: "/user/reports", icon: FileText, label: "Laporan" },
  { href: "/user/ai", icon: MessageSquare, label: "AI Chat" },
  { href: "/user/api-keys", icon: Key, label: "API Keys" },
  { href: "/user/settings", icon: Settings, label: "Pengaturan" },
];

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
  const { user } = useSession();
  const [dialogOpen, setDialogOpen] = React.useState(false);

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
    if (pathname.startsWith("/user/api-keys")) return "API Keys";
    if (pathname.startsWith("/user/settings")) return "Pengaturan";
    return "Dashboard";
  };

  const initials = user?.full_name
    ? user.full_name.split(" ").map((w) => w[0]).slice(0, 2).join("").toUpperCase()
    : "US";

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
                  const isActive = pathname === item.href || (item.href !== "/user" && pathname.startsWith(item.href));
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
                <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
                  <DialogTrigger asChild>
                    <button className="flex items-center gap-3 px-3 py-2.5 w-full text-sm font-bold text-sidebar-foreground/70 hover:text-red-400 hover:bg-sidebar-accent transition-all cursor-pointer">
                      <LogOut className="w-5 h-5" />
                      Logout
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
            </div>
          </SheetContent>
        </Sheet>

        <h1 className="text-base font-black uppercase tracking-wide text-foreground">{getPageTitle()}</h1>
      </div>

      <div className="flex items-center gap-2 sm:gap-4">
        <div className="hidden sm:flex items-center relative">
          <Search className="w-4 h-4 absolute left-3 text-muted-foreground" />
          <input
            type="text"
            placeholder="Cari..."
            aria-label="Cari"
            className="pl-9 pr-4 py-1.5 text-sm border-2 border-foreground bg-background focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none w-48 lg:w-64 placeholder:text-muted-foreground/70"
          />
        </div>
        <ThemeToggle />
        <button aria-label="Notifikasi" className="p-2 text-muted-foreground hover:text-foreground relative cursor-pointer">
          <Bell className="w-5 h-5" />
          <span className="absolute top-1.5 right-1.5 w-2 h-2 bg-red-500 border-2 border-card" />
        </button>
        <Avatar className="w-8 h-8 border-2 border-foreground cursor-pointer">
          <AvatarImage src={user?.avatar_url ?? ""} alt={user?.full_name ?? "Avatar pengguna"} />
          <AvatarFallback className="bg-primary text-primary-foreground text-xs font-bold">{initials}</AvatarFallback>
        </Avatar>
      </div>
    </header>
  );
}
