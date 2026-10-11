import { AlertTriangle, ArrowRight } from 'lucide-react'
import Link from 'next/link'
import { Button } from '@/components/ui/button'

interface CircuitBreakerBannerProps {
  count: number
}

export function CircuitBreakerBanner({ count }: CircuitBreakerBannerProps) {
  if (count <= 0) return null

  return (
    <div
      role="alert"
      className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-4 border-2 border-destructive bg-destructive/15 text-destructive font-medium rounded-none shadow-[4px_4px_0px_0px] shadow-destructive"
    >
      <div className="flex items-center gap-3">
        <AlertTriangle className="h-5 w-5 shrink-0" />
        <div>
          <span className="font-bold">Peringatan: </span>
          <span>
            {count} circuit breaker sedang dalam status terbuka. Beberapa request AI gateway dialihkan atau gagal.
          </span>
        </div>
      </div>
      <Button
        asChild
        variant="destructive"
        size="sm"
        className="shrink-0 self-start sm:self-auto gap-1 font-bold"
      >
        <Link href="/super-admin/ai-gateway">
          Periksa AI Gateway
          <ArrowRight className="h-4 w-4" />
        </Link>
      </Button>
    </div>
  )
}
