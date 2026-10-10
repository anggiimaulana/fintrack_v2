'use client'

import { useState } from 'react'
import { Plus, Search } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { CategoryTable } from '@/components/admin/CategoryTable'
import { CategoryForm } from '@/components/admin/CategoryForm'
import { useAdminCategories, AdminCategory } from '@/hooks/useAdminCategories'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'

export default function SavingsCategoriesPage() {
  const { 
    categories, 
    isLoading, 
    createCategory, 
    updateCategory, 
    deleteCategory,
    isMutating 
  } = useAdminCategories('savings')

  const [isDialogOpen, setIsDialogOpen] = useState(false)
  const [editingCategory, setEditingCategory] = useState<AdminCategory | null>(null)
  const [searchQuery, setSearchQuery] = useState('')

  const filteredCategories = categories?.filter((c: AdminCategory) => 
    c.name.toLowerCase().includes(searchQuery.toLowerCase())
  ) || []

  const handleOpenAdd = () => {
    setEditingCategory(null)
    setIsDialogOpen(true)
  }

  const handleOpenEdit = (category: AdminCategory) => {
    setEditingCategory(category)
    setIsDialogOpen(true)
  }

  const handleSubmit = async (values: Partial<AdminCategory>) => {
    try {
      if (editingCategory) {
        await updateCategory({ id: editingCategory.id, ...values })
      } else {
        await createCategory(values)
      }
      setIsDialogOpen(false)
    } catch (error) {
      console.error('Failed to save category:', error)
    }
  }

  const handleDelete = async (id: string) => {
    if (confirm('Apakah Anda yakin ingin menghapus kategori ini?')) {
      try {
        await deleteCategory(id)
      } catch (error) {
        console.error('Failed to delete category:', error)
      }
    }
  }

  return (
    <div className="space-y-8">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-xl font-black uppercase tracking-wide text-foreground">Kategori Tabungan</h1>
          <p className="text-xs font-bold text-muted-foreground">Kelola kategori untuk transaksi tabungan</p>
        </div>
        <Button onClick={handleOpenAdd}>
          <Plus className="mr-2 h-4 w-4" /> Tambah Kategori
        </Button>
      </div>

      <div className="flex items-center gap-2 max-w-md">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
          <Input
            placeholder="Cari kategori..."
            className="pl-10"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </div>
      </div>

      {isLoading ? (
        <div className="space-y-4">
          <Skeleton className="h-[400px] w-full border-2 border-foreground" />
        </div>
      ) : (
        <CategoryTable
          categories={filteredCategories}
          onEdit={handleOpenEdit}
          onDelete={handleDelete}
        />
      )}

      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent className="sm:max-w-[425px] border-2 border-foreground">
          <DialogHeader>
            <DialogTitle className="font-black uppercase tracking-wide">{editingCategory ? 'Edit Kategori' : 'Tambah Kategori'}</DialogTitle>
            <DialogDescription className="font-medium">
              Isi detail kategori di bawah ini. Klik simpan setelah selesai.
            </DialogDescription>
          </DialogHeader>
          <CategoryForm
            initialValues={editingCategory || undefined}
            onSubmit={handleSubmit}
            onCancel={() => setIsDialogOpen(false)}
            isLoading={isMutating}
          />
        </DialogContent>
      </Dialog>
    </div>
  )
}
