'use client'

import { useState } from 'react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { DataTablePagination } from '@/components/data-table/data-table-pagination'
import { DataTableSkeleton } from '@/components/data-table/data-table-skeleton'
import { EmptyState } from '@/components/shared/EmptyState'
import { formatTanggal } from '@/lib/format'
import { Search, SlidersHorizontal, ArrowUpDown } from 'lucide-react'
import {
  useTokenBalances,
  type TokenBalanceRow,
} from '@/hooks/useSuperAdminTokens'
import { AdjustTokensDialog } from './AdjustTokensDialog'

interface TokenBalancesTableProps {
  page: number
  perPage: number
  search: string
  sort: string
  order: 'asc' | 'desc'
  onPageChange: (page: number) => void
  onPerPageChange: (perPage: number) => void
  onSearchChange: (search: string) => void
  onSortChange: (sort: string, order: 'asc' | 'desc') => void
}

export function TokenBalancesTable({
  page,
  perPage,
  search,
  sort,
  order,
  onPageChange,
  onPerPageChange,
  onSearchChange,
  onSortChange,
}: TokenBalancesTableProps) {
  const [selectedUser, setSelectedUser] = useState<TokenBalanceRow | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [searchInput, setSearchInput] = useState(search)

  const { data, isLoading, error } = useTokenBalances({
    page,
    per_page: perPage,
    q: search || undefined,
    sort: sort || undefined,
    order,
  })

  const rows = data?.data ?? []
  const meta = data?.meta ?? { page: 1, per_page: perPage, total: 0, total_pages: 0 }

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSearchChange(searchInput.trim())
  }

  const handleSortToggle = (column: string) => {
    if (sort === column) {
      onSortChange(column, order === 'asc' ? 'desc' : 'asc')
    } else {
      onSortChange(column, 'desc')
    }
  }

  const handleAdjustClick = (user: TokenBalanceRow) => {
    setSelectedUser(user)
    setDialogOpen(true)
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <form onSubmit={handleSearchSubmit} className="flex flex-1 max-w-sm items-center gap-2">
          <div className="relative w-full">
            <Search className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              placeholder="Cari email atau nama pengguna..."
              className="pl-8"
              aria-label="Cari pengguna"
            />
          </div>
          <Button type="submit" variant="outline" size="sm">
            Cari
          </Button>
        </form>

        <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground font-medium">Urutkan:</span>
          <Select
            value={sort || 'default'}
            onValueChange={(val) => {
              if (val === 'default') {
                onSortChange('', 'desc')
              } else {
                onSortChange(val, order)
              }
            }}
          >
            <SelectTrigger className="w-[160px] h-9 text-xs" aria-label="Urutkan berdasarkan">
              <SelectValue placeholder="Bawaan" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="default">Bawaan</SelectItem>
              <SelectItem value="user">Email Pengguna</SelectItem>
              <SelectItem value="quota">Kuota</SelectItem>
              <SelectItem value="used">Terpakai</SelectItem>
            </SelectContent>
          </Select>

          {sort && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => onSortChange(sort, order === 'asc' ? 'desc' : 'asc')}
              className="h-9 px-2.5 text-xs"
              aria-label="Ubah arah pengurutan"
            >
              <ArrowUpDown className="h-3.5 w-3.5 mr-1" />
              {order === 'asc' ? 'A-Z / Naik' : 'Z-A / Turun'}
            </Button>
          )}
        </div>
      </div>

      {isLoading ? (
        <DataTableSkeleton columns={7} rows={6} />
      ) : error ? (
        <div className="border-2 rounded-lg border-destructive bg-destructive/10 p-4 text-sm font-semibold text-destructive">
          Gagal memuat saldo token: {error instanceof Error ? error.message : 'Terjadi kesalahan sistem'}
        </div>
      ) : rows.length === 0 ? (
        <EmptyState
          title="Tidak ada data saldo token"
          description={search ? 'Tidak ada pengguna yang cocok dengan pencarian.' : 'Belum ada data periode token.'}
        />
      ) : (
        <div className="rounded-md border bg-card">
          <Table>
            <TableHeader>
              <TableRow className="bg-muted/50 hover:bg-muted/50">
                <TableHead className="cursor-pointer select-none" onClick={() => handleSortToggle('user')}>
                  <div className="flex items-center gap-1">
                    Pengguna
                    {sort === 'user' && (
                      <span className="text-xs font-mono">{order === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead>Periode Aktif</TableHead>
                <TableHead
                  className="cursor-pointer select-none text-right"
                  onClick={() => handleSortToggle('quota')}
                >
                  <div className="flex items-center justify-end gap-1">
                    Kuota
                    {sort === 'quota' && (
                      <span className="text-xs font-mono">{order === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead
                  className="cursor-pointer select-none text-right"
                  onClick={() => handleSortToggle('used')}
                >
                  <div className="flex items-center justify-end gap-1">
                    Terpakai
                    {sort === 'used' && (
                      <span className="text-xs font-mono">{order === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead className="text-right">Dipesan</TableHead>
                <TableHead className="text-right">Tersedia</TableHead>
                <TableHead className="w-48">Penggunaan</TableHead>
                <TableHead className="text-right">Aksi</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => {
                const quota = row.quota_tokens || 0
                const used = row.used_tokens || 0
                const percent = quota > 0 ? Math.min(Math.round((used / quota) * 100), 100) : 0
                const isCritical = percent >= 80

                return (
                  <TableRow key={row.user_id}>
                    <TableCell>
                      <div className="font-semibold text-foreground">{row.user_full_name || '-'}</div>
                      <div className="text-xs text-muted-foreground font-mono">{row.user_email}</div>
                    </TableCell>
                    <TableCell>
                      {row.period_start && row.period_end ? (
                        <div className="text-xs text-muted-foreground whitespace-nowrap">
                          {formatTanggal(row.period_start)} – {formatTanggal(row.period_end)}
                        </div>
                      ) : (
                        <span className="text-xs text-muted-foreground italic">Tidak ada periode aktif</span>
                      )}
                    </TableCell>
                    <TableCell className="text-right font-mono text-sm tabular-nums">
                      {quota.toLocaleString('id-ID')}
                    </TableCell>
                    <TableCell className="text-right font-mono text-sm tabular-nums">
                      {used.toLocaleString('id-ID')}
                    </TableCell>
                    <TableCell className="text-right font-mono text-sm tabular-nums text-muted-foreground">
                      {row.reserved_tokens.toLocaleString('id-ID')}
                    </TableCell>
                    <TableCell className="text-right font-mono text-sm tabular-nums font-semibold">
                      {row.available_tokens.toLocaleString('id-ID')}
                    </TableCell>
                    <TableCell>
                      <div className="space-y-1">
                        <div className="flex items-center justify-between text-[11px] font-mono">
                          <span className={isCritical ? 'font-bold text-destructive' : 'text-muted-foreground'}>
                            {percent}%
                          </span>
                          <span className="text-muted-foreground">
                            {used.toLocaleString('id-ID')} / {quota.toLocaleString('id-ID')}
                          </span>
                        </div>
                        <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
                          <div
                            className={`h-full transition-all ${
                              isCritical ? 'bg-destructive' : 'bg-primary'
                            }`}
                            style={{ width: `${percent}%` }}
                          />
                        </div>
                      </div>
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleAdjustClick(row)}
                        className="h-8 text-xs font-semibold"
                      >
                        <SlidersHorizontal className="h-3.5 w-3.5 mr-1" />
                        Sesuaikan
                      </Button>
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </div>
      )}

      <DataTablePagination
        page={meta.page}
        perPage={meta.per_page}
        total={meta.total}
        totalPages={meta.total_pages}
        onPageChange={onPageChange}
        onPerPageChange={onPerPageChange}
      />

      <AdjustTokensDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        user={selectedUser}
      />
    </div>
  )
}
