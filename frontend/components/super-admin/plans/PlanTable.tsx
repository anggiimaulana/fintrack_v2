'use client'

import { useState, useMemo } from 'react'
import {
  Search,
  Plus,
  Edit2,
  Trash2,
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
  Layers,
  Gauge,
  Package,
  CheckCircle2,
  Globe,
  Lock,
} from 'lucide-react'
import { AdminStatCard } from '@/components/admin/AdminStatCard'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { Money } from '@/components/shared/Money'
import { EmptyState } from '@/components/shared/EmptyState'
import { PlanDTO } from '@/hooks/useSuperAdminPlans'

type SortField =
  | 'code'
  | 'name'
  | 'price_monthly_idr'
  | 'price_yearly_idr'
  | 'billing_period_days'
  | 'sort_order'
  | 'is_active'

interface PlanTableProps {
  plans: PlanDTO[]
  isLoading: boolean
  onEdit: (plan: PlanDTO) => void
  onDeactivate: (plan: PlanDTO) => void
  onCreate: () => void
}

export function PlanTable({
  plans,
  isLoading,
  onEdit,
  onDeactivate,
  onCreate,
}: PlanTableProps) {
  const [search, setSearch] = useState('')
  const [sortField, setSortField] = useState<SortField>('sort_order')
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('asc')

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortOrder('asc')
    }
  }

  const renderSortIcon = (field: SortField) => {
    if (sortField !== field) {
      return <ArrowUpDown className="ml-1 h-3.5 w-3.5 text-muted-foreground/60 inline" />
    }
    return sortOrder === 'asc' ? (
      <ArrowUp className="ml-1 h-3.5 w-3.5 text-foreground inline" />
    ) : (
      <ArrowDown className="ml-1 h-3.5 w-3.5 text-foreground inline" />
    )
  }

  const filteredAndSortedPlans = useMemo(() => {
    let result = [...plans]

    const q = search.trim().toLowerCase()
    if (q) {
      result = result.filter(
        (p) =>
          p.code.toLowerCase().includes(q) ||
          p.name.toLowerCase().includes(q) ||
          (p.description && p.description.toLowerCase().includes(q))
      )
    }

    result.sort((a, b) => {
      let valA: string | number | boolean = a[sortField] ?? ''
      let valB: string | number | boolean = b[sortField] ?? ''

      if (typeof valA === 'string') {
        valA = valA.toLowerCase()
        valB = (valB as string).toLowerCase()
      }

      if (valA < valB) return sortOrder === 'asc' ? -1 : 1
      if (valA > valB) return sortOrder === 'asc' ? 1 : -1
      return 0
    })

    return result
  }, [plans, search, sortField, sortOrder])
  const totalPlans = plans.length
  const activePlans = plans.filter((p) => p.is_active).length
  const publicPlans = plans.filter((p) => p.is_public).length
  const privatePlans = plans.filter((p) => !p.is_public).length


  return (
    <div className="space-y-4">
      {/* 4 KPI Stats Cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <AdminStatCard label="Total Paket" value={totalPlans} icon={Package} />
        <AdminStatCard label="Paket Aktif" value={activePlans} icon={CheckCircle2} colorClassName="text-green-600" />
        <AdminStatCard label="Paket Publik" value={publicPlans} icon={Globe} colorClassName="text-primary" />
        <AdminStatCard label="Paket Privat" value={privatePlans} icon={Lock} colorClassName="text-amber-600" />
      </div>

      {/* Toolbar */}
      <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Cari nama atau kode paket..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
        <Button onClick={onCreate} className="shrink-0">
          <Plus className="h-4 w-4 mr-2" />
          Buat Paket Baru
        </Button>
      </div>

      {/* Table view */}
      {isLoading ? (
        <div className="rounded-md border border-border bg-card">
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent bg-muted/40">
                <TableHead className="w-12 text-center font-bold">No</TableHead>
                <TableHead>Paket & Kode</TableHead>
                <TableHead>Harga Bulanan</TableHead>
                <TableHead>Harga Tahunan</TableHead>
                <TableHead>Periode</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Gateway</TableHead>
                <TableHead>Urutan</TableHead>
                <TableHead>Fitur / Batas</TableHead>
                <TableHead className="text-right">Aksi</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {[1, 2, 3, 4].map((i) => (
                <TableRow key={i}>
                  <TableCell><Skeleton className="h-4 w-6 mx-auto" /></TableCell>
                  <TableCell>
                    <Skeleton className="h-5 w-28 mb-1" />
                    <Skeleton className="h-3 w-16" />
                  </TableCell>
                  <TableCell><Skeleton className="h-5 w-24" /></TableCell>
                  <TableCell><Skeleton className="h-5 w-24" /></TableCell>
                  <TableCell><Skeleton className="h-5 w-16" /></TableCell>
                  <TableCell><Skeleton className="h-5 w-20" /></TableCell>
                  <TableCell><Skeleton className="h-5 w-16" /></TableCell>
                  <TableCell><Skeleton className="h-5 w-10" /></TableCell>
                  <TableCell><Skeleton className="h-5 w-20" /></TableCell>
                  <TableCell className="text-right">
                    <Skeleton className="h-8 w-16 ml-auto" />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : filteredAndSortedPlans.length === 0 ? (
        <EmptyState
          title={search ? 'Paket Tidak Ditemukan' : 'Belum Ada Paket'}
          description={
            search
              ? 'Tidak ada paket langganan yang cocok dengan kata kunci pencarian.'
              : 'Belum ada paket langganan yang dibuat dalam sistem.'
          }
          action={
            !search ? (
              <Button onClick={onCreate} className="mt-2">
                <Plus className="h-4 w-4 mr-2" />
                Buat Paket Pertama
              </Button>
            ) : undefined
          }
        />
      ) : (
        <div className="rounded-md border border-border bg-card">
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent bg-muted/40">
                <TableHead className="w-12 text-center font-bold">No</TableHead>
                <TableHead
                  className="cursor-pointer select-none"
                  onClick={() => handleSort('name')}
                >
                  Paket & Kode {renderSortIcon('name')}
                </TableHead>
                <TableHead
                  className="cursor-pointer select-none"
                  onClick={() => handleSort('price_monthly_idr')}
                >
                  Harga Bulanan {renderSortIcon('price_monthly_idr')}
                </TableHead>
                <TableHead
                  className="cursor-pointer select-none"
                  onClick={() => handleSort('price_yearly_idr')}
                >
                  Harga Tahunan {renderSortIcon('price_yearly_idr')}
                </TableHead>
                <TableHead
                  className="cursor-pointer select-none"
                  onClick={() => handleSort('billing_period_days')}
                >
                  Periode {renderSortIcon('billing_period_days')}
                </TableHead>
                <TableHead
                  className="cursor-pointer select-none"
                  onClick={() => handleSort('is_active')}
                >
                  Status {renderSortIcon('is_active')}
                </TableHead>
                <TableHead>Gateway</TableHead>
                <TableHead
                  className="cursor-pointer select-none"
                  onClick={() => handleSort('sort_order')}
                >
                  Urutan {renderSortIcon('sort_order')}
                </TableHead>
                <TableHead>Fitur / Batas</TableHead>
                <TableHead className="text-right">Aksi</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredAndSortedPlans.map((plan) => {
                const entitlementCount = plan.entitlements
                  ? Object.keys(plan.entitlements).length
                  : 0
                const limitCount = plan.limits ? Object.keys(plan.limits).length : 0

                return (
                  <TableRow key={plan.id}>
                    <TableCell className="text-center font-mono text-xs text-muted-foreground w-12">
                      {filteredAndSortedPlans.indexOf(plan) + 1}
                    </TableCell>
                    <TableCell>
                      <div className="font-bold text-foreground">{plan.name}</div>
                      <div className="font-mono text-xs text-muted-foreground">
                        {plan.code}
                      </div>
                      {plan.description && (
                        <div className="text-xs text-muted-foreground truncate max-w-xs mt-0.5">
                          {plan.description}
                        </div>
                      )}
                    </TableCell>
                    <TableCell>
                      <Money value={plan.price_monthly_idr} />
                      <span className="text-xs text-muted-foreground">/bln</span>
                    </TableCell>
                    <TableCell>
                      <Money value={plan.price_yearly_idr} />
                      <span className="text-xs text-muted-foreground">/thn</span>
                    </TableCell>
                    <TableCell>
                      <span className="font-mono text-xs">
                        {plan.billing_period_days} hari
                      </span>
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-col gap-1 items-start">
                        {plan.is_active ? (
                          <Badge variant="outline" className="border-emerald-500 text-emerald-600 bg-emerald-50 dark:bg-emerald-950/40">
                            Aktif
                          </Badge>
                        ) : (
                          <Badge variant="outline" className="border-muted-foreground text-muted-foreground bg-muted">
                            Nonaktif
                          </Badge>
                        )}
                        {plan.is_public ? (
                          <Badge variant="secondary" className="text-[10px] px-1.5 py-0">
                            Publik
                          </Badge>
                        ) : (
                          <Badge variant="outline" className="text-[10px] px-1.5 py-0 text-muted-foreground">
                            Privat
                          </Badge>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <span className="font-mono text-xs uppercase bg-muted px-1.5 py-0.5 rounded">
                        {plan.fallback_gateway || 'midtrans'}
                      </span>
                    </TableCell>
                    <TableCell>
                      <span className="font-mono text-xs font-semibold">
                        {plan.sort_order}
                      </span>
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-col gap-1 text-xs">
                        <span className="flex items-center text-muted-foreground" title={`${entitlementCount} hak akses fitur`}>
                          <Layers className="h-3 w-3 mr-1 inline" />
                          {entitlementCount} Fitur
                        </span>
                        <span className="flex items-center text-muted-foreground" title={`${limitCount} batas kuota`}>
                          <Gauge className="h-3 w-3 mr-1 inline" />
                          {limitCount} Limit
                        </span>
                      </div>
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => onEdit(plan)}
                          aria-label={`Edit paket ${plan.name}`}
                        >
                          <Edit2 className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => onDeactivate(plan)}
                          disabled={!plan.is_active}
                          className="text-destructive hover:text-destructive hover:bg-destructive/10"
                          aria-label={`Nonaktifkan paket ${plan.name}`}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  )
}
