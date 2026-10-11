"use client";

import * as React from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { 
  Home, ArrowLeftRight, FileText, MessageSquare, Settings, 
  LogOut, Wallet, User, MessagesSquare, History, Mic
} from "lucide-react";
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

const navItems = [
  { href: "/user", icon: Home, label: "Dashboard" },
  { href: "/user/transactions", icon: ArrowLeftRight, label: "Transaksi" },
  { href: "/user/ai", icon: MessageSquare, label: "AI Chat" },
  { href: "/user/reports", icon: FileText, label: "Laporan" },
  { href: "/user/profile", icon: User, label: "Profil" },
  { href: "/user/settings", icon: Settings, label: "Pengaturan" },
];

export function Sidebar() {
  const pathname = usePathname();
  const router = useRouter();

  const [dialogOpen, setDialogOpen] = React.useState(false);

  const handleSignOut = async () => {
    await logout();
    router.push("/login");
    router.refresh();
  };

  return (
    <aside className="hidden md:flex w-[240px] flex-col bg-sidebar text-sidebar-foreground h-screen sticky top-0 border-r-2 border-foreground">
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
        <div className="flex items-center gap-3 px-3 py-2 border-2 border-sidebar-border bg-sidebar-accent mb-2">
          <Avatar className="w-8 h-8 border-2 border-foreground">
            <AvatarImage src="" />
            <AvatarFallback className="bg-primary text-primary-foreground text-xs font-bold">US</AvatarFallback>
          </Avatar>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-bold text-sidebar-foreground truncate">User Biasa</p>
            <p className="text-xs text-sidebar-foreground/60 font-bold truncate uppercase">Premium</p>
          </div>
        </div>
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
              <DialogDescription>
                Apakah Anda yakin ingin logout?
              </DialogDescription>
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
