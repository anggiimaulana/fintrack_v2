'use client'

import { useState, useMemo } from 'react'
import {
  ShieldAlert,
  RotateCcw,
  Plus,
  Pencil,
  Trash2,
  KeyRound,
  Save,
  Undo2,
  Server,
  Network,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from '@/components/ui/dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  useSuperAdminGateway,
  GatewayEndpoint,
  GatewayRoute,
  EndpointModality,
  EndpointStatus,
  KillSwitchTarget,
} from '@/hooks/useSuperAdminGateway'

const ALL_MODALITIES: EndpointModality[] = ['chat', 'vision', 'voice', 'embedding', 'reasoning']
const ALL_STATUSES: EndpointStatus[] = ['active', 'degraded', 'inactive', 'suspended']
const ALL_USE_CASES: { key: EndpointModality; label: string; desc: string }[] = [
  { key: 'chat', label: 'Chat', desc: 'Percakapan AI & asisten keuangan interaktif' },
  { key: 'vision', label: 'Vision', desc: 'Pemindaian struk & pengenalan optik OCR' },
  { key: 'voice', label: 'Voice', desc: 'Input suara & transkripsi audio ke teks' },
  { key: 'embedding', label: 'Embedding', desc: 'Pencarian semantik & vektor transaksi' },
  { key: 'reasoning', label: 'Reasoning', desc: 'Analisis mendalam & rekomendasi SPK' },
]

