'use client'

import * as React from 'react'
import { Check, Minus, Infinity as InfinityIcon, CreditCard, ArrowDownCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Money } from '@/components/shared/Money'
import type { PublicPlan } from '@/hooks/use-billing'

/** Human labels for seeded feature keys; unknown keys fall back to the raw key. */
const FEATURE_LABELS: Record<string, string> = {
  ai_chat: 'Asisten AI (chat)',
  ocr_receipt: 'Pindai struk (OCR)',
  voice_input: 'Input suara',
  rag_search: 'Pencarian cerdas (RAG)',
  reports_export: 'Ekspor laporan',
  advanced_budgeting: 'Anggaran lanjutan',
  api_access: 'Akses API',
  bank_sync: 'Sinkronisasi bank',
}

/** Human labels for seeded limit keys. */
const LIMIT_LABELS: Record<string, string> = {
  ai_tokens_per_cycle: 'Token AI / siklus',
  ai_requests_per_cycle: 'Permintaan AI / siklus',
  max_transactions: 'Transaksi',
  max_categories: 'Kategori',
  max_wallets: 'Dompet',
  ocr_scans_per_cycle: 'Pindai struk / siklus',
}

function formatLimit(value: number): string {
  if (value < 0) return 'Tak terbatas'
  return value.toLocaleString('id-ID')
}

export interface PlanCardProps {
  plan: PublicPlan
  /** The user's current plan code ("" when none). */
  currentPlanCode: string
  /** Rank of the user's plan, used to decide upgrade vs downgrade affordance. */
  currentSortOrder: number
  onUpgrade: (planCode: string) => void
  onDowngrade: (planCode: string) => void
  isBusy: boolean
  /** A downgrade already scheduled to this plan. */
  scheduledPlanCode?: string
}

/**
 * A plan card driven entirely by the backend catalog (price, features, limits).
 * Shows "Upgrade" for higher tiers, "Turunkan paket" for lower tiers, and marks
 * the active plan. No price or feature is hardcoded in the UI.
 */
export function PlanCard({
  plan,
  currentPlanCode,
  currentSortOrder,
  onUpgrade,
  onDowngrade,
  isBusy,
  scheduledPlanCode,
}: PlanCardProps) {
  const isCurrent = plan.code === currentPlanCode
  const isScheduled = plan.code === scheduledPlanCode
  const isDowngrade = currentPlanCode !== '' && plan.sort_order < currentSortOrder
  const isFree = plan.price_monthly_idr === 0

  const featureEntries = Object.entries(plan.features).sort(([a], [b]) => a.localeCompare(b))
  const limitEntries = Object.entries(plan.limits).sort(([a], [b]) => a.localeCompare(b))

  return (
    <Card
      className={`flex flex-col justify-between border-2 ${
        isCurrent
          ? 'border-primary ring-2 ring-primary/20'
          : 'border-foreground shadow-[4px_4px_0px_0px] shadow-foreground'
      }`}
    >
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg font-black">{plan.name}</CardTitle>
          {isCurrent ? (
            <Badge variant="default">Paket Anda</Badge>
          ) : isScheduled ? (
            <Badge variant="secondary">Dijadwalkan</Badge>
          ) : (
            <Badge variant="outline">{isFree ? 'Gratis' : 'Berbayar'}</Badge>
          )}
        </div>
        {plan.description && <CardDescription>{plan.description}</CardDescription>}
        <div className="pt-2">
          <Money value={plan.price_monthly_idr} className="text-2xl font-black" />
          <span className="text-xs font-bold text-muted-foreground">
            {' '}
            / {plan.billing_period_days} hari
          </span>
        </div>
      </CardHeader>

      <CardContent className="space-y-3 text-sm font-medium">
        <ul className="space-y-1.5">
          {featureEntries.map(([key, enabled]) => (
            <li key={key} className="flex items-center gap-2">
              {enabled ? (
                <Check className="h-4 w-4 text-emerald-600 dark:text-emerald-400" aria-hidden="true" />
              ) : (
                <Minus className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
              )}
              <span className={enabled ? '' : 'text-muted-foreground line-through'}>
                {FEATURE_LABELS[key] ?? key}
              </span>
            </li>
          ))}
        </ul>
        {limitEntries.length > 0 && (
          <div className="border-t-2 border-foreground/20 pt-2">
            <p className="mb-1 text-xs font-black uppercase tracking-wide text-muted-foreground">Kuota</p>
            <ul className="space-y-1">
              {limitEntries.map(([key, value]) => (
                <li key={key} className="flex items-center justify-between gap-2 text-xs">
                  <span className="text-muted-foreground">{LIMIT_LABELS[key] ?? key}</span>
                  <span className="flex items-center gap-1 font-mono font-bold">
                    {value < 0 && <InfinityIcon className="h-3 w-3" aria-hidden="true" />}
                    {formatLimit(value)}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </CardContent>

      <CardFooter>
        {isCurrent ? (
          <Button className="w-full" variant="outline" disabled>
            Paket Sedang Aktif
          </Button>
        ) : isDowngrade ? (
          <Button
            className="w-full"
            variant="outline"
            disabled={isBusy || isScheduled}
            onClick={() => onDowngrade(plan.code)}
          >
            <ArrowDownCircle className="mr-2 h-4 w-4" />
            {isScheduled ? 'Penurunan dijadwalkan' : 'Turunkan ke Paket Ini'}
          </Button>
        ) : (
          <Button className="w-full" disabled={isBusy} onClick={() => onUpgrade(plan.code)}>
            <CreditCard className="mr-2 h-4 w-4" />
            {isBusy ? 'Memproses…' : `Tingkatkan ke ${plan.name}`}
          </Button>
        )}
      </CardFooter>
    </Card>
  )
}
