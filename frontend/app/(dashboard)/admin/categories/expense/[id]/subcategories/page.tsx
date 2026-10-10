'use client'

import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, Pencil, Trash2, Loader2, ArrowLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { SubcategoryManager } from '@/components/admin/SubcategoryManager'
import api from '@/lib/api'
import Link from 'next/link'
import { useParams } from 'next/navigation'
import { Skeleton } from '@/components/ui/skeleton'

export default function SubcategoriesPage() {
  const { id: categoryId } = useParams()
  const queryClient = useQueryClient()
  
  const [isDialogOpen, setIsDialogOpen] = useState(false)
  const [editingSub, setEditingSub] = useState<any>(null)
  const [name, setName] = useState('')
  const [isActive, setIsActive] = useState(true)

  const { data: category } = useQuery({
    queryKey: ['admin-category', categoryId],
    queryFn: async () => {
      const response = await api.get(`/admin/categories/expense/${categoryId}`)
      return response.data
    },
  })

  const { data: subcategories, isLoading } = useQuery({
    queryKey: ['admin-subcategories', categoryId],
    queryFn: async () => {
      const response = await api.get(`/admin/subcategories?category_id=${categoryId}`)
      return response.data
    },
  })

  const createMutation = useMutation({
    mutationFn: async (data: any) => {
      const response = await api.post('/admin/subcategories', { ...data, category_id: categoryId })
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-subcategories', categoryId] })
      setIsDialogOpen(false)
      resetForm()
    },
  })

  const updateMutation = useMutation({
    mutationFn: async ({ id, ...data }: any) => {
      const response = await api.put(`/admin/subcategories/${id}`, data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-subcategories', categoryId] })
      setIsDialogOpen(false)
      resetForm()
    },
  })

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      const response = await api.delete(`/admin/subcategories/${id}`)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-subcategories', categoryId] })
    },
  })

  const resetForm = () => {
    setEditingSub(null)
    setName('')
    setIsActive(true)
  }

  const handleOpenAdd = () => {
    resetForm()
    setIsDialogOpen(true)
  }

  const handleOpenEdit = (sub: any) => {
    setEditingSub(sub)
    setName(sub.name)
    setIsActive(sub.is_active)
    setIsDialogOpen(true)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (editingSub) {
      await updateMutation.mutateAsync({ id: editingSub.id, name, is_active: isActive })
    } else {
      await createMutation.mutateAsync({ name, is_active: isActive })
    }
  }

  const handleDelete = async (id: string) => {
    if (confirm('Apakah Anda yakin ingin menghapus sub-kategori ini?')) {
      await deleteMutation.mutateAsync(id)
    }
  }

  return (
    <div className="space-y-8">
      <div className="flex flex-col gap-4">
        <Button variant="ghost" size="sm" asChild className="w-fit -ml-2 text-neutral-500">
          <Link href="/admin/categories/expense">
            <ArrowLeft className="mr-2 h-4 w-4" /> Kembali ke Kategori
          </Link>
        </Button>
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <div className="flex items-center gap-2 text-sm text-neutral-500 mb-1">
              <span>Kategori Pengeluaran</span>
              <span>/</span>
              <span className="font-medium text-neutral-900">{category?.name || '...'}</span>
            </div>
            <h1 className="text-3xl font-bold tracking-tight text-neutral-900">Sub-kategori</h1>
            <p className="text-neutral-600">Kelola rincian sub-kategori untuk {category?.name}</p>
          </div>
          <Button onClick={handleOpenAdd} className="bg-primary-500 hover:bg-primary-600">
            <Plus className="mr-2 h-4 w-4" /> Tambah Sub-kategori
          </Button>
        </div>
      </div>

      <SubcategoryManager 
        subcategories={subcategories || []}
        onEdit={handleOpenEdit}
        onDelete={handleDelete}
        isLoading={isLoading}
      />

      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent className="sm:max-w-[425px]">
          <form onSubmit={handleSubmit}>
            <DialogHeader>
              <DialogTitle>{editingSub ? 'Edit Sub-kategori' : 'Tambah Sub-kategori'}</DialogTitle>
              <DialogDescription>
                Masukkan nama sub-kategori untuk {category?.name}.
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">Nama Sub-kategori</label>
                <Input
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Contoh: Makan Siang, Bensin"
                  required
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Status</label>
                <Select
                  onValueChange={(val) => setIsActive(val === 'active')}
                  defaultValue={isActive ? 'active' : 'inactive'}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Pilih status" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="active">Aktif</SelectItem>
                    <SelectItem value="inactive">Nonaktif</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <DialogFooter>
              <Button type="button" variant="ghost" onClick={() => setIsDialogOpen(false)}>
                Batal
              </Button>
              <Button type="submit" className="bg-primary-500 hover:bg-primary-600" disabled={createMutation.isPending || updateMutation.isPending}>
                {(createMutation.isPending || updateMutation.isPending) && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Simpan
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