export function KillSwitchPanel() {
  const { killSwitch, isLoadingKillSwitch, setKillSwitch, isSettingKillSwitch } = useSuperAdminGateway()
  const [confirmTarget, setConfirmTarget] = useState<KillSwitchTarget | null>(null)

  const isGlobalOn = Boolean(killSwitch.global)

  const handleToggleClick = (target: KillSwitchTarget, currentEnabled: boolean) => {
    if (!currentEnabled) {
      setConfirmTarget(target)
    } else {
      setKillSwitch({ target, enabled: false })
    }
  }

  const handleConfirmEnable = async () => {
    if (!confirmTarget) return
    const target = confirmTarget
    setConfirmTarget(null)
    await setKillSwitch({ target, enabled: true })
  }

  return (
    <>
      <Card className="border-border bg-card shadow-sm">
        <CardHeader className="pb-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <ShieldAlert className="h-5 w-5 text-red-500" />
              <CardTitle className="text-lg font-bold">Emergency Kill Switch</CardTitle>
            </div>
            {isGlobalOn && (
              <Badge variant="destructive" className="animate-pulse font-semibold">
                Layanan Global Dimatikan
              </Badge>
            )}
          </div>
          <CardDescription>
            Sakelar pemutus darurat AI Gateway. Menghentikan permintaan AI secara instan dan memicu fail-closed.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoadingKillSwitch ? (
            <div className="space-y-3">
              <Skeleton className="h-14 w-full" />
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              {/* Global kill switch */}
              <div
                className={`p-4 rounded-md border flex items-center justify-between transition-colors ${
                  isGlobalOn
                    ? 'border-red-500 bg-red-500/10'
                    : 'border-border bg-muted/40'
                }`}
              >
                <div className="space-y-0.5">
                  <div className="flex items-center gap-2">
                    <span className="font-bold text-foreground">Global Kill Switch</span>
                    <Badge variant={isGlobalOn ? 'destructive' : 'outline'}>
                      {isGlobalOn ? 'AKTIF (TERPUTUS)' : 'NONAKTIF (NORMAL)'}
                    </Badge>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    Mematikan seluruh lalu lintas AI di semua modalitas tanpa kecuali.
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <Switch
                    checked={isGlobalOn}
                    disabled={isSettingKillSwitch}
                    onCheckedChange={() => handleToggleClick('global', isGlobalOn)}
                    aria-label="Toggle Global Kill Switch"
                  />
                </div>
              </div>

              {/* Per-modality switches */}
              <div>
                <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                  Pemutus Per-Modalitas {isGlobalOn && '(Dinonaktifkan oleh sakelar global)'}
                </p>
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                  {ALL_MODALITIES.map((mod) => {
                    const isModOn = Boolean(killSwitch[mod])
                    const effectiveOn = isGlobalOn || isModOn
                    return (
                      <div
                        key={mod}
                        className={`p-3 rounded-md border flex items-center justify-between transition-colors ${
                          effectiveOn
                            ? 'border-red-500/60 bg-red-500/5'
                            : 'border-border bg-card'
                        } ${isGlobalOn ? 'opacity-60 cursor-not-allowed' : ''}`}
                      >
                        <div className="space-y-0.5">
                          <div className="flex items-center gap-1.5">
                            <span className="font-semibold text-sm capitalize text-foreground">
                              {mod}
                            </span>
                            <Badge
                              variant={effectiveOn ? 'destructive' : 'outline'}
                              className="text-[10px] px-1.5 py-0"
                            >
                              {effectiveOn ? 'PUTUS' : 'AKTIF'}
                            </Badge>
                          </div>
                          <p className="text-[11px] text-muted-foreground">
                            Modalitas {mod}
                          </p>
                        </div>
                        <Switch
                          checked={effectiveOn}
                          disabled={isGlobalOn || isSettingKillSwitch}
                          onCheckedChange={() => handleToggleClick(mod, isModOn)}
                          aria-label={`Toggle Kill Switch ${mod}`}
                        />
                      </div>
                    )
                  })}
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Confirmation dialog */}
      <Dialog open={Boolean(confirmTarget)} onOpenChange={(open) => !open && setConfirmTarget(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <div className="flex items-center gap-2 text-destructive">
              <ShieldAlert className="h-6 w-6" />
              <DialogTitle>Konfirmasi Pemutus Darurat</DialogTitle>
            </div>
            <DialogDescription className="pt-2 text-sm">
              {confirmTarget === 'global' ? (
                <span className="font-bold text-destructive">
                  Peringatan: Ini akan menghentikan seluruh layanan AI untuk semua pengguna seketika!
                </span>
              ) : (
                <span>
                  Peringatan: Ini akan menghentikan layanan modalitas{' '}
                  <strong className="text-destructive uppercase">{confirmTarget}</strong> untuk semua pengguna!
                </span>
              )}
            </DialogDescription>
          </DialogHeader>
          <div className="rounded-md border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive">
            Seluruh permintaan baru pada target ini akan ditolak (fail-closed). Lanjutkan hanya dalam kondisi darurat.
          </div>
          <DialogFooter className="gap-2 sm:gap-0">
            <Button
              type="button"
              variant="outline"
              onClick={() => setConfirmTarget(null)}
              disabled={isSettingKillSwitch}
            >
              Batal
            </Button>
            <Button
              type="button"
              variant="destructive"
              onClick={handleConfirmEnable}
              disabled={isSettingKillSwitch}
            >
              {isSettingKillSwitch ? 'Mengaktifkan...' : 'Ya, Aktifkan Kill Switch'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}

export function EndpointsPanel() {
  const {
    endpoints,
    isLoadingEndpoints,
    createEndpoint,
    isCreatingEndpoint,
    updateEndpoint,
    isUpdatingEndpoint,
    deleteEndpoint,
    isDeletingEndpoint,
    resetBreaker,
    isResettingBreaker,
  } = useSuperAdminGateway()

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingEndpoint, setEditingEndpoint] = useState<GatewayEndpoint | null>(null)
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null)

  // Form state
  const [name, setName] = useState('')
  const [baseUrl, setBaseUrl] = useState('')
  const [provider, setProvider] = useState('9router')
  const [priority, setPriority] = useState(1)
  const [weight, setWeight] = useState(100)
  const [status, setStatus] = useState<EndpointStatus>('active')
  const [modalities, setModalities] = useState<EndpointModality[]>(ALL_MODALITIES)
  const [apiKey, setApiKey] = useState('')

  const openCreateDialog = () => {
    setEditingEndpoint(null)
    setName('')
    setBaseUrl('')
    setProvider('9router')
    setPriority(1)
    setWeight(100)
    setStatus('active')
    setModalities([...ALL_MODALITIES])
    setApiKey('')
    setDialogOpen(true)
  }

  const openEditDialog = (ep: GatewayEndpoint) => {
    setEditingEndpoint(ep)
    setName(ep.name)
    setBaseUrl(ep.base_url)
    setProvider(ep.provider || '9router')
    setPriority(ep.priority ?? 1)
    setWeight(ep.weight ?? 100)
    setStatus(ep.status || 'active')
    setModalities(ep.modalities?.length ? [...ep.modalities] : [...ALL_MODALITIES])
    setApiKey('')
    setDialogOpen(true)
  }

  const toggleModality = (mod: EndpointModality) => {
    setModalities((prev) =>
      prev.includes(mod) ? prev.filter((m) => m !== mod) : [...prev, mod]
    )
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim() || !baseUrl.trim()) return

    if (editingEndpoint) {
      await updateEndpoint({
        id: editingEndpoint.id,
        payload: {
          name: name.trim(),
          base_url: baseUrl.trim(),
          provider: provider.trim(),
          priority: Number(priority) || 1,
          weight: Number(weight) || 100,
          status,
          modalities,
          api_key: apiKey.trim() || undefined,
        },
      })
    } else {
      await createEndpoint({
        name: name.trim(),
        base_url: baseUrl.trim(),
        provider: provider.trim() || '9router',
        priority: Number(priority) || 1,
        weight: Number(weight) || 100,
        status,
        modalities,
        api_key: apiKey.trim() || undefined,
      })
    }
    setDialogOpen(false)
  }

  const handleDelete = async () => {
    if (!deleteConfirmId) return
    const id = deleteConfirmId
    setDeleteConfirmId(null)
    await deleteEndpoint(id)
  }

  const getBreakerBadge = (state: string) => {
    switch (state) {
      case 'closed':
        return (
          <Badge className="bg-emerald-600/15 text-emerald-600 dark:text-emerald-400 border-emerald-600/30 font-mono text-[11px]">
            closed
          </Badge>
        )
      case 'half_open':
        return (
          <Badge className="bg-amber-600/15 text-amber-600 dark:text-amber-400 border-amber-600/30 font-mono text-[11px]">
            half_open
          </Badge>
        )
      case 'open':
        return (
          <Badge className="bg-red-600/15 text-red-700 dark:text-red-400 border-red-600/30 font-mono text-[11px]">
            open
          </Badge>
        )
      default:
        return (
          <Badge variant="outline" className="font-mono text-[11px]">
            {state || 'unknown'}
          </Badge>
        )
    }
  }

  const getStatusBadge = (st: EndpointStatus) => {
    switch (st) {
      case 'active':
        return <Badge variant="outline" className="text-emerald-600 border-emerald-600/30">active</Badge>
      case 'degraded':
        return <Badge variant="outline" className="text-amber-600 border-amber-600/30">degraded</Badge>
      case 'inactive':
        return <Badge variant="outline" className="text-muted-foreground border-border">inactive</Badge>
      case 'suspended':
        return <Badge variant="outline" className="text-red-600 border-red-600/30">suspended</Badge>
      default:
        return <Badge variant="outline">{st}</Badge>
    }
  }

  const getHealthBadge = (health: string) => {
    if (health === 'healthy') {
      return <Badge className="bg-emerald-500/10 text-emerald-600 border-emerald-500/20">healthy</Badge>
    }
    return <Badge variant="outline" className="text-amber-600 border-amber-600/30">{health || 'unhealthy'}</Badge>
  }

  return (
    <>
      <Card className="border-border bg-card shadow-sm">
        <CardHeader className="flex flex-row items-center justify-between pb-4">
          <div>
            <div className="flex items-center gap-2">
              <Server className="h-5 w-5 text-primary" />
              <CardTitle className="text-lg font-bold">Daftar Endpoint AI</CardTitle>
            </div>
            <CardDescription>
              Penyedia backend dan router AI yang terdaftar dalam gateway
            </CardDescription>
          </div>
          <Button onClick={openCreateDialog} className="gap-1.5" size="sm">
            <Plus className="h-4 w-4" />
            Tambah Endpoint
          </Button>
        </CardHeader>
        <CardContent>
          {isLoadingEndpoints ? (
            <div className="space-y-2">
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-14 w-full" />
              <Skeleton className="h-14 w-full" />
            </div>
          ) : endpoints.length === 0 ? (
            <div className="text-center py-12 border rounded-md border-dashed border-border">
              <Server className="h-10 w-10 text-muted-foreground mx-auto mb-3" />
              <h3 className="font-bold text-foreground mb-1">Belum ada endpoint AI</h3>
              <p className="text-sm text-muted-foreground mb-4">
                Tambahkan endpoint pertama untuk mengarahkan rute AI.
              </p>
              <Button onClick={openCreateDialog} size="sm">
                <Plus className="h-4 w-4 mr-1.5" />
                Tambah Endpoint Sekarang
              </Button>
            </div>
          ) : (
            <div className="rounded-md border border-border overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-[180px]">Nama & Provider</TableHead>
                    <TableHead className="min-w-[200px]">Base URL</TableHead>
                    <TableHead>Prioritas / Bobot</TableHead>
                    <TableHead>Status & Health</TableHead>
                    <TableHead>Circuit Breaker</TableHead>
                    <TableHead>Modalitas</TableHead>
                    <TableHead>Key</TableHead>
                    <TableHead>Check Terakhir</TableHead>
                    <TableHead className="text-right">Aksi</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {endpoints.map((ep) => (
                    <TableRow key={ep.id}>
                      <TableCell>
                        <div className="font-semibold text-foreground">{ep.name}</div>
                        <div className="text-xs text-muted-foreground">{ep.provider}</div>
                      </TableCell>
                      <TableCell>
                        <div className="font-mono text-xs max-w-[220px] truncate" title={ep.base_url}>
                          {ep.base_url}
                        </div>
                        {ep.last_error && (
                          <div
                            className="text-[11px] text-destructive truncate max-w-[220px] mt-0.5"
                            title={ep.last_error}
                          >
                            Err: {ep.last_error}
                          </div>
                        )}
                      </TableCell>
                      <TableCell>
                        <div className="text-xs">
                          P: <span className="font-mono font-semibold">{ep.priority}</span>
                          <span className="text-muted-foreground ml-1">/ W: {ep.weight}</span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-col gap-1 items-start">
                          {getStatusBadge(ep.status)}
                          {getHealthBadge(ep.health)}
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1.5">
                          {getBreakerBadge(ep.breaker_state)}
                          {ep.breaker_state !== 'closed' && (
                            <Button
                              variant="ghost"
                              size="sm"
                              className="h-7 px-2 text-xs text-muted-foreground hover:text-foreground"
                              onClick={() => resetBreaker(ep.id)}
                              disabled={isResettingBreaker}
                              title="Reset Breaker ke Closed"
                            >
                              <RotateCcw className="h-3 w-3" />
                            </Button>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-wrap gap-1 max-w-[150px]">
                          {ep.modalities?.map((m) => (
                            <Badge key={m} variant="secondary" className="text-[10px] px-1 py-0 font-normal">
                              {m}
                            </Badge>
                          ))}
                        </div>
                      </TableCell>
                      <TableCell>
                        {ep.has_api_key ? (
                          <Badge variant="outline" className="text-emerald-600 border-emerald-600/30 gap-1 text-[11px]">
                            <KeyRound className="h-3 w-3" /> Ada
                          </Badge>
                        ) : (
                          <Badge variant="outline" className="text-muted-foreground text-[11px]">
                            Kosong
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell>
                        <div className="text-[11px] text-muted-foreground whitespace-nowrap">
                          {ep.last_check_at ? new Date(ep.last_check_at).toLocaleTimeString() : '-'}
                        </div>
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-1">
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-8 w-8 text-muted-foreground hover:text-foreground"
                            onClick={() => openEditDialog(ep)}
                            aria-label={`Edit ${ep.name}`}
                          >
                            <Pencil className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-8 w-8 text-destructive hover:bg-destructive/10"
                            onClick={() => setDeleteConfirmId(ep.id)}
                            aria-label={`Hapus ${ep.name}`}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Create / Edit Dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-lg">
          <form onSubmit={handleSubmit}>
            <DialogHeader>
              <DialogTitle>
                {editingEndpoint ? 'Edit Endpoint AI' : 'Tambah Endpoint AI Baru'}
              </DialogTitle>
              <DialogDescription>
                Konfigurasi URL dan kredensial penyedia model AI
              </DialogDescription>
            </DialogHeader>

            <div className="grid gap-4 py-4">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <Label htmlFor="ep-name">Nama Endpoint *</Label>
                  <Input
                    id="ep-name"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="misal: router-primary"
                    required
                  />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="ep-provider">Provider</Label>
                  <Input
                    id="ep-provider"
                    value={provider}
                    onChange={(e) => setProvider(e.target.value)}
                    placeholder="9router / openai / gemini"
                    required
                  />
                </div>
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="ep-baseurl">Base URL *</Label>
                <Input
                  id="ep-baseurl"
                  value={baseUrl}
                  onChange={(e) => setBaseUrl(e.target.value)}
                  placeholder="https://api.openai.com/v1"
                  required
                />
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                <div className="space-y-1.5">
                  <Label htmlFor="ep-priority">Prioritas</Label>
                  <Input
                    id="ep-priority"
                    type="number"
                    min={1}
                    value={priority}
                    onChange={(e) => setPriority(parseInt(e.target.value, 10) || 1)}
                  />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="ep-weight">Bobot (Weight)</Label>
                  <Input
                    id="ep-weight"
                    type="number"
                    min={1}
                    value={weight}
                    onChange={(e) => setWeight(parseInt(e.target.value, 10) || 100)}
                  />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="ep-status">Status</Label>
                  <Select value={status} onValueChange={(v) => setStatus(v as EndpointStatus)}>
                    <SelectTrigger id="ep-status">
                      <SelectValue placeholder="Pilih status" />
                    </SelectTrigger>
                    <SelectContent>
                      {ALL_STATUSES.map((st) => (
                        <SelectItem key={st} value={st}>
                          {st}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <div className="space-y-1.5">
                <Label>Modalitas yang Didukung</Label>
                <div className="flex flex-wrap gap-2 pt-1">
                  {ALL_MODALITIES.map((mod) => {
                    const isChecked = modalities.includes(mod)
                    return (
                      <Button
                        key={mod}
                        type="button"
                        size="sm"
                        variant={isChecked ? 'default' : 'outline'}
                        className="text-xs h-8 capitalize"
                        onClick={() => toggleModality(mod)}
                      >
                        {mod}
                      </Button>
                    )
                  })}
                </div>
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="ep-apikey">API Key</Label>
                <Input
                  id="ep-apikey"
                  type="password"
                  value={apiKey}
                  onChange={(e) => setApiKey(e.target.value)}
                  placeholder={editingEndpoint ? '••••••••••••••••' : 'sk-...'}
                  autoComplete="new-password"
                />
                {editingEndpoint && (
                  <p className="text-[11px] text-muted-foreground">
                    kosongkan untuk mempertahankan kunci lama
                  </p>
                )}
              </div>
            </div>

            <DialogFooter className="gap-2 sm:gap-0">
              <Button
                type="button"
                variant="outline"
                onClick={() => setDialogOpen(false)}
                disabled={isCreatingEndpoint || isUpdatingEndpoint}
              >
                Batal
              </Button>
              <Button
                type="submit"
                disabled={isCreatingEndpoint || isUpdatingEndpoint || !name.trim() || !baseUrl.trim()}
              >
                {isCreatingEndpoint || isUpdatingEndpoint ? 'Menyimpan...' : 'Simpan Endpoint'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <Dialog open={Boolean(deleteConfirmId)} onOpenChange={(open) => !open && setDeleteConfirmId(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Nonaktifkan Endpoint AI</DialogTitle>
            <DialogDescription>
              Endpoint akan ditandai inactive. Jika endpoint masih dipakai rute aktif, backend akan menolak penghapusan.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2 sm:gap-0">
            <Button
              type="button"
              variant="outline"
              onClick={() => setDeleteConfirmId(null)}
              disabled={isDeletingEndpoint}
            >
              Batal
            </Button>
            <Button
              type="button"
              variant="destructive"
              onClick={handleDelete}
              disabled={isDeletingEndpoint}
            >
              {isDeletingEndpoint ? 'Menonaktifkan...' : 'Ya, Nonaktifkan'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}

export function RoutingMatrixPanel() {
  const { endpoints, routes, isLoadingRoutes, saveRoutes, isSavingRoutes } = useSuperAdminGateway()

  // Local state for full matrix editing
  const [localRoutes, setLocalRoutes] = useState<GatewayRoute[]>([])
  const [isConfirmOpen, setIsConfirmOpen] = useState(false)

  // Adopt fetched routes without a setState-in-effect: only a real change in
  // the server payload replaces the locally edited matrix.
  const routesSignature = useMemo(() => JSON.stringify(routes), [routes])
  const [adoptedRoutesSignature, setAdoptedRoutesSignature] = useState<string | null>(null)
  if (adoptedRoutesSignature !== routesSignature) {
    setAdoptedRoutesSignature(routesSignature)
    setLocalRoutes(routes)
  }
  // Calculate dirty state
  const isDirty = JSON.stringify(localRoutes) !== JSON.stringify(routes)

  const handleReset = () => {
    setLocalRoutes(routes)
  }

  const handleAddRow = (useCase: EndpointModality) => {
    const defaultEndpoint = endpoints[0]?.id || ''
    const newRoute: GatewayRoute = {
      use_case: useCase,
      endpoint_id: defaultEndpoint,
      model: 'gpt-4o-mini',
      priority: localRoutes.filter((r) => r.use_case === useCase).length + 1,
      timeout_ms: 30000,
      max_retries: 2,
      is_active: true,
    }
    setLocalRoutes((prev) => [...prev, newRoute])
  }

  const handleRemoveRow = (index: number) => {
    setLocalRoutes((prev) => prev.filter((_, i) => i !== index))
  }

  const handleUpdateRow = (index: number, patch: Partial<GatewayRoute>) => {
    setLocalRoutes((prev) =>
      prev.map((r, i) => (i === index ? { ...r, ...patch } : r))
    )
  }

  const handleConfirmSave = async () => {
    setIsConfirmOpen(false)
    await saveRoutes(localRoutes)
  }

  return (
    <>
      <Card className="border-border bg-card shadow-sm">
        <CardHeader className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 pb-4">
          <div>
            <div className="flex items-center gap-2">
              <Network className="h-5 w-5 text-primary" />
              <CardTitle className="text-lg font-bold">Matriks Rute AI (Routing Matrix)</CardTitle>
            </div>
            <CardDescription>
              Atur endpoint tujuan, model, batas timeout, dan retries untuk tiap modalitas. Penggantian disimpan secara atomik.
            </CardDescription>
          </div>
          <div className="flex items-center gap-2">
            {isDirty && (
              <Badge variant="outline" className="text-amber-500 border-amber-500/30">
                Ada perubahan belum disimpan
              </Badge>
            )}
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={handleReset}
              disabled={!isDirty || isSavingRoutes}
              className="gap-1.5"
            >
              <Undo2 className="h-4 w-4" />
              Batalkan perubahan
            </Button>
            <Button
              type="button"
              size="sm"
              onClick={() => setIsConfirmOpen(true)}
              disabled={!isDirty || isSavingRoutes}
              className="gap-1.5"
            >
              <Save className="h-4 w-4" />
              {isSavingRoutes ? 'Menyimpan...' : 'Simpan Matriks'}
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {isLoadingRoutes ? (
            <div className="space-y-4">
              <Skeleton className="h-20 w-full" />
              <Skeleton className="h-20 w-full" />
            </div>
          ) : (
            <div className="space-y-8">
              {ALL_USE_CASES.map(({ key: useCase, label, desc }) => {
                const caseRows = localRoutes
                  .map((r, originalIdx) => ({ route: r, originalIdx }))
                  .filter(({ route }) => route.use_case === useCase)

                return (
                  <div key={useCase} className="space-y-3">
                    <div className="flex items-center justify-between border-b pb-2">
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="font-bold text-foreground uppercase tracking-wide text-sm">
                            {label}
                          </span>
                          <Badge variant="secondary" className="text-[10px] px-1.5">
                            {caseRows.length} rute
                          </Badge>
                        </div>
                        <p className="text-xs text-muted-foreground">{desc}</p>
                      </div>
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => handleAddRow(useCase)}
                        className="h-8 gap-1 text-xs"
                      >
                        <Plus className="h-3.5 w-3.5" />
                        Tambah Rute
                      </Button>
                    </div>

                    {caseRows.length === 0 ? (
                      <div className="text-xs text-muted-foreground italic py-3 px-4 border border-dashed rounded bg-muted/20">
                        Belum ada rute untuk modalitas {label}. Klik &quot;Tambah Rute&quot; di atas.
                      </div>
                    ) : (
                      <div className="rounded-md border border-border overflow-x-auto">
                        <Table>
                          <TableHeader>
                            <TableRow>
                              <TableHead className="w-[200px]">Endpoint</TableHead>
                              <TableHead className="min-w-[160px]">Model</TableHead>
                              <TableHead className="w-[100px]">Prioritas</TableHead>
                              <TableHead className="w-[120px]">Timeout (ms)</TableHead>
                              <TableHead className="w-[100px]">Max Retries</TableHead>
                              <TableHead className="w-[90px]">Status</TableHead>
                              <TableHead className="w-[60px] text-right"></TableHead>
                            </TableRow>
                          </TableHeader>
                          <TableBody>
                            {caseRows.map(({ route, originalIdx }) => (
                              <TableRow key={route.id || `row-${originalIdx}`}>
                                <TableCell>
                                  <Select
                                    value={route.endpoint_id}
                                    onValueChange={(val) =>
                                      handleUpdateRow(originalIdx, { endpoint_id: val })
                                    }
                                  >
                                    <SelectTrigger className="h-8 text-xs">
                                      <SelectValue placeholder="Pilih endpoint" />
                                    </SelectTrigger>
                                    <SelectContent>
                                      {endpoints.map((ep) => (
                                        <SelectItem key={ep.id} value={ep.id} className="text-xs">
                                          {ep.name} ({ep.provider})
                                        </SelectItem>
                                      ))}
                                    </SelectContent>
                                  </Select>
                                </TableCell>
                                <TableCell>
                                  <Input
                                    value={route.model}
                                    onChange={(e) =>
                                      handleUpdateRow(originalIdx, { model: e.target.value })
                                    }
                                    className="h-8 text-xs font-mono"
                                    placeholder="misal: gpt-4o-mini"
                                  />
                                </TableCell>
                                <TableCell>
                                  <Input
                                    type="number"
                                    min={1}
                                    value={route.priority}
                                    onChange={(e) =>
                                      handleUpdateRow(originalIdx, {
                                        priority: parseInt(e.target.value, 10) || 1,
                                      })
                                    }
                                    className="h-8 text-xs"
                                  />
                                </TableCell>
                                <TableCell>
                                  <Input
                                    type="number"
                                    min={0}
                                    step={1000}
                                    value={route.timeout_ms}
                                    onChange={(e) =>
                                      handleUpdateRow(originalIdx, {
                                        timeout_ms: parseInt(e.target.value, 10) || 0,
                                      })
                                    }
                                    className="h-8 text-xs font-mono"
                                  />
                                </TableCell>
                                <TableCell>
                                  <Input
                                    type="number"
                                    min={0}
                                    value={route.max_retries}
                                    onChange={(e) =>
                                      handleUpdateRow(originalIdx, {
                                        max_retries: parseInt(e.target.value, 10) || 0,
                                      })
                                    }
                                    className="h-8 text-xs"
                                  />
                                </TableCell>
                                <TableCell>
                                  <div className="flex items-center gap-1.5">
                                    <Switch
                                      checked={route.is_active}
                                      onCheckedChange={(val) =>
                                        handleUpdateRow(originalIdx, { is_active: val })
                                      }
                                      aria-label="Aktifkan Rute"
                                    />
                                    <span className="text-[11px] text-muted-foreground">
                                      {route.is_active ? 'Aktif' : 'Off'}
                                    </span>
                                  </div>
                                </TableCell>
                                <TableCell className="text-right">
                                  <Button
                                    type="button"
                                    variant="ghost"
                                    size="icon"
                                    className="h-7 w-7 text-destructive hover:bg-destructive/10"
                                    onClick={() => handleRemoveRow(originalIdx)}
                                    aria-label="Hapus Rute"
                                  >
                                    <Trash2 className="h-3.5 w-3.5" />
                                  </Button>
                                </TableCell>
                              </TableRow>
                            ))}
                          </TableBody>
                        </Table>
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Confirm Save Matrix Dialog */}
      <Dialog open={isConfirmOpen} onOpenChange={setIsConfirmOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Simpan Matriks Rute AI</DialogTitle>
            <DialogDescription>
              Tindakan ini akan mengganti seluruh konfigurasi rute AI secara atomik. Pastikan semua endpoint tujuan dan model sudah benar.
            </DialogDescription>
          </DialogHeader>
          <div className="text-xs text-muted-foreground bg-muted p-3 rounded">
            Total rute yang akan diterapkan: <strong>{localRoutes.length}</strong> rute.
          </div>
          <DialogFooter className="gap-2 sm:gap-0">
            <Button
              type="button"
              variant="outline"
              onClick={() => setIsConfirmOpen(false)}
              disabled={isSavingRoutes}
            >
              Batal
            </Button>
            <Button
              type="button"
              onClick={handleConfirmSave}
              disabled={isSavingRoutes}
            >
              {isSavingRoutes ? 'Menyimpan...' : 'Ya, Simpan Matriks'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
