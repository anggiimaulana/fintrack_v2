'use client'

import * as React from 'react'
import { Users, UserCheck, ShieldAlert } from 'lucide-react'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { AdminStatCard } from '@/components/admin/AdminStatCard'
import { UserTable, type UserRow } from '@/components/admin/UserTable'
import { ConfirmActionModal } from '@/components/shared/ConfirmActionModal'
import { useAdminUsers } from '@/hooks/useAdminUsers'
import { useDataTableUrlState } from '@/hooks/use-data-table-url-state'

function UserManagementContent() {
  const tableState = useDataTableUrlState({
    sort: 'created_at',
    order: 'desc',
    perPage: 20,
  })

  const { users, meta, isLoading, toggleUserStatus, isMutating } = useAdminUsers({
    page: tableState.debounced.page,
    per_page: tableState.debounced.perPage,
    sort: tableState.debounced.sort,
    order: tableState.debounced.order,
    q: tableState.debounced.q,
    status: tableState.debounced.filters.status,
  })

  const [confirmDialog, setConfirmDialog] = React.useState<{
    open: boolean
    user: UserRow | null
  }>({
    open: false,
    user: null,
  })

  const handleToggleStatus = async () => {
    if (!confirmDialog.user) return
    try {
      await toggleUserStatus({
        id: confirmDialog.user.id,
        is_active: !confirmDialog.user.is_active,
      })
      setConfirmDialog({ open: false, user: null })
    } catch (error) {
      console.error('Failed to toggle user status:', error)
    }
  }

  const currentStatusFilter = tableState.immediate.filters.status || 'all'

  // KPI Stats
  const activeCount = users.filter((u: UserRow) => u.is_active).length
  const suspendedCount = users.filter((u: UserRow) => !u.is_active).length

  const statsSection = (
    <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
      <AdminStatCard
        label="Total Pengguna"
        value={meta.total}
        icon={Users}
        colorClassName="text-primary"
      />
      <AdminStatCard
        label="Pengguna Aktif (Hal Ini)"
        value={activeCount}
        icon={UserCheck}
        colorClassName="text-emerald-500"
      />
      <AdminStatCard
        label="Suspended (Hal Ini)"
        value={suspendedCount}
        icon={ShieldAlert}
        colorClassName="text-destructive"
      />
    </div>
  )

  const filterSection = (
    <div className="flex items-center gap-2">
      <Select
        value={currentStatusFilter}
        onValueChange={(val) => tableState.setFilter('status', val === 'all' ? '' : val)}
      >
        <SelectTrigger className="h-9 w-36 font-bold cursor-pointer">
          <SelectValue placeholder="Status Akun" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">Semua Status</SelectItem>
          <SelectItem value="active">Hanya Aktif</SelectItem>
          <SelectItem value="suspended">Hanya Suspended</SelectItem>
        </SelectContent>
      </Select>
    </div>
  )

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-black uppercase tracking-wide text-foreground">
          Manajemen Pengguna
        </h1>
        <p className="text-xs font-bold text-muted-foreground">
          Pantau status akun, penomoran urut, dan kontrol akses pengguna FinTrack
        </p>
      </div>

      <UserTable
        users={users}
        total={meta.total}
        totalPages={meta.total_pages}
        tableState={tableState}
        isLoading={isLoading}
        stats={statsSection}
        filters={filterSection}
        onToggleStatus={(user) => setConfirmDialog({ open: true, user })}
      />

      <ConfirmActionModal
        open={confirmDialog.open}
        onOpenChange={(open) => setConfirmDialog((prev) => ({ ...prev, open }))}
        actionType={confirmDialog.user?.is_active ? 'suspend' : 'deactivate'}
        title={confirmDialog.user?.is_active ? 'Suspend Akun Pengguna' : 'Aktifkan Akun Pengguna'}
        description={
          confirmDialog.user?.is_active
            ? 'Apakah Anda yakin ingin menangguhkan (suspend) akun pengguna ini?'
            : 'Apakah Anda yakin ingin mengaktifkan kembali akses akun pengguna ini?'
        }
        entityName={
          confirmDialog.user
            ? `${confirmDialog.user.full_name} (${confirmDialog.user.email})`
            : undefined
        }
        confirmLabel={confirmDialog.user?.is_active ? 'Ya, Suspend' : 'Ya, Aktifkan'}
        isPending={isMutating}
        onConfirm={handleToggleStatus}
      />
    </div>
  )
}

export default function UserManagementPage() {
  return (
    <React.Suspense fallback={<div className="h-40 animate-pulse bg-muted border-2 rounded-md border-foreground" />}>
      <UserManagementContent />
    </React.Suspense>
  )
}
