'use client'

import * as React from 'react'
import { Plus, ArrowUpCircle, CheckCircle2, XCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { AdminStatCard } from '@/components/admin/AdminStatCard'
import { CategoryTable } from '@/components/admin/CategoryTable'
import { CategoryForm } from '@/components/admin/CategoryForm'
import { useAdminCategories, AdminCategory } from '@/hooks/useAdminCategories'
import { useDataTableUrlState } from '@/hooks/use-data-table-url-state'
import { ConfirmActionModal } from '@/components/shared/ConfirmActionModal'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

function IncomeCategoriesContent() {
  const {
    categories,
    isLoading,
    createCategory,
    updateCategory,
    deleteCategory,
    isMutating,
  } = useAdminCategories('income')

  const tableState = useDataTableUrlState({ perPage: 20 })
  const [isDialogOpen, setIsDialogOpen] = React.useState(false)
  const [editingCategory, setEditingCategory] = React.useState<AdminCategory | null>(null)
  const [deleteId, setDeleteId] = React.useState<string | null>(null)

  const handleSubmit = async (values: Partial<AdminCategory>) => {
    if (editingCategory) {
      await updateCategory({ id: editingCategory.id, ...values })
    } else {
      await createCategory(values)
    }
    setIsDialogOpen(false)
  }

  const deleteTarget = categories?.find((c: AdminCategory) => c.id === deleteId)

  const handleConfirmDelete = async () => {
    if (!deleteId) return
    await deleteCategory(deleteId)
    setDeleteId(null)
  }

  const totalCount = categories?.length || 0
  const activeCount = categories?.filter((c) => c.is_active).length || 0
  const inactiveCount = categories?.filter((c) => !c.is_active).length || 0

  const statsSection = (
    <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
      <AdminStatCard label="Total Kategori" value={totalCount} icon={ArrowUpCircle} />
      <AdminStatCard label="Kategori Aktif" value={activeCount} icon={CheckCircle2} colorClassName="text-green-600" />
      <AdminStatCard label="Kategori Nonaktif" value={inactiveCount} icon={XCircle} colorClassName="text-rose-600" />
    </div>
  )

  const currentStatus = tableState.immediate.filters.status || 'all'
  const filterSection = (
    <div className="flex items-center gap-2">
      <Select
        value={currentStatus}
        onValueChange={(val) => tableState.setFilter('status', val === 'all' ? '' : val)}
      >
        <SelectTrigger className="w-[140px] h-9 text-xs font-bold border-2 border-foreground">
          <SelectValue placeholder="Status" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">Semua Status</SelectItem>
          <SelectItem value="active">Aktif</SelectItem>
          <SelectItem value="inactive">Nonaktif</SelectItem>
        </SelectContent>
      </Select>
    </div>
  )

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-xl font-black uppercase tracking-wide text-foreground">
            Kategori Pemasukan
          </h1>
          <p className="text-xs font-bold text-muted-foreground">
            Kelola master kategori pemasukan sistem
          </p>
        </div>
        <Button
          onClick={() => {
            setEditingCategory(null)
            setIsDialogOpen(true)
          }}
          className="font-bold border-2 border-foreground shadow-[2px_2px_0px_0px] shadow-foreground cursor-pointer"
        >
          <Plus className="mr-2 h-4 w-4" /> Tambah Kategori
        </Button>
      </div>

      <CategoryTable
        categories={categories || []}
        onEdit={(cat) => {
          setEditingCategory(cat)
          setIsDialogOpen(true)
        }}
        onDelete={(id) => setDeleteId(id)}
        isLoading={isLoading}
        type="income"
        tableState={tableState}
        stats={statsSection}
        filters={filterSection}
      />

      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent className="sm:max-w-[425px] border-2 rounded-md border-foreground">
          <DialogHeader>
            <DialogTitle className="font-black uppercase tracking-wide">
              {editingCategory ? 'Edit Kategori Pemasukan' : 'Tambah Kategori Pemasukan'}
            </DialogTitle>
            <DialogDescription className="font-medium text-xs">
              Isi detail kategori pemasukan di bawah ini.
            </DialogDescription>
          </DialogHeader>
          <CategoryForm
            initialValues={editingCategory || undefined}
            onSubmit={handleSubmit}
            isLoading={isMutating}
            onCancel={() => setIsDialogOpen(false)}
          />
        </DialogContent>
      </Dialog>

      <ConfirmActionModal
        open={Boolean(deleteId)}
        onOpenChange={(open) => {
          if (!open) setDeleteId(null)
        }}
        actionType="delete"
        title="Hapus Kategori Pemasukan"
        description="Apakah Anda yakin ingin menghapus kategori pemasukan ini? Tindakan ini tidak dapat dibatalkan."
        entityName={deleteTarget?.name}
        confirmLabel="Ya, Hapus"
        isPending={isMutating}
        onConfirm={handleConfirmDelete}
      />
    </div>
  )
}

export default function IncomeCategoriesPage() {
  return (
    <React.Suspense fallback={<div className="h-40 animate-pulse bg-muted border-2 rounded-md border-foreground" />}>
      <IncomeCategoriesContent />
    </React.Suspense>
  )
}
