import Link from 'next/link'
import { Card, CardContent } from '@/components/ui/card'
import { Money } from '@/components/shared/Money'
import { Skeleton } from '@/components/ui/skeleton'
import {
  TrendingUp,
  TrendingDown,
  Minus,
  Users,
  Coins,
  CheckCircle2,
  Clock,
  Server,
  ShieldAlert,
  ChevronRight,
  DollarSign,
} from 'lucide-react'
import type { OverviewMetrics } from '@/hooks/useSuperAdminMetrics'

interface OverviewKpiCardsProps {
  overview?: OverviewMetrics
  isLoading?: boolean
}

function GrowthBadge({ value }: { value: number }) {
  if (value > 0) {
    return (
      <span className="inline-flex items-center gap-0.5 text-xs font-bold text-emerald-600 dark:text-emerald-400">
        <TrendingUp className="h-3.5 w-3.5" />
        +{value.toFixed(1)}%
      </span>
    )
  }
  if (value < 0) {
    return (
      <span className="inline-flex items-center gap-0.5 text-xs font-bold text-rose-600 dark:text-rose-400">
        <TrendingDown className="h-3.5 w-3.5" />
        {value.toFixed(1)}%
      </span>
    )
  }
  return (
    <span className="inline-flex items-center gap-0.5 text-xs font-bold text-muted-foreground">
      <Minus className="h-3.5 w-3.5" />
      0.0%
    </span>
  )
}

export function OverviewKpiCardsSkeleton() {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      {Array.from({ length: 7 }).map((_, i) => (
        <Card key={i} className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-4">
          <div className="flex items-center justify-between mb-2">
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-6 w-6 rounded-full" />
          </div>
          <Skeleton className="h-8 w-32 my-1" />
          <Skeleton className="h-4 w-20 mt-2" />
        </Card>
      ))}
    </div>
  )
}

