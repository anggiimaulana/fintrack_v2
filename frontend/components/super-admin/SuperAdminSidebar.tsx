'use client'

import * as React from 'react'
import Link from 'next/link'
import { usePathname, useRouter } from 'next/navigation'
import {
  LayoutDashboard,
  Package,
  Repeat,
  CreditCard,
  FileCheck,
  Coins,
  Cpu,
  History,
  Settings,
  ShieldCheck,
  LogOut,
  Tag,
  Activity,
  SlidersHorizontal,
  Calculator,
  ListChecks,
  Gauge,
  ChevronDown,
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
import { useHash } from '@/hooks/use-hash'

// Sidebar menu, grouped for manageability. Every group is a collapsible
// section (all open by default). Ordering here is the display order.
const superAdminMenuGroups: {
  label: string
  items: { title: string; href: string; icon: React.ElementType; exact: boolean }[]
}[] = [
  {
    label: 'Ringkasan',
    items: [
      { title: 'Dashboard', href: '/super-admin', icon: LayoutDashboard, exact: true },
    ],
  },
  {
    label: 'Langganan & Pembayaran',
    items: [
      { title: 'Paket Langganan', href: '/super-admin/plans', icon: Package, exact: false },
      { title: 'Langganan', href: '/super-admin/subscriptions', icon: Repeat, exact: false },
      { title: 'Pembayaran', href: '/super-admin/payments', icon: CreditCard, exact: true },
      { title: 'Review Pembayaran', href: '/super-admin/payments/review', icon: FileCheck, exact: false },
    ],
  },
  {
    label: 'Token & AI',
    items: [
      { title: 'Token & Kuota', href: '/super-admin/tokens', icon: Coins, exact: false },
      { title: 'Pemantauan Kuota', href: '/super-admin/quota-monitor', icon: Activity, exact: false },
      { title: 'Harga & Biaya AI', href: '/super-admin/pricing', icon: Tag, exact: false },
      { title: 'Kalkulator Kuota', href: '/super-admin/kuota', icon: Calculator, exact: false },
      { title: 'Estimasi Token AI', href: '/super-admin/settings#kuota', icon: Gauge, exact: false },
      { title: 'Katalog Metrik', href: '/super-admin/metrics', icon: ListChecks, exact: false },
      { title: 'AI Gateway', href: '/super-admin/ai-gateway', icon: Cpu, exact: false },
    ],
  },
  {
    label: 'Sistem',
    items: [
      { title: 'Metrik SPK', href: '/super-admin/spk-metrics', icon: SlidersHorizontal, exact: false },
      { title: 'Audit Log', href: '/super-admin/audit', icon: History, exact: false },
      { title: 'Pengaturan', href: '/super-admin/settings', icon: Settings, exact: false },
    ],
  },
]

// Flat list (used for hash-aware active detection across groups).
const superAdminMenuItems = superAdminMenuGroups.flatMap((g) => g.items)

export function SuperAdminSidebar() {
  const pathname = usePathname()
  const router = useRouter()
  const { user } = useSession()

  const [dialogOpen, setDialogOpen] = React.useState(false)

  // Which groups are collapsed, keyed by label. All groups start OPEN (false),
  // and a group is never auto-closed: the operator decides.
  const [collapsed, setCollapsed] = React.useState<Record<string, boolean>>({})
  const toggleGroup = (label: string) => {
    setCollapsed((prev) => ({ ...prev, [label]: !prev[label] }))
  }

  const handleSignOut = async () => {
    await logout()
    router.push('/login')
    router.refresh()
  }

  // Two entries can share a pathname (Pengaturan vs. Estimasi Token AI, which
  // deep-links to a tab). The active one is decided by the hash: an entry with
  // a hash wins only when that hash is present, and an entry without one is
  // inactive while another entry claims the current hash.
  const currentHash = useHash()
  const isItemActive = (href: string, exact: boolean) => {
    const [path, hash] = href.split('#')
    if (hash) {
      return pathname === path && currentHash === hash
    }
    if (exact && pathname !== path) {
      return false
    }
    if (!exact && pathname !== path && !pathname.startsWith(`${path}/`)) {
      return false
    }
    // The bare path is inactive when a hashed sibling owns the current hash.
    const claimedByHash = superAdminMenuItems.some(
      (m) => m.href.startsWith(`${path}#`) && m.href.split('#')[1] === currentHash,
    )
    return !claimedByHash
  }

  return (
    <Sidebar className="border-r-2 border-foreground bg-sidebar text-sidebar-foreground">
      <SidebarHeader className="p-6 border-b-2 border-sidebar-border">
        <div className="flex items-center gap-2 text-xl font-black tracking-tight">
          <ShieldCheck className="w-6 h-6 text-primary" />
          Fin<span className="text-primary">Track</span>
        </div>
        <div className="text-[10px] text-sidebar-foreground/60 mt-1 font-black uppercase tracking-widest">
          Super Admin
        </div>
      </SidebarHeader>
      <SidebarContent>
        {superAdminMenuGroups.map((group) => {
          const isOpen = !collapsed[group.label]
          const groupHasActive = group.items.some((item) => isItemActive(item.href, item.exact))
          return (
            <SidebarGroup key={group.label} className="py-1">
              <SidebarGroupLabel asChild>
                <button
                  type="button"
                  onClick={() => toggleGroup(group.label)}
                  aria-expanded={isOpen}
                  className="cursor-pointer flex w-full items-center justify-between gap-2 px-4 mb-1 text-xs font-black uppercase tracking-widest text-sidebar-foreground/60 hover:text-sidebar-foreground transition-colors"
                >
                  <span className="flex items-center gap-2 truncate">
                    {group.label}
                    {groupHasActive && (
                      <span
                        className="h-1.5 w-1.5 rounded-full bg-primary shrink-0"
                        aria-hidden="true"
                      />
                    )}
                  </span>
                  <ChevronDown
                    size={14}
                    className={`shrink-0 transition-transform duration-200 ${
                      isOpen ? 'rotate-0' : '-rotate-90'
                    }`}
                  />
                </button>
              </SidebarGroupLabel>
              {isOpen && (
                <SidebarGroupContent>
                  <SidebarMenu>
                    {group.items.map((item) => {
                      const active = isItemActive(item.href, item.exact)
                      return (
                        <SidebarMenuItem key={item.href}>
                          <SidebarMenuButton
                            asChild
                            isActive={active}
                            className={`flex items-center gap-3 px-4 py-3 text-sm font-bold transition-all ${
                              active
                                ? 'bg-primary text-primary-foreground border-2 rounded-md border-foreground shadow-[2px_2px_0px_0px] shadow-foreground'
                                : 'text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-foreground'
                            }`}
                          >
                            <Link href={item.href}>
                              <item.icon size={20} />
                              <span>{item.title}</span>
                            </Link>
                          </SidebarMenuButton>
                        </SidebarMenuItem>
                      )
                    })}
                  </SidebarMenu>
                </SidebarGroupContent>
              )}
            </SidebarGroup>
          )
        })}
      </SidebarContent>
      <SidebarFooter className="p-4 border-t-2 border-sidebar-border">
        <div className="flex items-center gap-3 mb-4 px-2">
          <Avatar className="h-9 w-9 border-2 rounded-md border-foreground">
            <AvatarImage src={user?.avatar_url ?? undefined} />
            <AvatarFallback className="bg-primary text-primary-foreground font-bold">
              {user?.full_name?.substring(0, 2).toUpperCase() || 'SA'}
            </AvatarFallback>
          </Avatar>
          <div className="flex flex-col min-w-0">
            <span className="text-sm font-black truncate text-sidebar-foreground">
              {user?.full_name || 'Super Admin'}
            </span>
            <span className="text-xs font-bold text-sidebar-foreground/60 truncate">
              {user?.email || 'superadmin@fintrack.id'}
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
