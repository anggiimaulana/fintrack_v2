'use client'

import * as React from 'react'
import { toast } from 'sonner'
import { Tag, Plus, Pencil, CheckCircle2, Trash2, RotateCw, Save, Loader2 } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { ErrorState } from '@/components/shared/ErrorState'
import {
  usePricingRules,
  PRICING_MODALITIES,
  type PricingRuleDTO,
  type PricingRulePayload,
} from '@/hooks/useSuperAdminPricing'

const DEFAULT_MULTIPLIERS: Record<string, number> = {
  chat: 1.0,
  vision: 2.5,
  voice: 3.0,
  reasoning: 2.0,
  embedding: 0.5,
}

/**
 * sellPer1k normalises the configured base price to a per-1,000-token figure.
 * A "per_10k" rule (e.g. Rp3.000 / 10.000 token) is divided by 10, matching the
 * backend's SellPricePer1kIDR.
 */
function sellPer1k(base: number, unit: 'per_1k' | 'per_10k'): number {
  return unit === 'per_10k' ? Math.floor(base / 10) : base
}

/** costIDR mirrors the backend formula: floor(sell/1k × tokens/1000 × markup × (1+pct) × modality). */
function costIDR(base: number, unit: 'per_1k' | 'per_10k', markup: number, percent: number, modalityMult: number, tokens = 1000): number {
  const baseCost = sellPer1k(base, unit) * (tokens / 1000)
  const withMarkup = baseCost * markup * (1 + percent / 100)
  const total = withMarkup * modalityMult
  return total > 0 ? Math.floor(total) : 0
}

/** apiErrorMessage extracts a server message from an unknown error, if present. */
function apiErrorMessage(err: unknown): string | null {
  if (typeof err !== 'object' || err === null || !('response' in err)) return null
  const { response } = err
  if (typeof response !== 'object' || response === null || !('data' in response)) return null
  const { data } = response
  if (typeof data !== 'object' || data === null || !('message' in data)) return null
  const { message } = data
  return typeof message === 'string' ? message : null
}

function formatIDR(v: number): string {
  return `Rp ${v.toLocaleString('id-ID')}`
}

interface FormState {
  rule_name: string
  base_per_1k_idr: string
  price_unit: 'per_1k' | 'per_10k'
  provider_cost_per_1k_idr: string
  markup_multiplier: string
  markup_percent: string
  multipliers: Record<string, string>
}

const EMPTY_FORM: FormState = {
  rule_name: '',
  base_per_1k_idr: '300',
  price_unit: 'per_1k',
  provider_cost_per_1k_idr: '200',
  markup_multiplier: '1.25',
  markup_percent: '0',
  multipliers: Object.fromEntries(PRICING_MODALITIES.map((m) => [m, String(DEFAULT_MULTIPLIERS[m] ?? 1)])),
}

function toForm(rule: PricingRuleDTO): FormState {
  return {
    rule_name: rule.rule_name,
    base_per_1k_idr: String(rule.base_per_1k_idr),
    price_unit: rule.price_unit ?? 'per_1k',
    provider_cost_per_1k_idr: String(rule.provider_cost_per_1k_idr ?? 0),
    markup_multiplier: String(rule.markup_multiplier),
    markup_percent: String(rule.markup_percent),
    multipliers: Object.fromEntries(
      PRICING_MODALITIES.map((m) => [m, String(rule.modality_multipliers?.[m] ?? DEFAULT_MULTIPLIERS[m] ?? 1)])
    ),
  }
}

