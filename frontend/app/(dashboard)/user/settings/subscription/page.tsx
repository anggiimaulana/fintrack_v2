'use client'

import Link from 'next/link'
import { Check, Minus, Infinity as InfinityIcon, ShieldCheck } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useEntitlements } from '@/hooks/use-entitlements'

/**
 * Read-only view of the user's plan, features and limits.
 *
 * This page is the destination behind the backend's `upgrade_url`, so every
 * 403/402 the API returns points somewhere real. It deliberately has no purchase
 * flow — billing UX is owned by another phase; this only reports what the
 * entitlement resolver currently grants.
 */

// Human labels for the seeded feature keys. Unknown keys fall back to the raw key.
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

// Human labels for the seeded limit keys.
const LIMIT_LABELS: Record<string, string> = {
  ai_tokens_per_cycle: 'Token AI per siklus',
  ai_requests_per_cycle: 'Permintaan AI per siklus',
  max_transactions: 'Maksimum transaksi',
  max_categories: 'Maksimum kategori',
  max_wallets: 'Maksimum dompet',
  ocr_scans_per_cycle: 'Pindai struk per siklus',
}

// Status labels for the subscription lifecycle (plus the `none` sentinel that
// means "no subscription row — on the default plan").
const STATUS_LABELS: Record<string, string> = {
  active: 'Aktif',
  trialing: 'Uji coba',
  past_due: 'Menunggu pembayaran',
  grace: 'Masa tenggang',
  canceled: 'Dibatalkan',
  expired: 'Kedaluwarsa',
  none: 'Belum berlangganan',
}

function formatLimit(value: number | undefined): string {
  if (value === undefined) return '—'
  if (value < 0) return 'Tak terbatas'
  return value.toLocaleString('id-ID')
}

export default function SubscriptionPage() {
  const { entitlements, isLoading, isError, planName, planCode, status } = useEntitlements()

  const features = Object.entries(entitlements?.features ?? {}).sort(([a], [b]) => a.localeCompare(b))
  const limits = Object.entries(entitlements?.limits ?? {}).sort(([a], [b]) => a.localeCompare(b))

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Langganan &amp; Paket</h1>
        <p className="text-sm text-muted-foreground">
          Ringkasan fitur dan batas yang berlaku untuk akun Anda saat ini.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <ShieldCheck className="h-5 w-5" />
            Paket Anda
          </CardTitle>
          <CardDescription>
            Perubahan paket oleh admin berlaku seketika, tanpa perlu masuk ulang.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-6 w-40" />
              <Skeleton className="h-4 w-64" />
            </div>
          ) : isError ? (
            <p className="text-sm text-muted-foreground">
              Gagal memuat data paket. Coba muat ulang halaman ini.
            </p>
          ) : (
            <div className="flex flex-wrap items-center gap-3">
              <span className="text-lg font-bold">{planName || '—'}</span>
              <Badge variant="outline" className="font-mono text-xs">
                {planCode || '—'}
              </Badge>
              <Badge variant={status === 'active' ? 'default' : 'secondary'}>
                {STATUS_LABELS[status] ?? status ?? '—'}
              </Badge>
            </div>
          )}
        </CardContent>
      </Card>

      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Fitur</CardTitle>
            <CardDescription>Fitur yang aktif pada paket Anda.</CardDescription>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="space-y-2">
                {Array.from({ length: 4 }).map((_, i) => (
                  <Skeleton key={i} className="h-5 w-full" />
                ))}
              </div>
            ) : features.length === 0 ? (
              <p className="text-sm text-muted-foreground">Tidak ada data fitur.</p>
            ) : (
              <ul className="space-y-2">
                {features.map(([key, enabled]) => (
                  <li key={key} className="flex items-center justify-between gap-3 text-sm">
                    <span className="flex items-center gap-2">
                      {enabled ? (
                        <Check className="h-4 w-4 text-emerald-600 dark:text-emerald-400" />
                      ) : (
                        <Minus className="h-4 w-4 text-muted-foreground" />
                      )}
                      {FEATURE_LABELS[key] ?? key}
                    </span>
                    <span
                      className={
                        enabled
                          ? 'text-xs font-bold text-emerald-600 dark:text-emerald-400'
                          : 'text-xs font-bold text-muted-foreground'
                      }
                    >
                      {enabled ? 'Aktif' : 'Tidak tersedia'}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Batas penggunaan</CardTitle>
            <CardDescription>Kuota numerik pada paket Anda.</CardDescription>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="space-y-2">
                {Array.from({ length: 4 }).map((_, i) => (
                  <Skeleton key={i} className="h-5 w-full" />
                ))}
              </div>
            ) : limits.length === 0 ? (
              <p className="text-sm text-muted-foreground">Tidak ada data batas.</p>
            ) : (
              <ul className="space-y-2">
                {limits.map(([key, value]) => (
                  <li key={key} className="flex items-center justify-between gap-3 text-sm">
                    <span>{LIMIT_LABELS[key] ?? key}</span>
                    <span className="flex items-center gap-1 font-mono text-xs font-bold">
                      {value < 0 && <InfinityIcon className="h-3.5 w-3.5" />}
                      {formatLimit(value)}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Butuh fitur lain?</CardTitle>
          <CardDescription>
            Peningkatan paket saat ini dilakukan melalui tim kami. Hubungi admin untuk mengaktifkan
            paket yang lebih tinggi.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button variant="outline" asChild>
            <Link href="/user/profile">Hubungi dukungan</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
