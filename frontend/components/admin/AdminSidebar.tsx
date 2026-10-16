'use client'

import * as React from 'react'
import Link from 'next/link'
import { usePathname, useRouter } from 'next/navigation'
import {
  LayoutDashboard,
  ArrowUpCircle,
  ArrowDownCircle,
  Users,
  LogOut,
  Wallet,
  Bot,
  PiggyBank,
} from 'lucide-react'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { logout } from '@/lib/auth-client'
import { useSession } from '@/lib/session-context'
import { useSidebarCollapsed } from '@/hooks/use-sidebar-collapsed'
import { cn } from '@/lib/utils'

const adminMenuItems = [
  { title: 'Dashboard', href: '/admin', icon: LayoutDashboard },
  { title: 'Kategori Pemasukan', href: '/admin/categories/income', icon: ArrowUpCircle },
  { title: 'Kategori Pengeluaran', href: '/admin/categories/expense', icon: ArrowDownCircle },
  { title: 'Manajemen User', href: '/admin/users', icon: Users },
  { title: 'Kategori Tabungan', href: '/admin/categories/savings', icon: PiggyBank },
  { title: 'Provider AI', href: '/admin/providers', icon: Bot },
]

export function AdminSidebar() {
  const pathname = usePathname()
  const router = useRouter()
  const { user } = useSession()
  const { collapsed } = useSidebarCollapsed()
  const [dialogOpen, setDialogOpen] = React.useState(false)

  const handleSignOut = async () => {
    await logout()
    router.push('/login')
    router.refresh()
  }

  const displayName = user?.full_name || 'Administrator'
  const initials = displayName
    .split(' ')
    .map((w) => w[0])
    .filter(Boolean)
    .slice(0, 2)
    .join('')
    .toUpperCase()

  return (
    <aside
      data-collapsed={collapsed}
      className={cn(
        "hidden md:flex flex-col bg-sidebar text-sidebar-foreground h-screen sticky top-0 border-r-2 border-foreground shrink-0",
        "transition-[width] duration-300 ease-in-out",
        collapsed ? "w-[76px]" : "w-[240px]"
      )}
    >
      {/* Brand Header */}
      <div className={cn("border-b-2 border-sidebar-border flex items-center", collapsed ? "p-3 justify-center" : "p-4")}>
        <Link href="/admin" className="flex items-center gap-2 text-xl font-black tracking-tight overflow-hidden">
          <Wallet className="w-6 h-6 text-primary shrink-0" />
          <div
            className={cn(
              "flex flex-col transition-all duration-200",
              collapsed ? "w-0 opacity-0 overflow-hidden" : "w-auto opacity-100"
            )}
          >
            <span className="whitespace-nowrap leading-none">
              Fin<span className="text-primary">Track</span>
            </span>
            <span className="text-[10px] text-sidebar-foreground/60 font-black uppercase tracking-widest mt-1">
              Admin Panel
            </span>
          </div>
        </Link>
      </div>

      {/* Nav Content */}
      <nav className="flex-1 px-3 py-4 space-y-1 overflow-y-auto overflow-x-hidden no-scrollbar">
        {!collapsed && (
          <div className="text-sidebar-foreground/60 px-3 mb-2 text-xs font-black uppercase tracking-widest">
            Menu Utama
          </div>
        )}
        {adminMenuItems.map((item) => {
          const isActive = pathname === item.href || (item.href !== '/admin' && pathname.startsWith(item.href))
          return (
            <Link
              key={item.href}
              href={item.href}
              title={collapsed ? item.title : undefined}
              className={cn(
                "flex items-center gap-3 px-3 py-2.5 text-sm font-bold transition-all rounded-md",
                collapsed ? "justify-center" : "",
                isActive
                  ? "bg-primary text-primary-foreground border-2 border-foreground shadow-[2px_2px_0px_0px] shadow-foreground"
                  : "text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-foreground"
              )}
            >
              <item.icon className="w-5 h-5 shrink-0" />
              <span
                className={cn(
                  "truncate transition-all duration-200",
                  collapsed ? "w-0 opacity-0 overflow-hidden hidden" : "w-auto opacity-100 inline-block"
                )}
              >
                {item.title}
              </span>
            </Link>
          )
        })}
      </nav>

      {/* Footer Profile & Logout */}
      <div className={cn("border-t-2 border-sidebar-border", collapsed ? "p-3" : "p-4")}>
        <div className={cn("flex items-center gap-3 mb-3", collapsed ? "justify-center" : "px-1")}>
          <Avatar className="h-8 w-8 rounded-full shrink-0">
            <AvatarImage src={user?.avatar_url ?? undefined} />
            <AvatarFallback className="bg-primary text-primary-foreground font-bold text-xs">
              {initials || 'AD'}
            </AvatarFallback>
          </Avatar>
          {!collapsed && (
            <div className="flex flex-col min-w-0">
              <span className="text-sm font-black truncate text-sidebar-foreground">
                {displayName}
              </span>
              <span className="text-xs font-bold text-sidebar-foreground/60 truncate">
                {user?.email || 'admin@fintrack.id'}
              </span>
            </div>
          )}
        </div>

        <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
          <DialogTrigger asChild>
            <button
              type="button"
              title={collapsed ? "Keluar" : undefined}
              className={cn(
                "cursor-pointer flex items-center gap-3 w-full py-2 text-sm font-bold text-sidebar-foreground/70 hover:text-red-400 hover:bg-sidebar-accent transition-all rounded-md",
                collapsed ? "justify-center px-0" : "px-3"
              )}
            >
              <LogOut className="w-4 h-4 shrink-0" />
              {!collapsed && <span>Keluar</span>}
            </button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-sm">
            <DialogHeader>
              <DialogTitle>Konfirmasi Logout</DialogTitle>
              <DialogDescription>
                Apakah Anda yakin ingin keluar dari Admin Panel?
              </DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <Button variant="outline" onClick={() => setDialogOpen(false)}>
                Batal
              </Button>
              <Button variant="destructive" onClick={handleSignOut}>
                Keluar
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
    </aside>
  )
}
