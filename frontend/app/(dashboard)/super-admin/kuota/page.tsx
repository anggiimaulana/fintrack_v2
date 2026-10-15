'use client'

import * as React from 'react'
import { Calculator, AlertTriangle, TrendingUp } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useQuotaEstimate } from '@/hooks/useQuotaEstimate'
import { useSuperAdminPlans } from '@/hooks/useSuperAdminPlans'

/**
 * Quota calculator (P-9/P-13).
 *
 * Answers the operator's question directly: "given this quota and ratio, how
 * many chats does a user get, and what does each one cost me?" Everything is
 * read from the backend estimator, so the numbers always match what the ledger
 * will actually charge.
 */

const idr = (n: number) =>
  new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(n)

export default function QuotaCalculatorPage() {
  const { plans } = useSuperAdminPlans()
  const estimate = useQuotaEstimate()

  const [planCode, setPlanCode] = React.useState<string>('')
  const [quota, setQuota] = React.useState<number>(5000)
  const [ratio, setRatio] = React.useState<number>(5)
  const [avgTokens, setAvgTokens] = React.useState<number>(9945)
  const [voiceSec, setVoiceSec] = React.useState<number>(15)
  // Prefill from the selected plan so the operator starts from reality.
  const onPlanChange = (code: string) => {
    setPlanCode(code)
    const p = plans.find((x) => x.code === code)
    if (p) {
      setQuota(p.limits?.['ai_tokens_per_cycle'] ?? quota)
      setRatio(p.token_quota_ratio ?? 1)
    }
  }

  // Recompute whenever an input changes. The estimator is read-only, so this is
  // safe to fire on every keystroke (it is debounced by React's batching).
  const run = estimate.mutate
  React.useEffect(() => {
    if (quota < 1 || ratio < 1) return
    run({
      token_quota_ratio: ratio,
      ai_tokens_per_cycle: quota,
      avg_tokens_per_chat: avgTokens,
      voice_seconds: voiceSec,
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [quota, ratio, avgTokens, voiceSec])

  const data = estimate.data

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-black uppercase tracking-tight flex items-center gap-2">
          <Calculator className="w-6 h-6 text-primary" /> Kalkulator Kuota & Biaya
        </h1>
        <p className="text-sm text-muted-foreground font-bold">
          Simulasikan dampak kuota dan rasio sebelum menyimpannya.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Inputs */}
        <Card className="border-2 rounded-md border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
          <CardHeader>
            <CardTitle className="text-base font-black">Input</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="calc-plan">Plan</Label>
              <select
                id="calc-plan"
                value={planCode}
                onChange={(e) => onPlanChange(e.target.value)}
                className="h-9 w-full rounded-md border-2 border-foreground bg-background px-3 text-sm font-medium"
              >
                <option value="">— pilih plan (opsional) —</option>
                {plans.map((p) => (
                  <option key={p.code} value={p.code}>
                    {p.name}
                  </option>
                ))}
              </select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="calc-quota">Kuota per siklus (unit)</Label>
              <Input id="calc-quota" type="number" min={1} value={quota}
                onChange={(e) => setQuota(Math.max(1, parseInt(e.target.value, 10) || 1))} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="calc-ratio">Rasio token → kuota</Label>
              <Input id="calc-ratio" type="number" min={1} max={1000} value={ratio}
                onChange={(e) => setRatio(Math.max(1, parseInt(e.target.value, 10) || 1))} />
              <p className="text-xs text-muted-foreground">Rasio 1 = 1 token : 1 kuota. Rasio 5 = lebih hemat 5×.</p>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="calc-avg">Rata-rata token per chat</Label>
              <Input id="calc-avg" type="number" min={1} value={avgTokens}
                onChange={(e) => setAvgTokens(Math.max(1, parseInt(e.target.value, 10) || 1))} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="calc-voice">Simulasi durasi suara (detik)</Label>
              <Input id="calc-voice" type="number" min={1} max={300} value={voiceSec}
                onChange={(e) => setVoiceSec(Math.max(1, parseInt(e.target.value, 10) || 1))} />
              <p className="text-xs text-muted-foreground">Perkiraan durasi 1 panggilan suara (rata-rata 15s).</p>
            </div>
          </CardContent>
        </Card>

        {/* Results */}
        <Card className="border-2 rounded-md border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
          <CardHeader>
            <CardTitle className="text-base font-black flex items-center gap-2">
              <TrendingUp className="w-4 h-4 text-primary" /> Hasil
            </CardTitle>
            <CardDescription aria-live="polite">Perhitungan diperbarui otomatis.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {!data ? (
              <p className="text-sm text-muted-foreground">Menghitung…</p>
            ) : (
              <>
                <div className="grid grid-cols-2 gap-3 text-sm">
                  <Stat label="Kuota per chat" value={`${data.quota.unit_per_chat.toLocaleString('id-ID')} unit`} />
                  <Stat label="Chat per siklus" value={`~${data.quota.chats_per_cycle_exact}`} />
                  <Stat label="Sisa akhir siklus" value={`${data.quota.leftover_units.toLocaleString('id-ID')} unit`} />
                  <Stat label="Mode pembulatan" value={data.input.rounding_mode} />
                </div>

                <div className="border-t-2 rounded-md border-foreground/20 pt-3 grid grid-cols-2 gap-3 text-sm">
                  <Stat label="Biaya jual / chat" value={idr(data.cost.sell_per_chat_idr)} />
                  <Stat label="Biaya provider / chat" value={idr(data.cost.provider_cost_per_chat_idr)} />
                  <Stat label="Margin / chat" value={`${idr(data.cost.margin_per_chat_idr)} (${data.cost.margin_percent}%)`} />
                  <Stat label="Biaya siklus penuh" value={idr(data.cost.cost_per_full_cycle_idr)} />
                </div>

                <div className="border-t-2 rounded-md border-foreground/20 pt-3 text-xs font-bold text-muted-foreground">
                  Tanpa rasio: {data.comparison.without_ratio.chats_per_cycle} chat → dengan rasio:{' '}
                  <span className="text-emerald-600">{data.comparison.with_ratio.chats_per_cycle} chat</span>
                </div>

                {data.modalities?.voice && data.modalities?.vision && (
                  <div className="border-t-2 rounded-md border-foreground/20 pt-3 space-y-2">
                    <p className="text-[10px] font-black uppercase tracking-wide text-muted-foreground">Simulasi Modalitas Lain</p>
                    <div className="grid grid-cols-2 gap-3 text-sm">
                      <Stat
                        label={`Suara (${data.modalities.voice.avg_seconds}s)`}
                        value={`~${data.modalities.voice.calls_per_cycle} panggilan (${data.modalities.voice.unit_per_call} unit)`}
                      />
                      <Stat
                        label="Scan Struk (Vision)"
                        value={`~${data.modalities.vision.calls_per_cycle} scan (${data.modalities.vision.unit_per_call} unit)`}
                      />
                    </div>
                  </div>
                )}

                {data.warnings.length > 0 && (
                  <div role="status" className="space-y-1.5 border-2 rounded-md border-amber-500 bg-amber-50 p-3">
                    {data.warnings.map((w, i) => (
                      <p key={i} className="text-xs font-bold text-amber-800 flex items-start gap-1.5">
                        <AlertTriangle className="w-3.5 h-3.5 shrink-0 mt-0.5" /> {w}
                      </p>
                    ))}
                  </div>
                )}
              </>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-[10px] font-black uppercase tracking-wide text-muted-foreground">{label}</p>
      <p className="font-mono font-bold">{value}</p>
    </div>
  )
}
