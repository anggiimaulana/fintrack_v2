'use client'

import * as React from 'react'
import { format } from 'date-fns'
import { id as idLocale } from 'date-fns/locale'
import { UserX, UserCheck, ArrowUpDown } from 'lucide-react'
import { type ColumnDef } from '@tanstack/react-table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { DataTableLayout } from '@/components/data-table/DataTableLayout'
import type { UseDataTableUrlState } from '@/hooks/use-data-table-url-state'

export interface UserRow {
  id: string
  full_name: string
  email: string
  role?: string
  created_at: string
  is_active: boolean
  total_transactions?: number
}

interface UserTableProps {
  users: UserRow[]
  total: number
  totalPages: number
  tableState: UseDataTableUrlState
  onToggleStatus: (user: UserRow) => void
  isLoading?: boolean
  stats?: React.ReactNode
  filters?: React.ReactNode
}

export function UserTable({
  users,
  total,
  totalPages,
  tableState,
  onToggleStatus,
  isLoading,
  stats,
  filters,
}: UserTableProps) {
  const columns = React.useMemo<ColumnDef<UserRow>[]>(
    () => [
      {
        id: 'row_number',
        header: () => <span className="text-center font-bold block w-10">No</span>,
        cell: ({ row }) => {
          const num = (tableState.page - 1) * tableState.perPage + row.index + 1
          return <span className="font-mono text-xs text-muted-foreground block text-center w-10">{num}</span>
        },
      },
      {
        accessorKey: 'full_name',
        header: () => (
          <button
            type="button"
            onClick={() => tableState.setSort('full_name')}
            className="flex items-center gap-1.5 font-bold hover:text-foreground cursor-pointer"
          >
            Nama Lengkap
            <ArrowUpDown className="w-3.5 h-3.5" />
          </button>
        ),
        cell: ({ row }) => <span className="font-bold text-foreground">{row.original.full_name}</span>,
      },
      {
        accessorKey: 'email',
        header: () => (
          <button
            type="button"
            onClick={() => tableState.setSort('email')}
            className="flex items-center gap-1.5 font-bold hover:text-foreground cursor-pointer"
          >
            Email
            <ArrowUpDown className="w-3.5 h-3.5" />
          </button>
        ),
        cell: ({ row }) => (
          <span
            title={row.original.email}
            className="font-mono text-xs text-muted-foreground truncate max-w-[200px] inline-block cursor-default"
          >
            {row.original.email}
          </span>
        ),
      },
      {
        accessorKey: 'total_transactions',
        header: () => (
          <button
            type="button"
            onClick={() => tableState.setSort('total_transactions')}
            className="flex items-center gap-1.5 font-bold hover:text-foreground cursor-pointer"
          >
            Total Transaksi
            <ArrowUpDown className="w-3.5 h-3.5" />
          </button>
        ),
        cell: ({ row }) => (
          <span className="font-mono text-xs font-bold text-foreground">
            {row.original.total_transactions ?? 0}
          </span>
        ),
      },
      {
        accessorKey: 'is_active',
        header: () => <span className="font-bold">Status</span>,
        cell: ({ row }) => (
          <Badge
            variant="outline"
            className={
              row.original.is_active
                ? 'bg-emerald-500/10 text-emerald-600 border-emerald-500/30 font-bold'
                : 'bg-destructive/10 text-destructive border-destructive/30 font-bold'
            }
          >
            {row.original.is_active ? 'Aktif' : 'Suspended'}
          </Badge>
        ),
      },
      {
        accessorKey: 'created_at',
        header: () => (
          <button
            type="button"
            onClick={() => tableState.setSort('created_at')}
            className="flex items-center gap-1.5 font-bold hover:text-foreground cursor-pointer"
          >
            Terdaftar
            <ArrowUpDown className="w-3.5 h-3.5" />
          </button>
        ),
        cell: ({ row }) => {
          try {
            return (
              <span className="text-xs text-muted-foreground">
                {format(new Date(row.original.created_at), 'd MMM yyyy', { locale: idLocale })}
              </span>
            )
          } catch {
            return <span className="text-xs text-muted-foreground">-</span>
          }
        },
      },
      {
        id: 'actions',
        header: () => <span className="font-bold text-right block">Aksi</span>,
        cell: ({ row }) => {
          const user = row.original
          return (
            <div className="flex justify-end">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => onToggleStatus(user)}
                className={`cursor-pointer h-8 px-2.5 text-xs font-bold ${
                  user.is_active
                    ? 'text-destructive hover:bg-destructive/10 hover:text-destructive border-destructive/30'
                    : 'text-emerald-600 hover:bg-emerald-500/10 hover:text-emerald-600 border-emerald-500/30'
                }`}
              >
                {user.is_active ? (
                  <>
                    <UserX className="w-3.5 h-3.5 mr-1" />
                    Suspend
                  </>
                ) : (
                  <>
                    <UserCheck className="w-3.5 h-3.5 mr-1" />
                    Aktifkan
                  </>
                )}
              </Button>
            </div>
          )
        },
      },
    ],
    [onToggleStatus, tableState]
  )

  return (
    <DataTableLayout
      stats={stats}
      columns={columns}
      data={users}
      total={total}
      totalPages={totalPages}
      tableState={tableState}
      isLoading={isLoading}
      searchPlaceholder="Cari nama atau email pengguna..."
      filters={filters}
    />
  )
}
