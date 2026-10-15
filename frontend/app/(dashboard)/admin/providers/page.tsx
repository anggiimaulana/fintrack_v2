'use client'
import { useState } from 'react'
import { Plus, Search, Pencil, Power, PowerOff } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '@/lib/api'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

interface ProviderItem {
  id: string;
  name: string;
  supports_fast?: boolean;
  supports_reasoning?: boolean;
  display_name: string;
  base_url: string;
  is_active: boolean;
}

export default function AdminProvidersPage() {
  const queryClient = useQueryClient()
  const [isDialogOpen, setIsDialogOpen] = useState(false)
  const [editingProvider, setEditingProvider] = useState<ProviderItem | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [formName, setFormName] = useState('')
  const [formDisplayName, setFormDisplayName] = useState('')
  const [formSupportsFast, setFormSupportsFast] = useState(true)
  const [formSupportsReasoning, setFormSupportsReasoning] = useState(false)

  const { data: providers, isLoading } = useQuery({
    queryKey: ['admin', 'providers'],
    queryFn: async () => {
      try { const res = await api.get('/providers'); return res.data?.data || []; }
      catch { return []; }
    },
  })

  const createMutation = useMutation({
    mutationFn: async (data: Partial<ProviderItem>) => { const res = await api.post('/admin/providers', data); return res.data; },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['admin', 'providers'] }); },
  })

  const updateMutation = useMutation({
    mutationFn: async ({ id, ...data }: { id: string } & Partial<ProviderItem>) => { const res = await api.put(`/admin/providers/${id}`, data); return res.data; },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['admin', 'providers'] }); },
  })

  const isMutating = createMutation.isPending || updateMutation.isPending

  const filteredProviders = providers?.filter((p: ProviderItem) =>
    p.name?.toLowerCase().includes(searchQuery.toLowerCase()) ||
    p.display_name?.toLowerCase().includes(searchQuery.toLowerCase())
  ) || []

  const handleOpenAdd = () => {
    setEditingProvider(null)
    setFormName('')
    setFormDisplayName('')
    setFormSupportsFast(true)
    setFormSupportsReasoning(false)
    setIsDialogOpen(true)
  }

  const handleOpenEdit = (provider: ProviderItem) => {
    setEditingProvider(provider)
    setFormName(provider.name)
    setFormDisplayName(provider.display_name)
    setFormSupportsFast(provider.supports_fast || false)
    setFormSupportsReasoning(provider.supports_reasoning || false)
    setIsDialogOpen(true)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      if (editingProvider) {
        await updateMutation.mutateAsync({
          id: editingProvider.id,
          name: formName,
          display_name: formDisplayName,
          supports_fast: formSupportsFast,
          supports_reasoning: formSupportsReasoning,
        })
      } else {
        await createMutation.mutateAsync({
          name: formName,
          display_name: formDisplayName,
          supports_fast: formSupportsFast,
          supports_reasoning: formSupportsReasoning,
        })
      }
      setIsDialogOpen(false)
    } catch (error) {
      console.error('Failed to save provider:', error)
    }
  }

  const handleToggleActive = async (provider: ProviderItem) => {
    try {
      await updateMutation.mutateAsync({
        id: provider.id,
        is_active: !provider.is_active,
      })
    } catch (error) {
      console.error('Failed to toggle provider:', error)
    }
  }

  return (
    <div className="space-y-8">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-xl font-black uppercase tracking-wide text-foreground">Provider AI</h1>
          <p className="text-xs font-bold text-muted-foreground">Kelola provider AI yang tersedia untuk user</p>
        </div>
        <Button onClick={handleOpenAdd}>
          <Plus className="mr-2 h-4 w-4" /> Tambah Provider
        </Button>
      </div>

      <div className="flex items-center gap-2 max-w-md">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={18} />
          <Input
            placeholder="Cari provider..."
            className="pl-10"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </div>
      </div>

      {isLoading ? (
        <div className="space-y-4">
          <Skeleton className="h-[300px] w-full border-2 rounded-md border-foreground" />
        </div>
      ) : (
        <div className="border-2 rounded-md border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b-2 border-foreground bg-muted">
                <th className="text-left px-4 py-3 font-black uppercase tracking-wide text-xs">Nama</th>
                <th className="text-left px-4 py-3 font-black uppercase tracking-wide text-xs">Display Name</th>
                <th className="text-center px-4 py-3 font-black uppercase tracking-wide text-xs">Fast</th>
                <th className="text-center px-4 py-3 font-black uppercase tracking-wide text-xs">Reasoning</th>
                <th className="text-center px-4 py-3 font-black uppercase tracking-wide text-xs">Status</th>
                <th className="text-center px-4 py-3 font-black uppercase tracking-wide text-xs">Aksi</th>
              </tr>
            </thead>
            <tbody>
              {filteredProviders.map((p: ProviderItem) => (
                <tr key={p.id} className="border-b border-foreground/20 hover:bg-muted/50 transition-colors">
                  <td className="px-4 py-3 font-bold">{p.name}</td>
                  <td className="px-4 py-3">{p.display_name}</td>
                  <td className="px-4 py-3 text-center">
                    {p.supports_fast ? <span className="text-green-600 font-bold">Ya</span> : <span className="text-muted-foreground">-</span>}
                  </td>
                  <td className="px-4 py-3 text-center">
                    {p.supports_reasoning ? <span className="text-green-600 font-bold">Ya</span> : <span className="text-muted-foreground">-</span>}
                  </td>
                  <td className="px-4 py-3 text-center">
                    <span className={`inline-flex items-center gap-1 px-2 py-0.5 text-xs font-bold border rounded-md ${
                      p.is_active ? 'bg-green-100 text-green-700 border-green-300' : 'bg-gray-100 text-gray-500 border-gray-300'
                    }`}>
                      {p.is_active ? 'Aktif' : 'Nonaktif'}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-center">
                    <div className="flex items-center justify-center gap-2">
                      <Button variant="outline" size="sm" onClick={() => handleOpenEdit(p)} className="cursor-pointer">
                        <Pencil className="w-3.5 h-3.5" />
                      </Button>
                      <Button variant="outline" size="sm" onClick={() => handleToggleActive(p)} className="cursor-pointer">
                        {p.is_active ? <PowerOff className="w-3.5 h-3.5 text-red-500" /> : <Power className="w-3.5 h-3.5 text-green-500" />}
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
              {filteredProviders.length === 0 && (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-sm font-bold text-muted-foreground">
                    Tidak ada provider ditemukan.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}

      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent className="sm:max-w-[425px] border-2 rounded-md border-foreground">
          <DialogHeader>
            <DialogTitle className="font-black uppercase tracking-wide">
              {editingProvider ? 'Edit Provider' : 'Tambah Provider'}
            </DialogTitle>
            <DialogDescription className="font-medium">
              Isi detail provider di bawah ini.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <label htmlFor="provider-name" className="text-xs font-bold uppercase tracking-wide">Nama</label>
              <Input id="provider-name" value={formName} onChange={(e) => setFormName(e.target.value)} placeholder="gemini" required disabled={!!editingProvider} />
            </div>
            <div className="space-y-2">
              <label htmlFor="provider-display-name" className="text-xs font-bold uppercase tracking-wide">Display Name</label>
              <Input id="provider-display-name" value={formDisplayName} onChange={(e) => setFormDisplayName(e.target.value)} placeholder="Google Gemini" required />
            </div>
            <div className="flex items-center gap-6">
              <label className="flex items-center gap-2 cursor-pointer">
                <input type="checkbox" checked={formSupportsFast} onChange={(e) => setFormSupportsFast(e.target.checked)} className="w-4 h-4" />
                <span className="text-xs font-bold uppercase tracking-wide">Fast</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input type="checkbox" checked={formSupportsReasoning} onChange={(e) => setFormSupportsReasoning(e.target.checked)} className="w-4 h-4" />
                <span className="text-xs font-bold uppercase tracking-wide">Reasoning</span>
              </label>
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <Button type="button" variant="outline" onClick={() => setIsDialogOpen(false)} className="cursor-pointer">Batal</Button>
              <Button type="submit" disabled={isMutating} className="cursor-pointer">
                {isMutating ? 'Menyimpan...' : 'Simpan'}
              </Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