export function OverviewKpiCards({ overview, isLoading }: OverviewKpiCardsProps) {
  if (isLoading || !overview) {
    return <OverviewKpiCardsSkeleton />
  }

  const {
    mrr_idr,
    mrr_growth_percentage,
    active_users_30d,
    active_users_growth_percentage,
    tokens_consumed_24h,
    tokens_growth_percentage,
    payment_success_rate,
    pending_manual_payments,
    ai_endpoints_healthy,
    ai_endpoints_total,
    circuit_breaker_open_count,
  } = overview

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      {/* 1. MRR */}
      <Card className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground">
        <CardContent className="p-4">
          <div className="flex items-center justify-between text-muted-foreground">
            <span className="text-xs font-bold uppercase tracking-wider">MRR</span>
            <DollarSign className="h-4 w-4" />
          </div>
          <div className="mt-2 text-2xl font-black">
            <Money value={mrr_idr} />
          </div>
          <div className="mt-2 flex items-center gap-1.5">
            <GrowthBadge value={mrr_growth_percentage} />
            <span className="text-xs text-muted-foreground">vs 30 hari lalu</span>
          </div>
        </CardContent>
      </Card>

      {/* 2. Pengguna Aktif 30d */}
      <Card className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground">
        <CardContent className="p-4">
          <div className="flex items-center justify-between text-muted-foreground">
            <span className="text-xs font-bold uppercase tracking-wider">Pengguna Aktif (30h)</span>
            <Users className="h-4 w-4" />
          </div>
          <div className="mt-2 text-2xl font-black tabular-nums">
            {active_users_30d.toLocaleString('id-ID')}
          </div>
          <div className="mt-2 flex items-center gap-1.5">
            <GrowthBadge value={active_users_growth_percentage} />
            <span className="text-xs text-muted-foreground">vs 30 hari lalu</span>
          </div>
        </CardContent>
      </Card>

      {/* 3. Konsumsi Token 24h */}
      <Card className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground">
        <CardContent className="p-4">
          <div className="flex items-center justify-between text-muted-foreground">
            <span className="text-xs font-bold uppercase tracking-wider">Token Digunakan (24j)</span>
            <Coins className="h-4 w-4" />
          </div>
          <div className="mt-2 text-2xl font-black tabular-nums">
            {tokens_consumed_24h.toLocaleString('id-ID')}
          </div>
          <div className="mt-2 flex items-center gap-1.5">
            <GrowthBadge value={tokens_growth_percentage} />
            <span className="text-xs text-muted-foreground">vs 24 jam lalu</span>
          </div>
        </CardContent>
      </Card>

      {/* 4. Payment Success Rate */}
      <Card className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground">
        <CardContent className="p-4">
          <div className="flex items-center justify-between text-muted-foreground">
            <span className="text-xs font-bold uppercase tracking-wider">Tingkat Sukses Pembayaran</span>
            <CheckCircle2 className="h-4 w-4" />
          </div>
          <div className="mt-2 text-2xl font-black tabular-nums">
            {payment_success_rate.toFixed(1)}%
          </div>
          <div className="mt-2 text-xs text-muted-foreground">
            30 hari terakhir
          </div>
        </CardContent>
      </Card>

      {/* 5. Pending Manual Payments (Deep Link) */}
      <Link href="/super-admin/payments/review" className="block group">
        <Card className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground h-full transition-transform hover:-translate-y-0.5">
          <CardContent className="p-4">
            <div className="flex items-center justify-between text-muted-foreground">
              <span className="text-xs font-bold uppercase tracking-wider">Menunggu Review Manual</span>
              <Clock className="h-4 w-4 group-hover:text-primary transition-colors" />
            </div>
            <div className="mt-2 text-2xl font-black tabular-nums flex items-baseline gap-2">
              <span className={pending_manual_payments > 0 ? 'text-amber-600 dark:text-amber-400' : ''}>
                {pending_manual_payments}
              </span>
              <span className="text-xs font-normal text-muted-foreground">transaksi</span>
            </div>
            <div className="mt-2 flex items-center text-xs font-bold text-primary gap-1">
              <span>Buka review</span>
              <ChevronRight className="h-3 w-3" />
            </div>
          </CardContent>
        </Card>
      </Link>

      {/* 6. AI Endpoints Healthy / Total */}
      <Card className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground">
        <CardContent className="p-4">
          <div className="flex items-center justify-between text-muted-foreground">
            <span className="text-xs font-bold uppercase tracking-wider">AI Endpoints Sehat</span>
            <Server className="h-4 w-4" />
          </div>
          <div className="mt-2 text-2xl font-black tabular-nums">
            {ai_endpoints_healthy} / {ai_endpoints_total}
          </div>
          <div className="mt-2 text-xs text-muted-foreground">
            {ai_endpoints_total > 0 && ai_endpoints_healthy === ai_endpoints_total
              ? 'Semua endpoint normal'
              : `${ai_endpoints_total - ai_endpoints_healthy} endpoint terganggu`}
          </div>
        </CardContent>
      </Card>

      {/* 7. Circuit Breakers Open (Deep Link) */}
      <Link href="/super-admin/ai-gateway" className="block group">
        <Card className={`border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground h-full transition-transform hover:-translate-y-0.5 ${
          circuit_breaker_open_count > 0 ? 'border-destructive' : ''
        }`}>
          <CardContent className="p-4">
            <div className="flex items-center justify-between text-muted-foreground">
              <span className="text-xs font-bold uppercase tracking-wider">Circuit Breaker Terbuka</span>
              <ShieldAlert className={`h-4 w-4 ${circuit_breaker_open_count > 0 ? 'text-destructive' : ''}`} />
            </div>
            <div className="mt-2 text-2xl font-black tabular-nums flex items-baseline gap-2">
              <span className={circuit_breaker_open_count > 0 ? 'text-destructive' : ''}>
                {circuit_breaker_open_count}
              </span>
              <span className="text-xs font-normal text-muted-foreground">terbuka</span>
            </div>
            <div className="mt-2 flex items-center text-xs font-bold text-primary gap-1">
              <span>Kelola AI Gateway</span>
              <ChevronRight className="h-3 w-3" />
            </div>
          </CardContent>
        </Card>
      </Link>
    </div>
  )
}
