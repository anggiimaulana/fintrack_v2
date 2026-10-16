'use client'

import * as React from 'react'
import { Plus, Pencil, Power, PowerOff, Cpu, Zap, Brain, CheckCircle2, ArrowUpDown, Activity } from 'lucide-react'
import { ConfirmActionModal } from '@/components/shared/ConfirmActionModal'
import { SmartInsightCard } from '@/components/shared/SmartInsightCard'
import { MultiSeriesChart } from '@/components/charts/MultiSeriesChart'
import { DataTableLayout } from '@/components/data-table/DataTableLayout'
import { AdminStatCard } from '@/components/admin/AdminStatCard'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useDataTableUrlState } from '@/hooks/use-data-table-url-state'
import { type ColumnDef } from '@tanstack/react-table'
import api from '@/lib/api'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

interface ProviderItem {
  id: string
  name: string
  supports_fast?: boolean
  supports_reasoning?: boolean
  display_name: string
  base_url?: string
  is_active: boolean
}

function AdminProvidersContent() {
  const queryClient = useQueryClient()
  const tableState = useDataTableUrlState({ perPage: 20 })

  const [isDialogOpen, setIsDialogOpen] = React.useState(false)
  const [editingProvider, setEditingProvider] = React.useState<ProviderItem | null>(null)
  const [formName, setFormName] = React.useState('')
  const [formDisplayName, setFormDisplayName] = React.useState('')
  const [formSupportsFast, setFormSupportsFast] = React.useState(true)
  const [formSupportsReasoning, setFormSupportsReasoning] = React.useState(false)
  const [confirmToggleProvider, setConfirmToggleProvider] = React.useState<ProviderItem | null>(null)

  const { data: providers = [], isLoading } = useQuery<ProviderItem[]>({
    queryKey: ['admin', 'providers'],
    queryFn: async () => {
      try {
        const res = await api.get('/providers')
        return res.data?.data || []
      } catch {
        return []
      }
    },
  })

  const createMutation = useMutation({
    mutationFn: async (data: Partial<ProviderItem>) => {
      const res = await api.post('/admin/providers', data)
      return res.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'providers'] })
    },
  })

  const updateMutation = useMutation({
    mutationFn: async ({ id, ...data }: { id: string } & Partial<ProviderItem>) => {
      const res = await api.put(`/admin/providers/${id}`, data)
      return res.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'providers'] })
    },
  })

  const isMutating = createMutation.isPending || updateMutation.isPending

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
    if (!formName.trim() || !formDisplayName.trim()) return

    const payload = {
      name: formName.trim(),
      display_name: formDisplayName.trim(),
      supports_fast: formSupportsFast,
      supports_reasoning: formSupportsReasoning,
    }

    if (editingProvider) {
      await updateMutation.mutateAsync({ id: editingProvider.id, ...payload })
    } else {
      await createMutation.mutateAsync({ ...payload, is_active: true })
    }
    setIsDialogOpen(false)
  }

  const handleToggleActive = async (provider: ProviderItem) => {
    await updateMutation.mutateAsync({
      id: provider.id,
      is_active: !provider.is_active,
    })
    setConfirmToggleProvider(null)
  }

  // Filter & sort
  const filteredAndSorted = React.useMemo(() => {
    let result = [...providers]
    const q = (tableState.immediate.q || '').trim().toLowerCase()
    if (q) {
      result = result.filter(
        (p) =>
          p.name?.toLowerCase().includes(q) ||
          p.display_name?.toLowerCase().includes(q)
      )
    }
    const status = tableState.immediate.filters.status
    if (status === 'active') {
      result = result.filter((p) => p.is_active)
    } else if (status === 'inactive') {
      result = result.filter((p) => !p.is_active)
    }

    const sortCol = tableState.immediate.sort
    const isAsc = tableState.immediate.order === 'asc'
    if (sortCol === 'name') {
      result.sort((a, b) =>
        isAsc ? a.name.localeCompare(b.name) : b.name.localeCompare(a.name)
      )
    } else if (sortCol === 'display_name') {
      result.sort((a, b) =>
        isAsc ? a.display_name.localeCompare(b.display_name) : b.display_name.localeCompare(a.display_name)
      )
    } else if (sortCol === 'status') {
      result.sort((a, b) => {
        const aVal = a.is_active ? 1 : 0
        const bVal = b.is_active ? 1 : 0
        return isAsc ? aVal - bVal : bVal - aVal
      })
    }
    return result
  }, [providers, tableState.immediate.q, tableState.immediate.filters.status, tableState.immediate.sort, tableState.immediate.order])

  const total = filteredAndSorted.length
  const totalPages = Math.max(1, Math.ceil(total / tableState.perPage))
  const paginatedData = React.useMemo(() => {
    const start = (tableState.page - 1) * tableState.perPage
    return filteredAndSorted.slice(start, start + tableState.perPage)
  }, [filteredAndSorted, tableState.page, tableState.perPage])

  // KPI calculations
  const totalProviders = providers.length
  const activeProviders = providers.filter((p) => p.is_active).length
  const fastProviders = providers.filter((p) => p.supports_fast).length
  const reasoningProviders = providers.filter((p) => p.supports_reasoning).length

  const statsSection = (
    <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
      <AdminStatCard label="Total Provider" value={totalProviders} icon={Cpu} />
      <AdminStatCard label="Provider Aktif" value={activeProviders} icon={CheckCircle2} colorClassName="text-green-600" />
      <AdminStatCard label="Mode Fast" value={fastProviders} icon={Zap} colorClassName="text-amber-600" />
      <AdminStatCard label="Mode Reasoning" value={reasoningProviders} icon={Brain} colorClassName="text-blue-600" />
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

  const columns = React.useMemo<ColumnDef<ProviderItem>[]>(
    () => [
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
            Nama ID
            <ArrowUpDown className="w-3.5 h-3.5" />
          </button>
        ),
        cell: ({ row }) => <span className="font-bold text-foreground font-mono text-xs">{row.original.name}</span>,
      },
      {
        accessorKey: 'display_name',
        header: () => (
          <button
            type="button"
            onClick={() => tableState.setSort('display_name')}
            className="flex items-center gap-1.5 font-bold hover:text-foreground cursor-pointer"
          >
            Display Name
            <ArrowUpDown className="w-3.5 h-3.5" />
          </button>
        ),
        cell: ({ row }) => <span className="font-semibold text-foreground">{row.original.display_name}</span>,
      },
      {
        accessorKey: 'supports_fast',
        header: () => <span className="text-center font-bold block">Fast</span>,
        cell: ({ row }) => (
          <div className="text-center">
            {row.original.supports_fast ? (
              <Badge variant="outline" className="border-green-500 text-green-600 bg-green-50 dark:bg-green-950/40 text-xs">
                Ya
              </Badge>
            ) : (
              <span className="text-xs text-muted-foreground">-</span>
            )}
          </div>
        ),
      },
      {
        accessorKey: 'supports_reasoning',
        header: () => <span className="text-center font-bold block">Reasoning</span>,
        cell: ({ row }) => (
          <div className="text-center">
            {row.original.supports_reasoning ? (
              <Badge variant="outline" className="border-blue-500 text-blue-600 bg-blue-50 dark:bg-blue-950/40 text-xs">
                Ya
              </Badge>
            ) : (
              <span className="text-xs text-muted-foreground">-</span>
            )}
          </div>
        ),
      },
      {
        accessorKey: 'is_active',
        header: () => (
          <button
            type="button"
            onClick={() => tableState.setSort('status')}
            className="flex items-center gap-1.5 font-bold hover:text-foreground cursor-pointer mx-auto"
          >
            Status
            <ArrowUpDown className="w-3.5 h-3.5" />
          </button>
        ),
        cell: ({ row }) => (
          <div className="text-center">
            <Badge
              variant={row.original.is_active ? 'default' : 'secondary'}
              className={row.original.is_active ? 'bg-emerald-600 text-white' : 'bg-muted text-muted-foreground'}
            >
              {row.original.is_active ? 'Aktif' : 'Nonaktif'}
            </Badge>
          </div>
        ),
      },
      {
        id: 'actions',
        header: () => <span className="font-bold block text-right pr-2">Aksi</span>,
        cell: ({ row }) => (
          <div className="flex justify-end gap-1.5 pr-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => handleOpenEdit(row.original)}
              className="h-8 px-2.5 text-xs font-bold border-2 border-foreground cursor-pointer"
            >
              <Pencil className="w-3.5 h-3.5 mr-1" />
              Edit
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setConfirmToggleProvider(row.original)}
              className={`h-8 px-2.5 text-xs font-bold border-2 cursor-pointer ${
                row.original.is_active
                  ? 'border-rose-600 text-rose-600 hover:bg-rose-50'
                  : 'border-emerald-600 text-emerald-600 hover:bg-emerald-50'
              }`}
            >
              {row.original.is_active ? (
                <>
                  <PowerOff className="w-3.5 h-3.5 mr-1 text-rose-600" />
                  Nonaktifkan
                </>
              ) : (
                <>
                  <Power className="w-3.5 h-3.5 mr-1 text-emerald-600" />
                  Aktifkan
                </>
              )}
            </Button>
          </div>
        ),
      },
    ],
    [tableState]
  )

  // Chart data for Spec 03 table 3.1: Latensi Rata-rata (ms) & Success Rate (%) per Gateway Pembayaran / AI Provider
  const chartLabels = ['Sen', 'Sel', 'Rab', 'Kam', 'Jum', 'Sab', 'Min']
  const chartSeries = [
    { name: 'Latensi Rata-rata (ms)', data: [420, 390, 410, 450, 380, 360, 375], color: '#3b5bdb' },
    { name: 'Success Rate (%)', data: [99.4, 99.8, 98.9, 99.2, 99.7, 99.9, 99.5], color: '#10b981' },
  ]

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-xl font-black uppercase tracking-wide text-foreground">
            Monitoring Provider AI
          </h1>
          <p className="text-xs font-bold text-muted-foreground">
            Kelola dan pantau performa latensi, reliabilitas, dan status provider model AI
          </p>
        </div>
        <Button
          onClick={handleOpenAdd}
          className="font-bold border-2 border-foreground shadow-[2px_2px_0px_0px] shadow-foreground cursor-pointer"
        >
          <Plus className="mr-2 h-4 w-4" /> Tambah Provider
        </Button>
      </div>

      <SmartInsightCard
        title="Insight Status & Performa Provider AI"
        summaryText={`Terdapat ${totalProviders} provider terkonfigurasi (${activeProviders} aktif). Reliabilitas gateway AI stabil pada tingkat keberhasilan 99.6% dengan latensi p95 terkendali di bawah 450ms.`}
        deltaPercent={0.4}
        deltaDirection="up"
        isPositiveDelta={true}
        metricsSummary={[
          { label: 'Total Provider', value: totalProviders },
          { label: 'Provider Aktif', value: activeProviders },
          { label: 'Success Rate', value: '99.6%' },
          { label: 'Rata-rata Latensi', value: '395ms' },
        ]}
      />

      {/* MultiSeriesChart for Spec 03 */}
      <div className="border-2 rounded-md border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-5 space-y-3">
        <div className="flex items-center justify-between border-b-2 border-foreground pb-3">
          <div className="flex items-center gap-2">
            <Activity className="w-5 h-5 text-primary" />
            <h3 className="text-sm font-black uppercase tracking-wide">
              Tren Performa Provider (Latensi vs Success Rate)
            </h3>
          </div>
          <span className="text-xs font-bold text-muted-foreground">Periode: 7 Hari Terakhir</span>
        </div>
        <div className="h-[260px] w-full">
          <MultiSeriesChart labels={chartLabels} series={chartSeries} type="line" />
        </div>
      </div>

      <DataTableLayout
        stats={statsSection}
        filters={filterSection}
        columns={columns}
        data={paginatedData}
        isLoading={isLoading}
        total={total}
        totalPages={totalPages}
        tableState={tableState}
        searchPlaceholder="Cari nama atau display provider..."
        emptyMessage="Tidak ada provider ditemukan."
      />

      {/* Create / Edit Dialog */}
      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent className="sm:max-w-[425px] border-2 rounded-md border-foreground">
          <DialogHeader>
            <DialogTitle className="font-black uppercase tracking-wide">
              {editingProvider ? 'Edit Provider' : 'Tambah Provider'}
            </DialogTitle>
            <DialogDescription className="font-medium text-xs">
              Isi detail provider di bawah ini.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="provider-name" className="text-xs font-bold uppercase">Nama ID (slug)</Label>
              <Input
                id="provider-name"
                value={formName}
                onChange={(e) => setFormName(e.target.value)}
                placeholder="misal: openai, anthropic"
                disabled={Boolean(editingProvider)}
                className="font-medium text-sm border-2 border-foreground"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="provider-display" className="text-xs font-bold uppercase">Display Name</Label>
              <Input
                id="provider-display"
                value={formDisplayName}
                onChange={(e) => setFormDisplayName(e.target.value)}
                placeholder="misal: OpenAI GPT-4o"
                className="font-medium text-sm border-2 border-foreground"
              />
            </div>
            <div className="flex items-center gap-4 pt-2">
              <label className="flex items-center gap-2 text-xs font-bold cursor-pointer">
                <input
                  type="checkbox"
                  checked={formSupportsFast}
                  onChange={(e) => setFormSupportsFast(e.target.checked)}
                  className="rounded border-2 border-foreground"
                />
                Supports Fast
              </label>
              <label className="flex items-center gap-2 text-xs font-bold cursor-pointer">
                <input
                  type="checkbox"
                  checked={formSupportsReasoning}
                  onChange={(e) => setFormSupportsReasoning(e.target.checked)}
                  className="rounded border-2 border-foreground"
                />
                Supports Reasoning
              </label>
            </div>
            <div className="flex justify-end gap-2 pt-4">
              <Button type="button" variant="outline" onClick={() => setIsDialogOpen(false)} className="border-2 border-foreground">
                Batal
              </Button>
              <Button type="submit" disabled={isMutating} className="border-2 border-foreground font-bold">
                {isMutating ? 'Menyimpan...' : 'Simpan'}
              </Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>

      {/* Confirmation toggle active */}
      <ConfirmActionModal
        open={Boolean(confirmToggleProvider)}
        onOpenChange={(open) => {
          if (!open) setConfirmToggleProvider(null)
        }}
        actionType="deactivate"
        title={confirmToggleProvider?.is_active ? 'Nonaktifkan Provider' : 'Aktifkan Provider'}
        description={
          confirmToggleProvider?.is_active
            ? 'Apakah Anda yakin ingin menonaktifkan provider ini dari pool rute AI aktif?'
            : 'Apakah Anda yakin ingin mengaktifkan kembali provider ini untuk melayani permintaan pengguna?'
        }
        entityName={confirmToggleProvider?.display_name}
        confirmLabel={confirmToggleProvider?.is_active ? 'Ya, Nonaktifkan' : 'Ya, Aktifkan'}
        isPending={isMutating}
        onConfirm={() => {
          if (confirmToggleProvider) handleToggleActive(confirmToggleProvider)
        }}
      />
    </div>
  )
}

export default function AdminProvidersPage() {
  return (
    <React.Suspense fallback={<div className="h-40 animate-pulse bg-muted border-2 rounded-md border-foreground" />}>
      <AdminProvidersContent />
    </React.Suspense>
  )
}
