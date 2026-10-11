'use client'

import { useState } from 'react'
import { toast } from 'sonner'
import { PlanTable } from '@/components/super-admin/plans/PlanTable'
import { PlanFormDialog } from '@/components/super-admin/plans/PlanFormDialog'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import {
  useSuperAdminPlans,
  PlanDTO,
  PlanPayload,
} from '@/hooks/useSuperAdminPlans'

interface AxiosErrorShape {
  response?: {
    status?: number
    data?: {
      code?: string
      message?: string
    }
  }
}

export default function SuperAdminPlansPage() {
  const {
    plans,
    isLoading,
    createPlan,
    isCreating,
    updatePlan,
    isUpdating,
    deactivatePlan,
    isDeactivating,
  } = useSuperAdminPlans()

  // Form dialog state
  const [isFormOpen, setIsFormOpen] = useState(false)
  const [editingPlan, setEditingPlan] = useState<PlanDTO | null>(null)
  const [codeConflictError, setCodeConflictError] = useState<string | null>(null)

  // Confirm deactivate dialog state
  const [deactivatingPlan, setDeactivatingPlan] = useState<PlanDTO | null>(null)

  const handleOpenCreate = () => {
    setEditingPlan(null)
    setCodeConflictError(null)
    setIsFormOpen(true)
  }

  const handleOpenEdit = (plan: PlanDTO) => {
    setEditingPlan(plan)
    setCodeConflictError(null)
    setIsFormOpen(true)
  }

  const handleFormSubmit = async (payload: PlanPayload) => {
    try {
      if (editingPlan) {
        await updatePlan({ id: editingPlan.id, payload })
        toast.success(`Paket "${payload.name}" berhasil diperbarui`)
      } else {
        await createPlan(payload)
        toast.success(`Paket "${payload.name}" berhasil dibuat`)
      }
      setIsFormOpen(false)
      setEditingPlan(null)
      setCodeConflictError(null)
    } catch (err) {
      const axiosErr = err as AxiosErrorShape
      const status = axiosErr.response?.status
      const msg =
        axiosErr.response?.data?.message || 'Gagal menyimpan perubahan paket'

      if (status === 409) {
        setCodeConflictError(msg)
        toast.error(msg)
      } else {
        toast.error(msg)
      }
    }
  }

  const handleConfirmDeactivate = async () => {
    if (!deactivatingPlan) return

    try {
      await deactivatePlan(deactivatingPlan.id)
      toast.success(`Paket "${deactivatingPlan.name}" berhasil dinonaktifkan`)
      setDeactivatingPlan(null)
    } catch (err) {
      const axiosErr = err as AxiosErrorShape
      const msg =
        axiosErr.response?.data?.message || 'Gagal menonaktifkan paket'
      toast.error(msg)
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-black uppercase tracking-wide text-foreground">
          Manajemen Paket Langganan
        </h1>
        <p className="text-xs font-bold text-muted-foreground">
          Atur skema paket, harga bulanan/tahunan, kuota fitur, dan hak akses FinTrack
        </p>
      </div>

      <PlanTable
        plans={plans}
        isLoading={isLoading}
        onCreate={handleOpenCreate}
        onEdit={handleOpenEdit}
        onDeactivate={(plan) => setDeactivatingPlan(plan)}
      />

      {isFormOpen && (
        <PlanFormDialog
          key={editingPlan ? editingPlan.id : 'new-plan'}
          open={isFormOpen}
          onOpenChange={(open) => {
            setIsFormOpen(open)
            if (!open) {
              setEditingPlan(null)
              setCodeConflictError(null)
            }
          }}
          plan={editingPlan}
          onSubmit={handleFormSubmit}
          isSubmitting={isCreating || isUpdating}
          codeConflictError={codeConflictError}
          onClearCodeConflict={() => setCodeConflictError(null)}
        />
      )}

      <ConfirmDialog
        open={Boolean(deactivatingPlan)}
        onOpenChange={(open) => {
          if (!open) setDeactivatingPlan(null)
        }}
        onConfirm={handleConfirmDeactivate}
        title="Nonaktifkan Paket"
        message={
          deactivatingPlan
            ? `Paket "${deactivatingPlan.name}" (${deactivatingPlan.code}) akan dinonaktifkan, bukan dihapus dari database. Pengguna baru tidak dapat berlangganan paket ini.`
            : ''
        }
        confirmText="Ya, Nonaktifkan"
        cancelText="Batal"
        variant="danger"
        loading={isDeactivating}
      />
    </div>
  )
}
