'use client'

import { useState } from 'react'
import { Check, Minus, Infinity as InfinityIcon, ShieldCheck, Sparkles, ExternalLink, CreditCard } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle, CardFooter } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { useMutation } from '@tanstack/react-query'
import { toast } from 'sonner'
import api from '@/lib/api'
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

function getErrorMessage(err: unknown): string {
  if (err && typeof err === 'object' && 'response' in err) {
    const res = err.response
    if (res && typeof res === 'object' && 'data' in res) {
      const data = res.data
      if (data && typeof data === 'object' && 'message' in data && typeof data.message === 'string') {
        return data.message
      }
    }
  }
  return 'Gagal membuat sesi pembayaran'
}

function formatLimit(value: number | undefined): string {
  if (value === undefined) return '—'
  if (value < 0) return 'Tak terbatas'
  return value.toLocaleString('id-ID')
}

export default function SubscriptionPage() {
  const { entitlements, isLoading, isError, planName, planCode, status } = useEntitlements()
  const [checkoutModalOpen, setCheckoutModalOpen] = useState(false)
  const [checkoutData, setCheckoutData] = useState<{
    invoice?: { invoice_number?: string; total_idr?: number }
    gateway?: string
    redirect_url?: string
  } | null>(null)

  const checkoutMutation = useMutation({
    mutationFn: async (targetPlanCode: string) => {
      const res = await api.post('/payments/checkout', { plan_code: targetPlanCode })
      return res.data?.data
    },
    onSuccess: (data) => {
      setCheckoutData(data)
      setCheckoutModalOpen(true)
      if (data?.redirect_url) {
        window.open(data.redirect_url, '_blank')
      }
      toast.success('Sesi pembayaran berhasil dibuat!')
    },
    onError: (err: unknown) => {
      toast.error(getErrorMessage(err))
    },
  })

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

      {/* Pilihan Paket & Upgrade Langganan */}
      <div className="space-y-4 pt-4">
        <div>
          <h2 className="text-xl font-bold tracking-tight flex items-center gap-2">
            <Sparkles className="h-5 w-5 text-primary" />
            Tingkatkan Paket Langganan
          </h2>
          <p className="text-sm text-muted-foreground">
            Pilih paket yang sesuai dengan kebutuhan pencatatan dan kapasitas kecerdasan buatan Anda.
          </p>
        </div>

        <div className="grid gap-6 md:grid-cols-2">
          {/* Paket Pro */}
          <Card className={`border-2 flex flex-col justify-between ${planCode === 'pro' ? 'border-primary ring-2 ring-primary/20' : 'border-foreground shadow-[4px_4px_0px_0px] shadow-foreground'}`}>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-lg font-black">Paket Pro</CardTitle>
                <Badge variant={planCode === 'pro' ? 'default' : 'outline'}>
                  {planCode === 'pro' ? 'Paket Anda' : 'Populer'}
                </Badge>
              </div>
              <CardDescription>Ideal untuk profesional dan pengguna aktif.</CardDescription>
              <div className="pt-2">
                <span className="text-2xl font-black">Rp 39.000</span>
                <span className="text-xs font-bold text-muted-foreground"> / bulan</span>
              </div>
            </CardHeader>
            <CardContent className="space-y-2 text-sm font-medium">
              <div className="flex items-center gap-2"><Check className="h-4 w-4 text-emerald-600" /> 150.000 Token AI / siklus</div>
              <div className="flex items-center gap-2"><Check className="h-4 w-4 text-emerald-600" /> 5.000 Transaksi / bulan</div>
              <div className="flex items-center gap-2"><Check className="h-4 w-4 text-emerald-600" /> 100 Pemindaian Struk (OCR)</div>
              <div className="flex items-center gap-2"><Check className="h-4 w-4 text-emerald-600" /> Input Suara &amp; Ekspor Laporan</div>
              <div className="flex items-center gap-2"><Check className="h-4 w-4 text-emerald-600" /> Pencarian Cerdas (RAG 768d)</div>
            </CardContent>
            <CardFooter>
              <Button
                className="w-full"
                disabled={planCode === 'pro' || checkoutMutation.isPending}
                onClick={() => checkoutMutation.mutate('pro')}
              >
                <CreditCard className="h-4 w-4 mr-2" />
                {planCode === 'pro' ? 'Paket Sedang Aktif' : checkoutMutation.isPending ? 'Memproses...' : 'Tingkatkan ke Pro'}
              </Button>
            </CardFooter>
          </Card>

          {/* Paket Business */}
          <Card className={`border-2 flex flex-col justify-between ${planCode === 'business' ? 'border-primary ring-2 ring-primary/20' : 'border-foreground shadow-[4px_4px_0px_0px] shadow-foreground'}`}>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-lg font-black">Paket Business</CardTitle>
                <Badge variant={planCode === 'business' ? 'default' : 'outline'}>
                  {planCode === 'business' ? 'Paket Anda' : 'Tanpa Batas'}
                </Badge>
              </div>
              <CardDescription>Untuk bisnis dan pengelolaan aset intensif.</CardDescription>
              <div className="pt-2">
                <span className="text-2xl font-black">Rp 99.000</span>
                <span className="text-xs font-bold text-muted-foreground"> / bulan</span>
              </div>
            </CardHeader>
            <CardContent className="space-y-2 text-sm font-medium">
              <div className="flex items-center gap-2"><Check className="h-4 w-4 text-emerald-600" /> 1.000.000 Token AI / siklus</div>
              <div className="flex items-center gap-2"><Check className="h-4 w-4 text-emerald-600" /> Transaksi Tanpa Batas</div>
              <div className="flex items-center gap-2"><Check className="h-4 w-4 text-emerald-600" /> Pemindaian Struk (OCR) Tanpa Batas</div>
              <div className="flex items-center gap-2"><Check className="h-4 w-4 text-emerald-600" /> Prioritas Pemrosesan AI Gateway</div>
              <div className="flex items-center gap-2"><Check className="h-4 w-4 text-emerald-600" /> Dukungan Eksklusif &amp; Analisis Mendalam</div>
            </CardContent>
            <CardFooter>
              <Button
                className="w-full"
                variant={planCode === 'business' ? 'outline' : 'default'}
                disabled={planCode === 'business' || checkoutMutation.isPending}
                onClick={() => checkoutMutation.mutate('business')}
              >
                <CreditCard className="h-4 w-4 mr-2" />
                {planCode === 'business' ? 'Paket Sedang Aktif' : checkoutMutation.isPending ? 'Memproses...' : 'Tingkatkan ke Business'}
              </Button>
            </CardFooter>
          </Card>
        </div>
      </div>

      {/* Modal Sesi Pembayaran */}
      <Dialog open={checkoutModalOpen} onOpenChange={setCheckoutModalOpen}>
        <DialogContent className="border-2 border-foreground shadow-[6px_6px_0px_0px] shadow-foreground bg-card sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-lg font-black flex items-center gap-2">
              <CreditCard className="w-5 h-5 text-primary" /> Sesi Pembayaran Dibuat
            </DialogTitle>
            <DialogDescription className="text-xs font-bold text-muted-foreground mt-2">
              Faktur telah diterbitkan. Silakan selesaikan pembayaran untuk mengaktifkan paket secara instan.
            </DialogDescription>
          </DialogHeader>
          {checkoutData && (
            <div className="space-y-3 py-2 text-sm">
              <div className="flex justify-between border-b pb-2">
                <span className="text-muted-foreground font-medium">Nomor Faktur</span>
                <span className="font-mono font-bold">{checkoutData.invoice?.invoice_number || '—'}</span>
              </div>
              <div className="flex justify-between border-b pb-2">
                <span className="text-muted-foreground font-medium">Total Pembayaran</span>
                <span className="font-black text-primary">Rp {(checkoutData.invoice?.total_idr ?? 0).toLocaleString('id-ID')}</span>
              </div>
              <div className="flex justify-between border-b pb-2">
                <span className="text-muted-foreground font-medium">Metode Gateway</span>
                <span className="font-bold uppercase">{checkoutData.gateway || '—'}</span>
              </div>
            </div>
          )}
          <DialogFooter className="gap-2 sm:gap-0 mt-4">
            <Button variant="outline" onClick={() => setCheckoutModalOpen(false)}>
              Tutup
            </Button>
            {checkoutData?.redirect_url && (
              <Button asChild>
                <a href={checkoutData.redirect_url} target="_blank" rel="noopener noreferrer">
                  <ExternalLink className="h-4 w-4 mr-2" /> Buka Laman Pembayaran
                </a>
              </Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
