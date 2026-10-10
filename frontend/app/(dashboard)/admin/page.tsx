'use client'

import { useQuery } from '@tanstack/react-query'
import { Users, Receipt, Layers, UserPlus, ArrowRight } from 'lucide-react'
import { AdminStatCard, AdminStatCardSkeleton } from '@/components/admin/AdminStatCard'
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

export default function AdminDashboardPage() {
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
        const response = await api.get('/admin/users?limit=5')
        return response.data?.data || []
      } catch {
        return []
      }
    },
    staleTime: staleTimeConfig.admin,
    placeholderData: (prev) => prev,
  })

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

      <div className="grid gap-6">
        <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground">
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
                      <TableCell><Skeleton className="h-5 w-32 border-2 border-foreground" /></TableCell>
                      <TableCell><Skeleton className="h-5 w-48 border-2 border-foreground" /></TableCell>
                      <TableCell><Skeleton className="h-5 w-24 border-2 border-foreground" /></TableCell>
                      <TableCell className="text-right"><Skeleton className="h-5 w-16 ml-auto border-2 border-foreground" /></TableCell>
                    </TableRow>
                  ))
                ) : (
                  <>
                    {recentUsers?.map((user: any) => (
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
