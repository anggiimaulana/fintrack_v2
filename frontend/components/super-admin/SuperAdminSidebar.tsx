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
  Wallet,
  LogOut,
  Tag,
  Activity,
  SlidersHorizontal,
  Calculator,
  ListChecks,
  Gauge,
  ChevronDown,
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
import { useHash } from '@/hooks/use-hash'
import { useSidebarCollapsed } from '@/hooks/use-sidebar-collapsed'
import { cn } from '@/lib/utils'

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
  const { collapsed: isRailCollapsed } = useSidebarCollapsed()

  const [dialogOpen, setDialogOpen] = React.useState(false)

  // Which groups are collapsed, keyed by label. All groups start OPEN (false),
  // and a group is never auto-closed: the operator decides.
  const [collapsedGroups, setCollapsedGroups] = React.useState<Record<string, boolean>>({})
  const toggleGroup = (label: string) => {
    setCollapsedGroups((prev) => ({ ...prev, [label]: !prev[label] }))
  }

  const handleSignOut = async () => {
    await logout()
    router.push('/login')
    router.refresh()
  }

  // Two entries can share a pathname (Pengaturan vs. Estimasi Token AI, which
  // deep-links to a tab). The active one is decided by the hash.
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
    const claimedByHash = superAdminMenuItems.some(
      (m) => m.href.startsWith(`${path}#`) && m.href.split('#')[1] === currentHash,
    )
    return !claimedByHash
  }

  const displayName = user?.full_name || 'Super Admin'
  const initials = displayName
    .split(' ')
    .map((w) => w[0])
    .filter(Boolean)
    .slice(0, 2)
    .join('')
    .toUpperCase()

  return (
    <aside
      data-collapsed={isRailCollapsed}
      className={cn(
        "hidden md:flex flex-col bg-sidebar text-sidebar-foreground h-screen sticky top-0 border-r-2 border-foreground shrink-0",
        "transition-[width] duration-300 ease-in-out",
        isRailCollapsed ? "w-[76px]" : "w-[240px]"
      )}
    >
      {/* Brand Header */}
      <div className={cn("border-b-2 border-sidebar-border flex items-center", isRailCollapsed ? "p-3 justify-center" : "p-4")}>
        <Link href="/super-admin" className="flex items-center gap-2 text-xl font-black tracking-tight overflow-hidden">
          <Wallet className="w-6 h-6 text-primary shrink-0" />
          <div
            className={cn(
              "flex flex-col transition-all duration-200",
              isRailCollapsed ? "w-0 opacity-0 overflow-hidden" : "w-auto opacity-100"
            )}
          >
            <span className="whitespace-nowrap leading-none">
              Fin<span className="text-primary">Track</span>
            </span>
            <span className="text-[10px] text-sidebar-foreground/60 font-black uppercase tracking-widest mt-1">
              Super Admin
            </span>
          </div>
        </Link>
      </div>

      {/* Nav Groups */}
      <nav className="flex-1 px-3 py-3 space-y-3 overflow-y-auto overflow-x-hidden no-scrollbar">
        {superAdminMenuGroups.map((group) => {
          const isOpen = !collapsedGroups[group.label]
          const groupHasActive = group.items.some((item) => isItemActive(item.href, item.exact))

          return (
            <div key={group.label} className="space-y-1">
              {!isRailCollapsed && (
                <button
                  type="button"
                  onClick={() => toggleGroup(group.label)}
                  aria-expanded={isOpen}
                  className="cursor-pointer flex w-full items-center justify-between gap-2 px-3 py-1 text-xs font-black uppercase tracking-widest text-sidebar-foreground/60 hover:text-sidebar-foreground transition-colors"
                >
                  <span className="flex items-center gap-2 truncate">
                    {group.label}
                    {groupHasActive && (
                      <span className="h-1.5 w-1.5 rounded-full bg-primary shrink-0" aria-hidden="true" />
                    )}
                  </span>
                  <ChevronDown
                    size={14}
                    className={cn("shrink-0 transition-transform duration-200", isOpen ? "rotate-0" : "-rotate-90")}
                  />
                </button>
              )}

              {(isRailCollapsed || isOpen) && (
                <div className="space-y-1">
                  {group.items.map((item) => {
                    const active = isItemActive(item.href, item.exact)
                    return (
                      <Link
                        key={item.href}
                        href={item.href}
                        title={isRailCollapsed ? item.title : undefined}
                        className={cn(
                          "flex items-center gap-3 px-3 py-2 text-sm font-bold transition-all rounded-md",
                          isRailCollapsed ? "justify-center" : "",
                          active
                            ? "bg-primary text-primary-foreground border-2 border-foreground shadow-[2px_2px_0px_0px] shadow-foreground"
                            : "text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-foreground"
                        )}
                      >
                        <item.icon className="w-5 h-5 shrink-0" />
                        <span
                          className={cn(
                            "truncate transition-all duration-200",
                            isRailCollapsed ? "w-0 opacity-0 overflow-hidden hidden" : "w-auto opacity-100 inline-block"
                          )}
                        >
                          {item.title}
                        </span>
                      </Link>
                    )
                  })}
                </div>
              )}
            </div>
          )
        })}
      </nav>

      {/* Footer Profile & Logout */}
      <div className={cn("border-t-2 border-sidebar-border", isRailCollapsed ? "p-3" : "p-4")}>
        <div className={cn("flex items-center gap-3 mb-3", isRailCollapsed ? "justify-center" : "px-1")}>
          <Avatar className="h-8 w-8 rounded-full shrink-0">
            <AvatarImage src={user?.avatar_url ?? undefined} />
            <AvatarFallback className="bg-primary text-primary-foreground font-bold text-xs">
              {initials || 'SA'}
            </AvatarFallback>
          </Avatar>
          {!isRailCollapsed && (
            <div className="flex flex-col min-w-0">
              <span className="text-sm font-black truncate text-sidebar-foreground">
                {displayName}
              </span>
              <span className="text-xs font-bold text-sidebar-foreground/60 truncate">
                {user?.email || 'superadmin@fintrack.id'}
              </span>
            </div>
          )}
        </div>

        <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
          <DialogTrigger asChild>
            <button
              type="button"
              title={isRailCollapsed ? "Keluar" : undefined}
              className={cn(
                "cursor-pointer flex items-center gap-3 w-full py-2 text-sm font-bold text-sidebar-foreground/70 hover:text-red-400 hover:bg-sidebar-accent transition-all rounded-md",
                isRailCollapsed ? "justify-center px-0" : "px-3"
              )}
            >
              <LogOut className="w-4 h-4 shrink-0" />
              {!isRailCollapsed && <span>Keluar</span>}
            </button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-sm">
            <DialogHeader>
              <DialogTitle>Konfirmasi Logout</DialogTitle>
              <DialogDescription>
                Apakah Anda yakin ingin keluar dari Super Admin Panel?
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
