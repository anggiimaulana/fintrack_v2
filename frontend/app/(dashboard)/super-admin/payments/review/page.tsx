import Link from 'next/link'
import { ArrowLeft, Clock } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { PaymentReviewQueue } from '@/components/super-admin/payments/PaymentReviewQueue'

export default function SuperAdminPaymentReviewPage() {
  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="sm" asChild className="-ml-2 h-8 px-2 text-muted-foreground">
              <Link href="/super-admin/payments">
                <ArrowLeft className="h-4 w-4 mr-1" />
                Kembali ke Riwayat
              </Link>
            </Button>
          </div>
          <h1 className="mt-1 font-heading text-2xl font-bold tracking-tight text-foreground flex items-center gap-2">
            <Clock className="h-6 w-6 text-amber-500" />
            Antrean Verifikasi Pembayaran
          </h1>
          <p className="text-xs text-muted-foreground">
            Verifikasi transfer manual pelanggan berdasarkan bukti bayar (FIFO: terlama diproses terlebih dahulu).
          </p>
        </div>
      </div>

      <PaymentReviewQueue />
    </div>
  )
}
