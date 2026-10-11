'use client'

import * as React from 'react'
import { ShieldAlert } from 'lucide-react'
import { toast } from 'sonner'
import { useDataTableUrlState } from '@/hooks/use-data-table-url-state'
import {
  useSuperAdminAudit,
  type AuditLogItem,
  type AuditLogFilters,
} from '@/hooks/useSuperAdminAudit'
import { AuditFilters } from '@/components/super-admin/audit/AuditFilters'
import { AuditTable } from '@/components/super-admin/audit/AuditTable'
import { AuditDetailDrawer } from '@/components/super-admin/audit/AuditDetailDrawer'

function SuperAdminAuditContent() {
  const tableState = useDataTableUrlState({
    sort: 'created_at',
    order: 'desc',
    perPage: 20,
  })

  // Selected audit log item for detail drawer
  const [selectedLog, setSelectedLog] = React.useState<AuditLogItem | null>(null)
  const [drawerOpen, setDrawerOpen] = React.useState(false)

  // Extract filters from tableState (filter_<name> prefix or search q)
  // Mapping URL state filter keys to contract §7 query params:
  const startDate = tableState.filters['start_date'] || ''
  const endDate = tableState.filters['end_date'] || ''
  const email = tableState.filters['email'] || ''
  const userId = tableState.filters['user_id'] || ''
  const entity = tableState.filters['entity'] || ''
  const action = tableState.filters['action'] || ''
  const entityId = tableState.filters['entity_id'] || ''

  const queryFilters: AuditLogFilters = React.useMemo(
    () => ({
      page: tableState.page,
      per_page: tableState.perPage,
      sort: tableState.sort,
      order: tableState.order,
      start_date: startDate || undefined,
      end_date: endDate || undefined,
      email: email || undefined,
      user_id: userId || undefined,
      entity: entity || undefined,
      action: action || undefined,
      entity_id: entityId || undefined,
    }),
    [
      tableState.page,
      tableState.perPage,
      tableState.sort,
      tableState.order,
      startDate,
      endDate,
      email,
      userId,
      entity,
      action,
      entityId,
    ]
  )

  const { data, isLoading, error } = useSuperAdminAudit(queryFilters)

  // Surface backend 400 validation for invalid UUID or other errors inline & toast
  const axiosError = error as { response?: { status?: number; data?: { message?: string; error?: string } } } | null
  const backendErrorMsg = axiosError?.response?.data?.message || null
  const backendErrorCode = axiosError?.response?.data?.error || null

  const userIdError =
    backendErrorCode === 'validation' && userId
      ? backendErrorMsg || 'User ID tidak valid (harus UUID)'
      : null

  React.useEffect(() => {
    if (backendErrorMsg && !userIdError) {
      toast.error(backendErrorMsg)
    }
  }, [backendErrorMsg, userIdError])

  const handleSelectRow = (log: AuditLogItem) => {
    setSelectedLog(log)
    setDrawerOpen(true)
  }

  const handleDrawerClose = (open: boolean) => {
    setDrawerOpen(open)
    if (!open) {
      setTimeout(() => setSelectedLog(null), 200)
    }
  }

  return (
    <div className="space-y-6 p-6">
      {/* Header section */}
      <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-2xl font-black tracking-tight text-foreground">
            Audit Trail
          </h1>
          <p className="text-sm font-medium text-muted-foreground">
            Riwayat log aktivitas sistem dan mutasi administratif yang bersifat kekal (append-only).
          </p>
        </div>

        <div className="inline-flex items-center gap-2 rounded-md border border-border bg-muted/50 px-3 py-1.5 text-xs font-semibold text-muted-foreground self-start md:self-auto">
          <ShieldAlert className="h-4 w-4 text-primary" />
          <span>Append-Only Trail (UU PDP)</span>
        </div>
      </div>

      {/* Filter controls */}
      <AuditFilters
        startDate={startDate}
        endDate={endDate}
        email={email}
        userId={userId}
        entity={entity}
        action={action}
        entityId={entityId}
        userIdError={userIdError}
        onFilterChange={tableState.setFilter}
        onReset={tableState.reset}
      />

      {/* Audit table */}
      <AuditTable
        data={data?.data || []}
        isLoading={isLoading}
        total={data?.meta?.total || 0}
        page={tableState.page}
        perPage={tableState.perPage}
        sort={tableState.sort}
        order={tableState.order}
        onPageChange={tableState.setPage}
        onPerPageChange={tableState.setPerPage}
        onSort={tableState.setSort}
        onSelectRow={handleSelectRow}
      />

      {/* Detail drawer with diff view */}
      <AuditDetailDrawer
        log={selectedLog}
        open={drawerOpen}
        onOpenChange={handleDrawerClose}
      />
    </div>
  )
}

// useDataTableUrlState reads useSearchParams, so it must sit under a Suspense
// boundary for the route to stay statically prerenderable (see Next.js
// missing-suspense-with-csr-bailout).
export default function SuperAdminAuditPage() {
  return (
    <React.Suspense fallback={<div className="h-40 animate-pulse bg-muted border-2 border-foreground" />}>
      <SuperAdminAuditContent />
    </React.Suspense>
  )
}
