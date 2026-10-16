'use client'

import * as React from 'react'
import * as Icons from 'lucide-react'
import { LucideIcon, Pencil, Trash2, ArrowUpDown } from 'lucide-react'
import { type ColumnDef } from '@tanstack/react-table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { DataTableLayout } from '@/components/data-table/DataTableLayout'
import type { UseDataTableUrlState } from '@/hooks/use-data-table-url-state'
import Link from 'next/link'

export interface Category {
  id: string
  name: string
  icon: string
  color: string
  is_active: boolean
  description?: string
  sort_order?: number
}

export interface CategoryTableProps {
  categories: Category[]
  onEdit: (category: Category) => void
  onDelete: (id: string) => void
  isLoading?: boolean
  type?: 'income' | 'expense' | 'savings'
  tableState: UseDataTableUrlState
  stats?: React.ReactNode
  filters?: React.ReactNode
}

export function CategoryTable({
  categories,
  onEdit,
  onDelete,
  isLoading = false,
  type = 'income',
  tableState,
  stats,
  filters,
}: CategoryTableProps) {
  const filteredAndSorted = React.useMemo(() => {
    let result = [...categories]
    const q = (tableState.immediate.q || '').trim().toLowerCase()
    if (q) {
      result = result.filter(
        (c) =>
          c.name.toLowerCase().includes(q) ||
          (c.description && c.description.toLowerCase().includes(q))
      )
    }
    const status = tableState.immediate.filters.status
    if (status === 'active') {
      result = result.filter((c) => c.is_active)
    } else if (status === 'inactive') {
      result = result.filter((c) => !c.is_active)
    }

    const sortCol = tableState.immediate.sort
    const isAsc = tableState.immediate.order === 'asc'
    if (sortCol === 'name') {
      result.sort((a, b) =>
        isAsc ? a.name.localeCompare(b.name) : b.name.localeCompare(a.name)
      )
    } else if (sortCol === 'status') {
      result.sort((a, b) => {
        const aVal = a.is_active ? 1 : 0
        const bVal = b.is_active ? 1 : 0
        return isAsc ? aVal - bVal : bVal - aVal
      })
    }
    return result
  }, [
    categories,
    tableState.immediate.q,
    tableState.immediate.filters.status,
    tableState.immediate.sort,
    tableState.immediate.order,
  ])

  const total = filteredAndSorted.length
  const totalPages = Math.max(1, Math.ceil(total / tableState.perPage))

  const paginatedData = React.useMemo(() => {
    const start = (tableState.page - 1) * tableState.perPage
    return filteredAndSorted.slice(start, start + tableState.perPage)
  }, [filteredAndSorted, tableState.page, tableState.perPage])

  const columns = React.useMemo<ColumnDef<Category>[]>(() => {
    const cols: ColumnDef<Category>[] = [
      {
        id: 'row_number',
        header: () => <span className="text-center font-bold block w-10">No</span>,
        cell: ({ row }) => {
          const num = (tableState.page - 1) * tableState.perPage + row.index + 1
          return <span className="font-mono text-xs text-muted-foreground block text-center w-10">{num}</span>
        },
      },
      {
        accessorKey: 'name',
        header: () => (
          <button
            type="button"
            onClick={() => tableState.setSort('name')}
            className="flex items-center gap-1.5 font-bold hover:text-foreground cursor-pointer"
          >
            Nama Kategori
            <ArrowUpDown className="w-3.5 h-3.5" />
          </button>
        ),
        cell: ({ row }) => {
          const cat = row.original
          const Icon = Icons[cat.icon as keyof typeof Icons] as LucideIcon | undefined
          return (
            <div className="flex items-center gap-2.5">
              <div
                className="flex items-center justify-center h-8 w-8 rounded-md border-2 border-foreground"
                style={{ backgroundColor: `${cat.color}20`, color: cat.color }}
              >
                {Icon ? <Icon size={18} /> : <Icons.Tag size={18} />}
              </div>
              <div>
                <span className="font-bold text-foreground block">{cat.name}</span>
                {cat.description ? (
                  <span className="text-xs text-muted-foreground">{cat.description}</span>
                ) : null}
              </div>
            </div>
          )
        },
      },
      {
        accessorKey: 'color',
        header: () => <span className="font-bold">Warna</span>,
        cell: ({ row }) => (
          <div className="flex items-center gap-2">
            <div
              className="h-4 w-4 rounded-full border-2 border-foreground"
              style={{ backgroundColor: row.original.color }}
            />
            <span className="text-xs font-mono font-bold text-muted-foreground">{row.original.color}</span>
          </div>
        ),
      },
      {
        accessorKey: 'is_active',
        header: () => (
          <button
            type="button"
            onClick={() => tableState.setSort('status')}
            className="flex items-center gap-1.5 font-bold hover:text-foreground cursor-pointer"
          >
            Status
            <ArrowUpDown className="w-3.5 h-3.5" />
          </button>
        ),
        cell: ({ row }) => (
          <Badge
            variant={row.original.is_active ? 'default' : 'secondary'}
            className={row.original.is_active ? "bg-emerald-600 text-white" : "bg-muted text-muted-foreground"}
          >
            {row.original.is_active ? 'Aktif' : 'Nonaktif'}
          </Badge>
        ),
      },
    ]

    if (type === 'expense') {
      cols.push({
        id: 'subcategories',
        header: () => <span className="font-bold">Sub-kategori</span>,
        cell: ({ row }) => (
          <Button variant="outline" size="sm" asChild className="h-8 text-xs font-bold border-2 border-foreground cursor-pointer">
            <Link href={`/admin/categories/expense/${row.original.id}/subcategories`}>
              Kelola
            </Link>
          </Button>
        ),
      })
    }

    cols.push({
      id: 'actions',
      header: () => <span className="font-bold block text-right pr-2">Aksi</span>,
      cell: ({ row }) => (
        <div className="flex justify-end gap-1.5 pr-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => onEdit(row.original)}
            className="h-8 px-2.5 text-xs font-bold border-2 border-foreground cursor-pointer"
          >
            <Pencil className="w-3.5 h-3.5 mr-1" />
            Edit
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => onDelete(row.original.id)}
            className="h-8 px-2.5 text-xs font-bold border-2 border-destructive text-destructive hover:bg-destructive/10 cursor-pointer"
          >
            <Trash2 className="w-3.5 h-3.5 mr-1" />
            Hapus
          </Button>
        </div>
      ),
    })

    return cols
  }, [type, onEdit, onDelete, tableState])

  return (
    <DataTableLayout
      stats={stats}
      filters={filters}
      columns={columns}
      data={paginatedData}
      isLoading={isLoading}
      total={total}
      totalPages={totalPages}
      tableState={tableState}
      searchPlaceholder="Cari kategori..."
      emptyMessage="Belum ada kategori ditemukan."
    />
  )
}
