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
import { Badge } from '@/components/ui/badge'
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
import { Money } from '@/components/shared/Money'
import { formatTanggal } from '@/lib/format'
import { Copy, Check, ArrowUpDown } from 'lucide-react'
import { toast } from 'sonner'
import { useTokenLedger } from '@/hooks/useSuperAdminTokens'

interface TokenLedgerTableProps {
  page: number
  perPage: number
  entryType: string
  modality: string
  userId: string
  startDate: string
  endDate: string
  sort: string
  order: 'asc' | 'desc'
  onPageChange: (page: number) => void
  onPerPageChange: (perPage: number) => void
  onFilterChange: (key: string, value: string) => void
  onSortChange: (sort: string, order: 'asc' | 'desc') => void
  onResetFilters: () => void
}

export function TokenLedgerTable({
  page,
  perPage,
  entryType,
  modality,
  userId,
  startDate,
  endDate,
  sort,
  order,
  onPageChange,
  onPerPageChange,
  onFilterChange,
  onSortChange,
  onResetFilters,
}: TokenLedgerTableProps) {
  const [copiedKey, setCopiedKey] = useState<string | null>(null)
  const [userInput, setUserInput] = useState(userId)

  const { data, isLoading, error } = useTokenLedger({
    page,
    per_page: perPage,
    entry_type: entryType || undefined,
    modality: modality || undefined,
    user_id: userId || undefined,
    start_date: startDate || undefined,
    end_date: endDate || undefined,
    sort: sort || undefined,
    order,
  })

  const rows = data?.data ?? []
  const meta = data?.meta ?? { page: 1, per_page: perPage, total: 0, total_pages: 0 }

  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text)
    setCopiedKey(text)
    toast.success('Idempotency key disalin ke clipboard')
    setTimeout(() => {
      setCopiedKey((curr) => (curr === text ? null : curr))
    }, 2000)
  }

  const handleSortToggle = (col: string) => {
    if (sort === col) {
      onSortChange(col, order === 'asc' ? 'desc' : 'asc')
    } else {
      onSortChange(col, 'desc')
    }
  }

  const getEntryTypeBadge = (type: string) => {
    switch (type) {
      case 'grant':
        return <Badge variant="secondary" className="font-mono text-[11px]">grant</Badge>
      case 'refund':
        return <Badge variant="outline" className="font-mono text-[11px] border-emerald-500 text-emerald-600 dark:text-emerald-400">refund</Badge>
      case 'commit':
        return <Badge variant="outline" className="font-mono text-[11px]">commit</Badge>
      case 'adjust':
        return <Badge variant="outline" className="font-mono text-[11px] border-amber-500 text-amber-600 dark:text-amber-400">adjust</Badge>
      case 'reset':
        return <Badge variant="destructive" className="font-mono text-[11px]">reset</Badge>
      default:
        return <Badge variant="outline" className="font-mono text-[11px]">{type}</Badge>
    }
  }

  const getModalityBadge = (m: string) => {
    return (
      <Badge variant="outline" className="bg-muted font-mono text-[11px] text-muted-foreground">
        {m}
      </Badge>
    )
  }

  const hasActiveFilters = Boolean(entryType || modality || userId || startDate || endDate)

  return (
    <div className="space-y-4">
      {/* Facet Filters Toolbar */}
      <div className="grid gap-3 rounded-md border bg-card p-3 text-xs md:grid-cols-6 sm:grid-cols-3">
        {/* entry_type */}
        <div className="space-y-1">
          <label htmlFor="ledger-filter-entry-type" className="text-muted-foreground font-semibold">Tipe Entri</label>
          <Select
            value={entryType || 'all'}
            onValueChange={(val) => onFilterChange('entry_type', val === 'all' ? '' : val)}
          >
            <SelectTrigger className="h-8 text-xs" id="ledger-filter-entry-type" aria-label="Filter tipe entri">
              <SelectValue placeholder="Semua Tipe" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Semua Tipe</SelectItem>
              <SelectItem value="commit">Commit</SelectItem>
              <SelectItem value="refund">Refund</SelectItem>
              <SelectItem value="adjust">Adjust</SelectItem>
              <SelectItem value="grant">Grant</SelectItem>
              <SelectItem value="reset">Reset</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* modality */}
        <div className="space-y-1">
          <label htmlFor="ledger-filter-modality" className="text-muted-foreground font-semibold">Modalitas</label>
          <Select
            value={modality || 'all'}
            onValueChange={(val) => onFilterChange('modality', val === 'all' ? '' : val)}
          >
            <SelectTrigger className="h-8 text-xs" id="ledger-filter-modality" aria-label="Filter modalitas">
              <SelectValue placeholder="Semua Modalitas" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Semua Modalitas</SelectItem>
              <SelectItem value="chat">Chat</SelectItem>
              <SelectItem value="vision">Vision</SelectItem>
              <SelectItem value="voice">Voice</SelectItem>
              <SelectItem value="embedding">Embedding</SelectItem>
              <SelectItem value="admin">Admin</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* user_id */}
        <div className="space-y-1">
          <label htmlFor="ledger-filter-user" className="text-muted-foreground font-semibold">User ID (UUID)</label>
          <div className="flex gap-1">
            <Input
              id="ledger-filter-user"
              value={userInput}
              onChange={(e) => setUserInput(e.target.value)}
              onBlur={() => onFilterChange('user_id', userInput.trim())}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  onFilterChange('user_id', userInput.trim())
                }
              }}
              placeholder="UUID user..."
              className="h-8 text-xs font-mono"
              aria-label="Filter berdasarkan User ID"
            />
          </div>
        </div>

        {/* start_date */}
        <div className="space-y-1">
          <label htmlFor="ledger-filter-from" className="text-muted-foreground font-semibold">Dari Tanggal</label>
          <Input
            id="ledger-filter-from"
            type="date"
            value={startDate}
            onChange={(e) => onFilterChange('start_date', e.target.value)}
            className="h-8 text-xs"
            aria-label="Dari tanggal"
          />
        </div>

        {/* end_date */}
        <div className="space-y-1">
          <label htmlFor="ledger-filter-to" className="text-muted-foreground font-semibold">Sampai Tanggal</label>
          <Input
            id="ledger-filter-to"
            type="date"
            value={endDate}
            onChange={(e) => onFilterChange('end_date', e.target.value)}
            className="h-8 text-xs"
            aria-label="Sampai tanggal"
          />
        </div>

        {/* Sort & Reset */}
        <div className="flex flex-col justify-end space-y-1">
          <div className="flex items-center gap-1">
            <Select
              value={sort || 'created_at'}
              onValueChange={(val) => onSortChange(val, order)}
            >
              <SelectTrigger className="h-8 text-xs flex-1" aria-label="Urutkan ledger">
                <SelectValue placeholder="Urutan" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="created_at">Waktu</SelectItem>
                <SelectItem value="amount">Jumlah</SelectItem>
                <SelectItem value="cost">Biaya</SelectItem>
                <SelectItem value="user">User</SelectItem>
                <SelectItem value="type">Tipe</SelectItem>
              </SelectContent>
            </Select>

            <Button
              variant="outline"
              size="sm"
              onClick={() => onSortChange(sort || 'created_at', order === 'asc' ? 'desc' : 'asc')}
              className="h-8 px-2"
              aria-label="Arah urutan"
            >
              <ArrowUpDown className="h-3.5 w-3.5" />
            </Button>

            {hasActiveFilters && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  setUserInput('')
                  onResetFilters()
                }}
                className="h-8 px-2 text-xs font-semibold text-destructive hover:bg-destructive/10"
              >
                Reset
              </Button>
            )}
          </div>
        </div>
      </div>

      {isLoading ? (
        <DataTableSkeleton columns={8} rows={6} />
      ) : error ? (
        <div className="border-2 rounded-lg border-destructive bg-destructive/10 p-4 text-sm font-semibold text-destructive">
          Gagal memuat ledger: {error instanceof Error ? error.message : 'Terjadi kesalahan sistem'}
        </div>
      ) : rows.length === 0 ? (
        <EmptyState
          title="Tidak ada entri ledger"
          description={
            hasActiveFilters
              ? 'Tidak ada transaksi token yang sesuai filter pencarian.'
              : 'Belum ada transaksi di ledger token.'
          }
        />
      ) : (
        <div className="rounded-md border bg-card">
          <Table>
            <TableHeader>
              <TableRow className="bg-muted/50 hover:bg-muted/50">
                <TableHead
                  className="cursor-pointer select-none"
                  onClick={() => handleSortToggle('created_at')}
                >
                  <div className="flex items-center gap-1">
                    Waktu
                    {sort === 'created_at' && (
                      <span className="text-xs font-mono">{order === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead
                  className="cursor-pointer select-none"
                  onClick={() => handleSortToggle('user')}
                >
                  <div className="flex items-center gap-1">
                    Pengguna
                    {sort === 'user' && (
                      <span className="text-xs font-mono">{order === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead
                  className="cursor-pointer select-none"
                  onClick={() => handleSortToggle('type')}
                >
                  <div className="flex items-center gap-1">
                    Tipe
                    {sort === 'type' && (
                      <span className="text-xs font-mono">{order === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead>Modalitas</TableHead>
                <TableHead
                  className="cursor-pointer select-none text-right"
                  onClick={() => handleSortToggle('amount')}
                >
                  <div className="flex items-center justify-end gap-1">
                    Jumlah Token
                    {sort === 'amount' && (
                      <span className="text-xs font-mono">{order === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead
                  className="cursor-pointer select-none text-right"
                  onClick={() => handleSortToggle('cost')}
                >
                  <div className="flex items-center justify-end gap-1">
                    Biaya (IDR)
                    {sort === 'cost' && (
                      <span className="text-xs font-mono">{order === 'asc' ? '↑' : '↓'}</span>
                    )}
                  </div>
                </TableHead>
                <TableHead>Ref / Catatan</TableHead>
                <TableHead className="w-36">Idempotency Key</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => {
                const isPositive = row.amount > 0
                const isNegative = row.amount < 0
                const formattedAmount = `${isPositive ? '+' : ''}${row.amount.toLocaleString('id-ID')}`

                return (
                  <TableRow key={row.id}>
                    <TableCell className="text-xs text-muted-foreground whitespace-nowrap">
                      {formatTanggal(row.created_at)}
                    </TableCell>
                    <TableCell>
                      <div className="font-mono text-xs">{row.user_email}</div>
                    </TableCell>
                    <TableCell>{getEntryTypeBadge(row.entry_type)}</TableCell>
                    <TableCell>{getModalityBadge(row.modality)}</TableCell>
                    <TableCell className="text-right font-mono text-sm tabular-nums">
                      <span
                        className={
                          isPositive
                            ? 'font-bold text-emerald-600 dark:text-emerald-400'
                            : isNegative
                            ? 'font-bold text-destructive'
                            : 'text-muted-foreground'
                        }
                      >
                        {formattedAmount}
                      </span>
                    </TableCell>
                    <TableCell className="text-right">
                      <Money value={row.cost_idr} />
                    </TableCell>
                    <TableCell className="max-w-[180px] truncate text-xs text-muted-foreground">
                      {row.notes || row.reference_id || '-'}
                    </TableCell>
                    <TableCell>
                      {row.idempotency_key ? (
                        <div className="flex items-center gap-1 font-mono text-xs text-muted-foreground">
                          <span title={row.idempotency_key}>
                            {row.idempotency_key.slice(0, 8)}…
                          </span>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-6 w-6 shrink-0"
                            onClick={() => handleCopy(row.idempotency_key)}
                            aria-label="Salin idempotency key"
                          >
                            {copiedKey === row.idempotency_key ? (
                              <Check className="h-3 w-3 text-emerald-600" />
                            ) : (
                              <Copy className="h-3 w-3" />
                            )}
                          </Button>
                        </div>
                      ) : (
                        <span className="text-xs text-muted-foreground">-</span>
                      )}
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
    </div>
  )
}
