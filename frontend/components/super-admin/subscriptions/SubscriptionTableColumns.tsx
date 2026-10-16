'use client'

import * as React from 'react'
import {
  MoreHorizontal,
  CalendarPlus,
  ArrowDownCircle,
  XCircle,
} from 'lucide-react'
import type { ColumnDef } from '@tanstack/react-table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { DataTableColumnHeader } from '@/components/data-table/data-table-column-header'
import { formatTanggal } from '@/lib/format'
import type { SortOrder } from '@/lib/table-params'
import type { AdminSubscription } from '@/hooks/useSuperAdminSubscriptions'

interface SubscriptionTableActionsProps {
  onExtend: (sub: AdminSubscription) => void
  onDowngrade: (sub: AdminSubscription) => void
  onCancel: (sub: AdminSubscription) => void
}

function renderStatusBadge(status: string) {
  switch (status.toLowerCase()) {
    case 'active':
      return <Badge className="bg-emerald-500/15 text-emerald-700 dark:text-emerald-400 border-emerald-500/30">Aktif</Badge>
    case 'past_due':
      return <Badge className="bg-amber-500/15 text-amber-700 dark:text-amber-400 border-amber-500/30">Menunggak</Badge>
    case 'canceled':
      return <Badge className="bg-rose-500/15 text-rose-700 dark:text-rose-400 border-rose-500/30">Dibatalkan</Badge>
    case 'expired':
      return <Badge variant="secondary">Kedaluwarsa</Badge>
    default:
      return <Badge variant="outline">{status}</Badge>
  }
}

export function createSubscriptionColumns(
  actions: SubscriptionTableActionsProps,
  sortState?: { sort?: string; order?: SortOrder; onSort?: (col: string, order: SortOrder) => void },
  pagination?: { page: number; perPage: number }
): ColumnDef<AdminSubscription, unknown>[] {
  const { sort, order, onSort } = sortState || {}
  const page = pagination?.page || 1
  const perPage = pagination?.perPage || 20

  return [
    {
      id: 'row_number',
      header: () => <span className="text-center font-bold block w-10">No</span>,
      cell: ({ row }) => {
        const num = (page - 1) * perPage + row.index + 1
        return <span className="font-mono text-xs text-muted-foreground block text-center w-10">{num}</span>
      },
    },
    {
      accessorKey: 'user',
      id: 'user',
      header: () => (
        <DataTableColumnHeader
          title="Pengguna"
          columnId="user"
          sortable
          currentSort={sort}
          currentOrder={order}
          onSort={onSort}
        />
      ),
      cell: ({ row }) => {
        const sub = row.original
        return (
          <div className="flex flex-col min-w-[180px]">
            <span className="font-semibold text-foreground text-sm truncate">
              {sub.user_full_name || '-'}
            </span>
            <span className="text-xs text-muted-foreground truncate">
              {sub.user_email}
            </span>
          </div>
        )
      },
    },
    {
      accessorKey: 'plan',
      id: 'plan',
      header: () => (
        <DataTableColumnHeader
          title="Paket"
          columnId="plan"
          sortable
          currentSort={sort}
          currentOrder={order}
          onSort={onSort}
        />
      ),
      cell: ({ row }) => {
        const sub = row.original
        return (
          <div className="flex flex-col">
            <span className="font-medium text-foreground text-sm">
              {sub.plan_name || sub.plan_code}
            </span>
            <span className="text-xs text-muted-foreground uppercase">
              {sub.plan_code} • {sub.gateway}
            </span>
          </div>
        )
      },
    },
    {
      accessorKey: 'status',
      id: 'status',
      header: () => (
        <DataTableColumnHeader
          title="Status"
          columnId="status"
          sortable
          currentSort={sort}
          currentOrder={order}
          onSort={onSort}
        />
      ),
      cell: ({ row }) => {
        const sub = row.original
        return (
          <div className="flex flex-col items-start gap-1">
            {renderStatusBadge(sub.status)}
            {sub.cancel_at_period_end && (
              <Badge variant="outline" className="text-[10px] py-0 px-1.5 border-dashed border-amber-500/50 text-amber-600 dark:text-amber-400">
                Berhenti di akhir periode
              </Badge>
            )}
          </div>
        )
      },
    },
    {
      accessorKey: 'ends_at',
      id: 'ends_at',
      header: () => (
        <DataTableColumnHeader
          title="Periode Berjalan"
          columnId="ends_at"
          sortable
          currentSort={sort}
          currentOrder={order}
          onSort={onSort}
        />
      ),
      cell: ({ row }) => {
        const sub = row.original
        return (
          <div className="text-xs text-muted-foreground flex flex-col whitespace-nowrap">
            <span>Mulai: {formatTanggal(sub.current_period_start)}</span>
            <span className="font-medium text-foreground">
              Akhir: {sub.current_period_end ? formatTanggal(sub.current_period_end) : 'Selamanya'}
            </span>
          </div>
        )
      },
    },
    {
      accessorKey: 'created_at',
      id: 'created_at',
      header: () => (
        <DataTableColumnHeader
          title="Dibuat"
          columnId="created_at"
          sortable
          currentSort={sort}
          currentOrder={order}
          onSort={onSort}
        />
      ),
      cell: ({ row }) => {
        return (
          <span className="text-xs text-muted-foreground whitespace-nowrap">
            {formatTanggal(row.original.created_at)}
          </span>
        )
      },
    },
    {
      id: 'actions',
      header: () => <span className="text-xs font-semibold uppercase">Aksi</span>,
      cell: ({ row }) => {
        const sub = row.original
        return (
          <div className="text-right">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  className="h-8 w-8 p-0"
                  aria-label={`Aksi langganan ${sub.user_email}`}
                >
                  <MoreHorizontal className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-52">
                <DropdownMenuLabel>Aksi Langganan</DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={() => actions.onExtend(sub)}
                  className="cursor-pointer"
                >
                  <CalendarPlus className="mr-2 h-4 w-4 text-emerald-600 dark:text-emerald-400" />
                  <span>Perpanjang Hari</span>
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => actions.onDowngrade(sub)}
                  className="cursor-pointer"
                >
                  <ArrowDownCircle className="mr-2 h-4 w-4 text-amber-600 dark:text-amber-400" />
                  <span>Paksa Turunkan Paket</span>
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={() => actions.onCancel(sub)}
                  className="cursor-pointer text-destructive focus:text-destructive"
                >
                  <XCircle className="mr-2 h-4 w-4" />
                  <span>Batalkan Langganan</span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        )
      },
    },
  ]
}
