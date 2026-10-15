import { Suspense } from 'react'
import Link from 'next/link'
import { CheckSquare, CreditCard } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { PaymentHistoryTable } from '@/components/super-admin/payments/PaymentHistoryTable'

export default function SuperAdminPaymentsPage() {
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

      {/* PaymentHistoryTable reads useSearchParams, so it needs a Suspense
          boundary for the route to stay statically prerenderable. */}
      <Suspense fallback={<div className="h-40 animate-pulse bg-muted border-2 rounded-md border-foreground" />}>
        <PaymentHistoryTable />
      </Suspense>
    </div>
  )
}
