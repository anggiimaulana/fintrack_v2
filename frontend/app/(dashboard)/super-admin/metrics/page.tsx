'use client'

import * as React from 'react'
import { ListChecks, Plus, AlertTriangle } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from '@/components/ui/dialog'
import { useMetricDefinitions, useResolvers } from '@/hooks/useConfigCatalog'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import api from '@/lib/api'
import { toast } from 'sonner'
import { apiErrorMessage } from '@/lib/api-error'
import { configCatalogKeys } from '@/hooks/useConfigCatalog'

/**
 * Metric catalog editor (P-13).
 *
 * The catalog is DATA: adding a metric here makes it available in the plan
 * editor's limit dropdown WITHOUT a deploy. The resolver (how usage is
 * computed) is chosen from the capabilities the backend advertises.
 */
export default function MetricsPage() {
  const { metrics, isLoading } = useMetricDefinitions()
  const { resolvers } = useResolvers()
  const queryClient = useQueryClient()

  const [open, setOpen] = React.useState(false)
  const [form, setForm] = React.useState({
    key: '', label: '', unit: '', description: '', resolver_key: '', is_enforced: true,
  })

  const create = useMutation({
    mutationFn: async () => {
      await api.post('/admin/metric-definitions', {
        key: form.key.trim(),
        label: form.label.trim(),
        unit: form.unit.trim(),
        description: form.description.trim(),
        resolver_key: form.is_enforced ? form.resolver_key : null,
        is_enforced: form.is_enforced,
      })
    },
    onSuccess: () => {
      toast.success('Metrik berhasil ditambahkan.')
      setOpen(false)
      setForm({ key: '', label: '', unit: '', description: '', resolver_key: '', is_enforced: true })
      queryClient.invalidateQueries({ queryKey: configCatalogKeys.metrics })
    },
    onError: (e: unknown) => {
      toast.error(apiErrorMessage(e, 'Gagal menambahkan metrik.'))
    },
  })

  const remove = useMutation({
    mutationFn: async (key: string) => {
      await api.delete(`/admin/metric-definitions/${key}`)
    },
    onSuccess: () => {
      toast.success('Metrik dinonaktifkan.')
      queryClient.invalidateQueries({ queryKey: configCatalogKeys.metrics })
    },
    onError: (e: unknown) => {
      toast.error(apiErrorMessage(e, 'Gagal menonaktifkan metrik.'))
    },
  })

  const valid = form.key.trim() && form.label.trim() && (!form.is_enforced || form.resolver_key)

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-black uppercase tracking-tight flex items-center gap-2">
            <ListChecks className="w-6 h-6 text-primary" /> Katalog Metrik
          </h1>
          <p className="text-sm text-muted-foreground font-bold">
            Metrik adalah data — menambah di sini membuatnya tersedia di editor plan tanpa deploy.
          </p>
        </div>
        <Button onClick={() => setOpen(true)}>
          <Plus className="w-4 h-4 mr-2" /> Metrik Baru
        </Button>
      </div>

      <Card className="border-2 rounded-md border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
        <CardHeader>
          <CardTitle className="text-base font-black">Metrik terdaftar</CardTitle>
          <CardDescription>{isLoading ? 'Memuat…' : `${metrics.length} metrik`}</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b-2 border-foreground text-left text-xs font-black uppercase">
                  <th className="py-2 pr-3">Key</th>
                  <th className="py-2 pr-3">Label</th>
                  <th className="py-2 pr-3">Satuan</th>
                  <th className="py-2 pr-3">Ditegakkan</th>
                  <th className="py-2 pr-3">Resolver</th>
                  <th className="py-2 pr-3">Aksi</th>
                </tr>
              </thead>
              <tbody>
                {metrics.map((m) => (
                  <tr key={m.key} className="border-b border-border/50">
                    <td className="py-2 pr-3 font-mono text-xs">{m.key}</td>
                    <td className="py-2 pr-3 font-bold">{m.label}</td>
                    <td className="py-2 pr-3">{m.unit}</td>
                    <td className="py-2 pr-3">
                      {m.is_enforced ? <Badge>ya</Badge> : <Badge variant="outline">tidak</Badge>}
                    </td>
                    <td className="py-2 pr-3 font-mono text-xs text-muted-foreground">
                      {m.resolver_key || '—'}
                    </td>
                    <td className="py-2 pr-3">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-destructive"
                        disabled={remove.isPending}
                        onClick={() => remove.mutate(m.key)}
                        aria-label={`Nonaktifkan metrik ${m.label}`}
                      >
                        Nonaktifkan
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {metrics.some((m) => !m.is_enforced) && (
            <p className="mt-3 text-xs font-bold text-muted-foreground flex items-center gap-1.5">
              <AlertTriangle className="w-3.5 h-3.5" />
              Metrik tanpa tanda &quot;ditegakkan&quot; hanya informasional — backend tidak menghitungnya.
            </p>
          )}
        </CardContent>
      </Card>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="sm:max-w-lg border-2 rounded-lg border-foreground">
          <DialogHeader>
            <DialogTitle className="font-black">Metrik Baru</DialogTitle>
            <DialogDescription>
              Resolver adalah kemampuan backend. Pilih dari daftar — menambah kemampuan baru butuh deploy.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3">
            <Field id="m-key" label="Key" value={form.key} placeholder="budgets_created"
              onChange={(v) => setForm({ ...form, key: v })} />
            <Field id="m-label" label="Label" value={form.label} placeholder="Anggaran / siklus"
              onChange={(v) => setForm({ ...form, label: v })} />
            <Field id="m-unit" label="Satuan" value={form.unit} placeholder="anggaran"
              onChange={(v) => setForm({ ...form, unit: v })} />
            <Field id="m-desc" label="Deskripsi" value={form.description} placeholder="Jumlah anggaran yang dapat dibuat."
              onChange={(v) => setForm({ ...form, description: v })} />
            <div className="space-y-1.5">
              <Label htmlFor="m-enforced">Ditegakkan backend</Label>
              <select id="m-enforced" value={form.is_enforced ? 'yes' : 'no'}
                onChange={(e) => setForm({ ...form, is_enforced: e.target.value === 'yes' })}
                className="h-9 w-full rounded-md border-2 border-foreground bg-background px-3 text-sm font-medium">
                <option value="yes">Ya (butuh resolver)</option>
                <option value="no">Tidak (informasional)</option>
              </select>
            </div>
            {form.is_enforced && (
              <div className="space-y-1.5">
                <Label htmlFor="m-resolver">Resolver</Label>
                <select id="m-resolver" value={form.resolver_key}
                  onChange={(e) => setForm({ ...form, resolver_key: e.target.value })}
                  className="h-9 w-full rounded-md border-2 border-foreground bg-background px-3 text-sm font-medium">
                  <option value="">— pilih resolver —</option>
                  {resolvers.map((r) => (
                    <option key={r.key} value={r.key}>{r.label} ({r.key})</option>
                  ))}
                </select>
              </div>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setOpen(false)}>Batal</Button>
            <Button onClick={() => create.mutate()} disabled={!valid || create.isPending}>
              {create.isPending ? 'Menyimpan…' : 'Simpan'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function Field({ id, label, value, placeholder, onChange }: {
  id: string; label: string; value: string; placeholder?: string; onChange: (v: string) => void
}) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>{label}</Label>
      <Input id={id} value={value} placeholder={placeholder} onChange={(e) => onChange(e.target.value)} />
    </div>
  )
}
