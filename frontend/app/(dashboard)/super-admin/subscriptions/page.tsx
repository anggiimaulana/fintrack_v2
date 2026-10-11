'use client'

import { Suspense } from 'react'

import { SubscriptionsView } from '@/components/super-admin/subscriptions/SubscriptionsView'

export default function SuperAdminSubscriptionsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-foreground">
          Kelola Langganan
        </h1>
        <p className="text-sm text-muted-foreground mt-1">
          Pantau status langganan pengguna, perpanjang masa aktif, turunkan paket, atau batalkan langganan.
        </p>
      </div>

      {/* SubscriptionsView reads useSearchParams, so it needs a Suspense
          boundary for the route to stay statically prerenderable. */}
      <Suspense fallback={<div className="h-40 animate-pulse bg-muted border-2 border-foreground" />}>
        <SubscriptionsView />
      </Suspense>
    </div>
  )
}
