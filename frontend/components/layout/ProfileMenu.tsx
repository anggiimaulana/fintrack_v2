"use client";

import Link from "next/link";
import { User, CreditCard, LogOut } from "lucide-react";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useSession } from "@/lib/session-context";

interface ProfileMenuProps {
  /** Called after the user confirms logout (the header owns the confirm dialog). */
  onLogout: () => void;
}

/**
 * The header avatar menu: profile, plan, settings, and logout.
 *
 * Logout is delegated to the parent so the confirmation dialog lives in one
 * place (the header already owns it) and this menu stays presentational.
 */
export function ProfileMenu({ onLogout }: ProfileMenuProps) {
  const { user } = useSession();

  const displayName = user?.full_name || "Pengguna";
  const initials = displayName
    .split(" ")
    .map((w) => w[0])
    .filter(Boolean)
    .slice(0, 2)
    .join("")
    .toUpperCase();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label="Menu profil"
          className="rounded-full cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
        >
          <Avatar className="w-8 h-8 rounded-full">
            {user?.avatar_url ? <AvatarImage src={user.avatar_url} alt={displayName} /> : null}
            <AvatarFallback className="bg-primary text-primary-foreground text-xs font-bold">{initials}</AvatarFallback>
          </Avatar>
        </button>
      </DropdownMenuTrigger>

      <DropdownMenuContent
        align="end"
        className="w-56 p-1 border-2 rounded-md border-foreground shadow-[4px_4px_0px_0px] shadow-foreground bg-card"
      >
        <DropdownMenuLabel className="px-2 py-1.5">
          <span className="block text-sm font-black truncate">{displayName}</span>
          <span className="block text-xs font-bold text-muted-foreground truncate">{user?.email ?? ""}</span>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />

        <DropdownMenuItem asChild>
          <Link href="/user/profile" className="flex items-center gap-2 cursor-pointer font-bold">
            <User className="w-4 h-4" /> Profil &amp; Pengaturan
          </Link>
        </DropdownMenuItem>
        <DropdownMenuItem asChild>
          <Link href="/user/subscription" className="flex items-center gap-2 cursor-pointer font-bold">
            <CreditCard className="w-4 h-4" /> Langganan &amp; Paket
          </Link>
        </DropdownMenuItem>

        <DropdownMenuSeparator />
        <DropdownMenuItem
          variant="destructive"
          onSelect={onLogout}
          className="flex items-center gap-2 cursor-pointer font-bold"
        >
          <LogOut className="w-4 h-4" /> Keluar
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