export default function PricingPage() {
  const { rules, isLoading, isError, refetch, createRule, updateRule, activateRule, deleteRule } = usePricingRules()

  const [dialogOpen, setDialogOpen] = React.useState(false)
  const [editing, setEditing] = React.useState<PricingRuleDTO | null>(null)
  const [form, setForm] = React.useState<FormState>(EMPTY_FORM)
  const [deleteTarget, setDeleteTarget] = React.useState<PricingRuleDTO | null>(null)

  const openCreate = () => {
    setEditing(null)
    setForm(EMPTY_FORM)
    setDialogOpen(true)
  }

  const openEdit = (rule: PricingRuleDTO) => {
    setEditing(rule)
    setForm(toForm(rule))
    setDialogOpen(true)
  }

  const numeric = {
    base: Number(form.base_per_1k_idr) || 0,
    providerCost: Number(form.provider_cost_per_1k_idr) || 0,
    markup: Number(form.markup_multiplier) || 0,
    percent: Number(form.markup_percent) || 0,
  }

  const handleSave = async () => {
    if (!form.rule_name.trim()) {
      toast.error('Nama aturan wajib diisi.')
      return
    }
    const multipliers: Record<string, number> = {}
    for (const m of PRICING_MODALITIES) multipliers[m] = Number(form.multipliers[m]) || 0

    const payload: PricingRulePayload = {
      rule_name: form.rule_name.trim(),
      base_per_1k_idr: numeric.base,
      price_unit: form.price_unit,
      provider_cost_per_1k_idr: numeric.providerCost,
      markup_multiplier: numeric.markup,
      markup_percent: numeric.percent,
      modality_multipliers: multipliers,
    }
    try {
      if (editing) {
        await updateRule.mutateAsync({ id: editing.id, payload })
        toast.success('Aturan harga diperbarui.')
      } else {
        await createRule.mutateAsync({ ...payload, is_active: rules.length === 0 })
        toast.success('Aturan harga dibuat.')
      }
      setDialogOpen(false)
    } catch (err) {
      toast.error(apiErrorMessage(err) || 'Gagal menyimpan aturan harga.')
    }
  }

  const handleActivate = async (rule: PricingRuleDTO) => {
    try {
      await activateRule.mutateAsync(rule.id)
      toast.success(`Aturan "${rule.rule_name}" diaktifkan.`)
    } catch {
      toast.error('Gagal mengaktifkan aturan.')
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteRule.mutateAsync(deleteTarget.id)
      toast.success('Aturan dihapus.')
      setDeleteTarget(null)
    } catch (err) {
      toast.error(apiErrorMessage(err) || 'Gagal menghapus aturan.')
    }
  }

  const saving = createRule.isPending || updateRule.isPending

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-black uppercase tracking-wide flex items-center gap-2">
            <Tag className="w-6 h-6 text-primary" /> Harga &amp; Biaya AI
          </h1>
          <p className="text-sm font-bold text-muted-foreground mt-1">
            Atur harga dasar, markup, dan pengali per modalitas. Perubahan berlaku langsung tanpa restart.
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => refetch()} aria-label="Muat ulang">
            <RotateCw className="w-4 h-4" />
          </Button>
          <Button onClick={openCreate}>
            <Plus className="w-4 h-4 mr-2" /> Aturan Baru
          </Button>
        </div>
      </div>

      {isLoading ? (
        <div className="space-y-3">
          {[1, 2].map((i) => (
            <Skeleton key={i} className="h-28 w-full" />
          ))}
        </div>
      ) : isError ? (
        <ErrorState title="Gagal memuat aturan harga" onRetry={() => refetch()} />
      ) : rules.length === 0 ? (
        <Card>
          <CardContent className="py-10 text-center text-sm font-bold text-muted-foreground">
            Belum ada aturan harga. Buat satu agar biaya AI tercatat.
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {rules.map((rule) => (
            <Card key={rule.id} className={rule.is_active ? 'border-primary border-2' : 'border-2 border-foreground'}>
              <CardHeader>
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <CardTitle className="flex items-center gap-2">
                      {rule.rule_name}
                      {rule.is_active && (
                        <Badge className="gap-1">
                          <CheckCircle2 className="w-3 h-3" /> Aktif
                        </Badge>
                      )}
                    </CardTitle>
                    <CardDescription className="mt-1">
                      Jual {formatIDR(rule.base_per_1k_idr)} / {rule.price_unit === 'per_10k' ? '10.000' : '1.000'} token ·
                      {' '}modal {formatIDR(rule.provider_cost_per_1k_idr)}/1k · markup ×{rule.markup_multiplier}
                      {rule.markup_percent ? ` + ${rule.markup_percent}%` : ''}
                    </CardDescription>
                  </div>
                  <div className="flex gap-2 shrink-0">
                    {!rule.is_active && (
                      <Button size="sm" variant="outline" onClick={() => handleActivate(rule)} disabled={activateRule.isPending}>
                        Aktifkan
                      </Button>
                    )}
                    <Button size="sm" variant="outline" onClick={() => openEdit(rule)} aria-label="Ubah">
                      <Pencil className="w-4 h-4" />
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => setDeleteTarget(rule)}
                      disabled={rule.is_active}
                      aria-label="Hapus"
                    >
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-2 sm:grid-cols-5 gap-3">
                  {PRICING_MODALITIES.map((m) => {
                    const mult = rule.modality_multipliers?.[m] ?? 1
                    const cost = costIDR(rule.base_per_1k_idr, rule.price_unit ?? 'per_1k', rule.markup_multiplier, rule.markup_percent, mult)
                    const margin = cost - rule.provider_cost_per_1k_idr
                    return (
                      <div key={m} className="border-2 border-foreground/20 p-3">
                        <p className="text-xs font-black uppercase text-muted-foreground">{m}</p>
                        <p className="text-sm font-bold">×{mult}</p>
                        <p className="text-xs font-bold text-primary mt-1">1.000 = {formatIDR(cost)}</p>
                        <p className={`text-xs font-bold mt-0.5 ${margin >= 0 ? 'text-emerald-600' : 'text-destructive'}`}>
                          margin {formatIDR(margin)}
                        </p>
                      </div>
                    )
                  })}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-lg border-2 border-foreground">
          <DialogHeader>
            <DialogTitle>{editing ? 'Ubah Aturan Harga' : 'Aturan Harga Baru'}</DialogTitle>
            <DialogDescription>
              Rumus: floor(base × token/1000 × markup × (1 + persen/100) × pengali modalitas).
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-2">
            <div className="space-y-1.5">
              <Label htmlFor="rule-name">Nama Aturan</Label>
              <Input
                id="rule-name"
                value={form.rule_name}
                onChange={(e) => setForm((s) => ({ ...s, rule_name: e.target.value }))}
                placeholder="default"
              />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <Label htmlFor="rule-unit">Satuan Harga Jual</Label>
                <select
                  id="rule-unit"
                  className="h-10 w-full border-2 border-foreground bg-background px-3 text-sm font-bold"
                  value={form.price_unit}
                  onChange={(e) => setForm((s) => ({ ...s, price_unit: e.target.value as 'per_1k' | 'per_10k' }))}
                >
                  <option value="per_1k">per 1.000 token</option>
                  <option value="per_10k">per 10.000 token</option>
                </select>
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="rule-provider-cost">Harga Modal / 1k (Rp)</Label>
                <Input
                  id="rule-provider-cost"
                  type="number"
                  min={0}
                  value={form.provider_cost_per_1k_idr}
                  onChange={(e) => setForm((s) => ({ ...s, provider_cost_per_1k_idr: e.target.value }))}
                />
              </div>
            </div>

            <div className="grid grid-cols-3 gap-3">
              <div className="space-y-1.5">
                <Label htmlFor="rule-base">Harga Jual (Rp)</Label>
                <Input
                  id="rule-base"
                  type="number"
                  min={0}
                  value={form.base_per_1k_idr}
                  onChange={(e) => setForm((s) => ({ ...s, base_per_1k_idr: e.target.value }))}
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="rule-markup">Markup (×)</Label>
                <Input
                  id="rule-markup"
                  type="number"
                  step="0.01"
                  min={0}
                  value={form.markup_multiplier}
                  onChange={(e) => setForm((s) => ({ ...s, markup_multiplier: e.target.value }))}
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="rule-percent">Markup (%)</Label>
                <Input
                  id="rule-percent"
                  type="number"
                  min={0}
                  max={100}
                  value={form.markup_percent}
                  onChange={(e) => setForm((s) => ({ ...s, markup_percent: e.target.value }))}
                />
              </div>
            </div>

            <div>
              <Label className="mb-2 block">Pengali Modalitas</Label>
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
                {PRICING_MODALITIES.map((m) => (
                  <div key={m} className="space-y-1.5">
                    <span className="text-xs font-bold uppercase text-muted-foreground">{m}</span>
                    <Input
                      type="number"
                      step="0.1"
                      min={0}
                      aria-label={`Pengali ${m}`}
                      value={form.multipliers[m]}
                      onChange={(e) => setForm((s) => ({ ...s, multipliers: { ...s.multipliers, [m]: e.target.value } }))}
                    />
                  </div>
                ))}
              </div>
            </div>

            <div className="border-2 border-foreground/20 p-3 bg-muted/40">
              <p className="text-xs font-black uppercase text-muted-foreground mb-2">Pratinjau biaya (1.000 token)</p>
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-2 text-sm font-bold">
                {PRICING_MODALITIES.map((m) => (
                  <span key={m}>
                    {m}: {formatIDR(costIDR(numeric.base, form.price_unit, numeric.markup, numeric.percent, Number(form.multipliers[m]) || 0))}
                    {' '}
                    <span className={numeric.base >= 0 && costIDR(numeric.base, form.price_unit, numeric.markup, numeric.percent, Number(form.multipliers[m]) || 0) - numeric.providerCost >= 0 ? 'text-emerald-600' : 'text-destructive'}>
                      (margin {formatIDR(costIDR(numeric.base, form.price_unit, numeric.markup, numeric.percent, Number(form.multipliers[m]) || 0) - numeric.providerCost)})
                    </span>
                  </span>
                ))}
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              Batal
            </Button>
            <Button onClick={handleSave} disabled={saving}>
              {saving ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : <Save className="w-4 h-4 mr-2" />}
              Simpan
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null)
        }}
        title="Hapus Aturan Harga"
        message={`Hapus aturan "${deleteTarget?.rule_name}"? Tindakan ini tidak dapat dibatalkan.`}
        confirmText="Hapus"
        loading={deleteRule.isPending}
        onConfirm={handleDelete}
      />
    </div>
  )
}
