'use client'

import * as React from 'react'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Button } from '@/components/ui/button'
import { DataTableToolbar } from '@/components/data-table/data-table-toolbar'
import type { AdminPlanOption } from '@/hooks/useSuperAdminSubscriptions'

interface SubscriptionTableToolbarProps {
  search: string
  onSearch: (value: string) => void
  status: string
  onStatusChange: (status: string) => void
  planCode: string
  onPlanCodeChange: (planCode: string) => void
  plans: AdminPlanOption[]
  filters: Record<string, string>
  onRemoveFilter: (key: string) => void
  onReset: () => void
}

const ALL_STATUS_VALUE = '__all_status__'
const ALL_PLAN_VALUE = '__all_plan__'

export function SubscriptionTableToolbar({
  search,
  onSearch,
  status,
  onStatusChange,
  planCode,
  onPlanCodeChange,
  plans,
  filters,
  onRemoveFilter,
  onReset,
}: SubscriptionTableToolbarProps) {
  const hasActiveFilters = Boolean(search || status || planCode || Object.keys(filters).length > 0)

  return (
    <DataTableToolbar
      search={search}
      onSearch={onSearch}
      placeholder="Cari email atau nama pengguna..."
      filters={filters}
      onRemoveFilter={onRemoveFilter}
    >
      <div className="flex flex-wrap items-center gap-2">
        <Select
          value={status || ALL_STATUS_VALUE}
          onValueChange={(val) => onStatusChange(val === ALL_STATUS_VALUE ? '' : val)}
        >
          <SelectTrigger className="w-[140px] h-9">
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL_STATUS_VALUE}>Semua Status</SelectItem>
            <SelectItem value="active">Aktif</SelectItem>
            <SelectItem value="past_due">Menunggak</SelectItem>
            <SelectItem value="canceled">Dibatalkan</SelectItem>
            <SelectItem value="expired">Kedaluwarsa</SelectItem>
          </SelectContent>
        </Select>

        <Select
          value={planCode || ALL_PLAN_VALUE}
          onValueChange={(val) => onPlanCodeChange(val === ALL_PLAN_VALUE ? '' : val)}
        >
          <SelectTrigger className="w-[160px] h-9">
            <SelectValue placeholder="Semua Paket" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL_PLAN_VALUE}>Semua Paket</SelectItem>
            {plans.map((p) => (
              <SelectItem key={p.id} value={p.code}>
                {p.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        {hasActiveFilters && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onReset}
            className="h-9 px-2.5 text-xs text-muted-foreground hover:text-foreground"
          >
            Reset
          </Button>
        )}
      </div>
    </DataTableToolbar>
  )
}
