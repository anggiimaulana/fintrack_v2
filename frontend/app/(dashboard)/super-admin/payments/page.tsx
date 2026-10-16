import { Suspense } from 'react'
import Link from 'next/link'
import { CheckSquare, CreditCard, BarChart3 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { PaymentHistoryTable } from '@/components/super-admin/payments/PaymentHistoryTable'
import { SmartInsightCard } from '@/components/shared/SmartInsightCard'
import { BarChartView } from '@/components/charts/BarChartView'

export default function SuperAdminPaymentsPage() {
  const funnelData = [
    { name: 'Dibuat', income: 150, expense: 0, savings: 0 },
    { name: 'Berhasil', income: 142, expense: 0, savings: 0 },
    { name: 'Review Manual', income: 0, expense: 0, savings: 8 },
    { name: 'Kedaluwarsa', income: 0, expense: 5, savings: 0 },
  ]

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="font-heading text-2xl font-bold tracking-tight text-foreground flex items-center gap-2">
            <CreditCard className="h-6 w-6 text-primary" />
            Riwayat Pembayaran
          </h1>
          <p className="text-xs text-muted-foreground">
            Semua transaksi pembayaran masuk dari gateway otomatis maupun transfer manual.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <Button asChild className="gap-2 font-bold" variant="default">
            <Link href="/super-admin/payments/review">
              <CheckSquare className="h-4 w-4" />
              Antrean Verifikasi
            </Link>
          </Button>
        </div>
      </div>

      <SmartInsightCard
        title="Insight Konversi & Funnel Pembayaran"
        summaryText="Tingkat keberhasilan pembayaran stabil pada level 96.8%. Transaksi didominasi pembayaran QRIS & Virtual Account instan."
        deltaPercent={3.1}
        deltaDirection="up"
        isPositiveDelta={true}
        metricsSummary={[
          { label: 'Success Rate', value: '96.8%' },
          { label: 'Rata-rata Nominal', value: 'Rp 175.000' },
          { label: 'Metode Dominan', value: 'QRIS' },
        ]}
        actionRecommendation={{
          text: 'Periksa antrean bukti transfer manual untuk memastikan aktivasi langganan pengguna tepat waktu.',
          actionLabel: 'Antrean Verifikasi',
          href: '/super-admin/payments/review',
        }}
      />

      {/* Spec 03 BarChartView for Payment Funnel */}
      <div className="border-2 rounded-md border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-5 space-y-3">
        <div className="flex items-center justify-between border-b-2 border-foreground pb-3">
          <div className="flex items-center gap-2">
            <BarChart3 className="w-5 h-5 text-primary" />
            <h3 className="text-sm font-black uppercase tracking-wide">
              Funnel Pembayaran & Status Penyelesaian
            </h3>
          </div>
          <span className="text-xs font-bold text-muted-foreground">Periode: 30 Hari Terakhir</span>
        </div>
        <div className="h-[220px] w-full">
          <BarChartView data={funnelData} />
        </div>
      </div>

      {/* PaymentHistoryTable reads useSearchParams, so it needs a Suspense
          boundary for the route to stay statically prerenderable. */}
      <Suspense fallback={<div className="h-40 animate-pulse bg-muted border-2 rounded-md border-foreground" />}>
        <PaymentHistoryTable />
      </Suspense>
    </div>
  )
}
