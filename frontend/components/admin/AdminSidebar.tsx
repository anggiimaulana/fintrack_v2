'use client'

import * as React from 'react'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
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
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarGroupContent,
} from '@/components/ui/sidebar'
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
import { useRouter } from 'next/navigation'

const adminMenuItems = [
  {
    title: 'Dashboard',
    href: '/admin',
    icon: LayoutDashboard,
  },
  {
    title: 'Kategori Pemasukan',
    href: '/admin/categories/income',
    icon: ArrowUpCircle,
  },
  {
    title: 'Kategori Pengeluaran',
    href: '/admin/categories/expense',
    icon: ArrowDownCircle,
  },
  {
    title: 'Manajemen User',
    href: '/admin/users',
    icon: Users,
  },
  {
    title: 'Kategori Tabungan',
    href: '/admin/categories/savings',
    icon: PiggyBank,
  },
  {
    title: 'Provider AI',
    href: '/admin/providers',
    icon: Bot,
  },
]

export function AdminSidebar() {
  const pathname = usePathname()
  const router = useRouter()
  const { user } = useSession()

  const [dialogOpen, setDialogOpen] = React.useState(false)

  const handleSignOut = async () => {
    await logout()
    router.push('/login')
    router.refresh()
  }

  return (
    <Sidebar className="border-r-2 border-foreground bg-sidebar text-sidebar-foreground">
      <SidebarHeader className="p-6 border-b-2 border-sidebar-border">
        <div className="flex items-center gap-2 text-xl font-black tracking-tight">
          <Wallet className="w-6 h-6 text-primary" />
          Fin<span className="text-primary">Track</span>
        </div>
        <div className="text-[10px] text-sidebar-foreground/60 mt-1 font-black uppercase tracking-widest">
          Admin Panel
        </div>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel className="text-sidebar-foreground/60 px-4 mb-2 text-xs font-black uppercase tracking-widest">
            Menu Utama
          </SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {adminMenuItems.map((item) => (
                <SidebarMenuItem key={item.href}>
                  <SidebarMenuButton
                    asChild
                    isActive={pathname === item.href}
                    className={`flex items-center gap-3 px-4 py-3 text-sm font-bold transition-all ${
                      pathname === item.href
                        ? 'bg-primary text-primary-foreground border-2 border-foreground shadow-[2px_2px_0px_0px] shadow-foreground'
                        : 'text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-foreground'
                    }`}
                  >
                    <Link href={item.href}>
                      <item.icon size={20} />
                      <span>{item.title}</span>
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter className="p-4 border-t-2 border-sidebar-border">
        <div className="flex items-center gap-3 mb-4 px-2">
          <Avatar className="h-9 w-9 border-2 border-foreground">
            <AvatarImage src={user?.avatar_url ?? undefined} />
            <AvatarFallback className="bg-primary text-primary-foreground font-bold">
              {user?.full_name?.substring(0, 2).toUpperCase() || 'AD'}
            </AvatarFallback>
          </Avatar>
          <div className="flex flex-col min-w-0">
            <span className="text-sm font-black truncate text-sidebar-foreground">
              {user?.full_name || 'Administrator'}
            </span>
            <span className="text-xs font-bold text-sidebar-foreground/60 truncate">
              {user?.email || 'admin@fintrack.id'}
            </span>
          </div>
        </div>
        <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
          <DialogTrigger asChild>
            <button
              className="cursor-pointer flex items-center gap-3 w-full px-4 py-2 text-sm font-bold text-sidebar-foreground/70 hover:text-red-400 hover:bg-sidebar-accent transition-all"
            >
              <LogOut size={18} />
              <span>Keluar</span>
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
                Keluar
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </SidebarFooter>
    </Sidebar>
  )
}
