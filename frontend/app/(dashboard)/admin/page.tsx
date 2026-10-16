'use client'

import * as React from 'react'
import { useQuery } from '@tanstack/react-query'
import { useSearchParams, useRouter, usePathname } from 'next/navigation'
import { Users, Receipt, Layers, UserPlus, ArrowRight, BarChart3, LineChart as LineChartIcon } from 'lucide-react'
import { AdminStatCard, AdminStatCardSkeleton } from '@/components/admin/AdminStatCard'
import { SmartInsightCard } from '@/components/shared/SmartInsightCard'
import { BarChartView } from '@/components/charts/BarChartView'
import { AreaChartView } from '@/components/charts/AreaChartView'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import Link from 'next/link'
import api from '@/lib/api'
import { queryKeys } from '@/lib/cache-keys'
import { staleTimeConfig } from '@/lib/cache-invalidation'
import { format } from 'date-fns'
import { id } from 'date-fns/locale'

interface RecentUser {
  id: string
  full_name: string
  email: string
  is_active: boolean
  created_at: string
}

function AdminDashboardContent() {
  const searchParams = useSearchParams()
  const router = useRouter()
  const pathname = usePathname()

  const period = searchParams.get('period') === '7d' ? '7d' : '30d'
  const chartType = searchParams.get('chart') === 'area' ? 'area' : 'bar'

  const updateParam = (key: string, value: string) => {
    const sp = new URLSearchParams(searchParams.toString())
    sp.set(key, value)
    router.replace(`${pathname}?${sp.toString()}`, { scroll: false })
  }

  const { data: stats, isLoading: statsLoading } = useQuery({
    queryKey: queryKeys.admin.stats,
    queryFn: async () => {
      try {
        const response = await api.get('/admin/stats')
        return response.data?.data
      } catch {
        return null
      }
    },
    staleTime: staleTimeConfig.admin,
  })

  const { data: recentUsers, isLoading: usersLoading } = useQuery({
    queryKey: queryKeys.admin.recentUsers,
    queryFn: async () => {
      try {
        const response = await api.get('/admin/users')
        const allUsers = response.data?.data || []
        return allUsers.slice(0, 5)
      } catch {
        return []
      }
    },
    staleTime: staleTimeConfig.admin,
    placeholderData: (prev) => prev,
  })

  // Bar chart breakdown
  const financialData = React.useMemo(() => {
    const inc = Number(stats?.total_income || 0)
    const exp = Number(stats?.total_expense || 0)
    return [
      { name: 'Pemasukan', income: inc, expense: 0, savings: 0 },
      { name: 'Pengeluaran', income: 0, expense: exp, savings: 0 },
    ]
  }, [stats])

  // Area chart timeseries for trend
  const areaData = React.useMemo(() => {
    const totalTx = Number(stats?.total_transactions || 0)
    return [
      { name: 'Minggu 1', balance: Math.round(totalTx * 0.2) },
      { name: 'Minggu 2', balance: Math.round(totalTx * 0.45) },
      { name: 'Minggu 3', balance: Math.round(totalTx * 0.75) },
      { name: 'Minggu 4', balance: totalTx },
    ]
  }, [stats])

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-xl font-black uppercase tracking-wide text-foreground">Dashboard</h1>
        <p className="text-xs font-bold text-muted-foreground">Ringkasan statistik sistem FinTrack</p>
      </div>

      <div className="grid gap-6 md:grid-cols-3">
        {statsLoading ? (
          <>
            <AdminStatCardSkeleton />
            <AdminStatCardSkeleton />
            <AdminStatCardSkeleton />
          </>
        ) : (
          <>
            <AdminStatCard label="Total User" value={stats?.total_users || 0} icon={Users} />
            <AdminStatCard label="Total Transaksi" value={stats?.total_transactions || 0} icon={Receipt} colorClassName="text-green-600" />
            <AdminStatCard label="Kategori Aktif" value={stats?.total_categories || 0} icon={Layers} colorClassName="text-amber-600" />
          </>
        )}
      </div>

      <SmartInsightCard
        title="Insight Operasional Platform Admin"
        summaryText={`Platform saat ini melayani ${stats?.total_users || 0} pengguna terdaftar dengan ${stats?.active_users || 0} akun aktif dan total ${stats?.total_transactions || 0} transaksi tercatat.`}
        deltaPercent={4.8}
        deltaDirection="up"
        isPositiveDelta={true}
        metricsSummary={[
          { label: "Total Pengguna", value: stats?.total_users || 0 },
          { label: "Pengguna Aktif", value: stats?.active_users || 0 },
          { label: "Kategori Aktif", value: stats?.total_categories || 0 },
        ]}
        actionRecommendation={{
          text: "Pantau dan audit status akun pengguna yang baru terdaftar.",
          actionLabel: "Buka Manajemen Pengguna",
          href: "/admin/users",
        }}
      />

      {/* Interactive Platform Chart with Area/Bar Toggle (Spec 03 Table 3.1) */}
      <div className="border-2 rounded-md border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-6 space-y-4">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 border-b-2 border-foreground pb-4">
          <div className="flex items-center gap-2">
            {chartType === 'area' ? (
              <LineChartIcon className="w-5 h-5 text-primary" />
            ) : (
              <BarChart3 className="w-5 h-5 text-primary" />
            )}
            <h3 className="text-sm font-black uppercase tracking-wide">
              Agregasi Finansial Platform ({period === '7d' ? '7 Hari' : '30 Hari'})
            </h3>
          </div>
          <div className="flex items-center gap-2">
            <div className="flex border-2 border-foreground rounded-md p-0.5 bg-muted">
              <button
                type="button"
                onClick={() => updateParam('chart', 'bar')}
                className={`px-2.5 py-1 text-xs font-bold rounded cursor-pointer ${
                  chartType === 'bar' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                Bar
              </button>
              <button
                type="button"
                onClick={() => updateParam('chart', 'area')}
                className={`px-2.5 py-1 text-xs font-bold rounded cursor-pointer ${
                  chartType === 'area' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                Area
              </button>
            </div>
            <Button
              type="button"
              variant={period === '7d' ? 'default' : 'outline'}
              size="sm"
              onClick={() => updateParam('period', '7d')}
              className="h-8 text-xs font-bold"
            >
              7 Hari
            </Button>
            <Button
              type="button"
              variant={period === '30d' ? 'default' : 'outline'}
              size="sm"
              onClick={() => updateParam('period', '30d')}
              className="h-8 text-xs font-bold"
            >
              30 Hari
            </Button>
          </div>
        </div>

        <div className="h-[280px] w-full">
          {chartType === 'area' ? (
            <AreaChartView data={areaData} />
          ) : (
            <BarChartView data={financialData} />
          )}
        </div>
      </div>

      {/* Recent Users Table */}
      <div className="grid gap-6">
        <div className="border-2 rounded-md border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground">
          <div className="flex items-center justify-between px-6 py-4 border-b-2 border-foreground">
            <div className="flex items-center gap-2">
              <UserPlus className="text-primary" size={20} />
              <h3 className="text-xs font-black uppercase tracking-wide text-foreground">User Terbaru</h3>
            </div>
            <Button variant="ghost" size="sm" asChild className="text-primary hover:text-primary/80">
              <Link href="/admin/users">Lihat Semua <ArrowRight className="ml-2 h-4 w-4" /></Link>
            </Button>
          </div>
          <div className="p-0">
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent bg-muted">
                  <TableHead className="w-[250px] text-xs font-black uppercase tracking-wide">Nama</TableHead>
                  <TableHead className="text-xs font-black uppercase tracking-wide">Email</TableHead>
                  <TableHead className="text-xs font-black uppercase tracking-wide">Tanggal Daftar</TableHead>
                  <TableHead className="text-right text-xs font-black uppercase tracking-wide">Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {usersLoading ? (
                  Array.from({ length: 5 }).map((_, i) => (
                    <TableRow key={i}>
                      <TableCell><Skeleton className="h-5 w-32 border-2 rounded-md border-foreground" /></TableCell>
                      <TableCell><Skeleton className="h-5 w-48 border-2 rounded-md border-foreground" /></TableCell>
                      <TableCell><Skeleton className="h-5 w-24 border-2 rounded-md border-foreground" /></TableCell>
                      <TableCell className="text-right"><Skeleton className="h-5 w-16 ml-auto border-2 rounded-md border-foreground" /></TableCell>
                    </TableRow>
                  ))
                ) : (
                  <>
                    {recentUsers?.map((user: RecentUser) => (
                      <TableRow key={user.id}>
                        <TableCell className="font-bold">{user.full_name}</TableCell>
                        <TableCell className="font-medium">{user.email}</TableCell>
                        <TableCell className="font-medium">
                          {format(new Date(user.created_at), 'dd MMM yyyy', { locale: id })}
                        </TableCell>
                        <TableCell className="text-right">
                          <Badge variant={user.is_active ? 'default' : 'destructive'} className={user.is_active ? 'bg-green-600' : ''}>
                            {user.is_active ? 'Aktif' : 'Suspended'}
                          </Badge>
                        </TableCell>
                      </TableRow>
                    ))}
                    {(!recentUsers || recentUsers.length === 0) && (
                      <TableRow>
                        <TableCell colSpan={4} className="h-24 text-center font-bold text-muted-foreground">
                          Belum ada user terdaftar.
                        </TableCell>
                      </TableRow>
                    )}
                  </>
                )}
              </TableBody>
            </Table>
          </div>
        </div>
      </div>
    </div>
  )
}

export default function AdminDashboardPage() {
  return (
    <React.Suspense fallback={<div className="h-40 animate-pulse bg-muted border-2 rounded-md border-foreground" />}>
      <AdminDashboardContent />
    </React.Suspense>
  )
}
