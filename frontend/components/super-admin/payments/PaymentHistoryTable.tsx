'use client'

import * as React from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import { Eye, Undo2 } from 'lucide-react'
import { DataTable } from '@/components/data-table/data-table'
import { DataTableToolbar } from '@/components/data-table/data-table-toolbar'
import { DataTablePagination } from '@/components/data-table/data-table-pagination'
import { DataTableColumnHeader } from '@/components/data-table/data-table-column-header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Money } from '@/components/shared/Money'
import { formatTanggal } from '@/lib/format'
import { PaymentStatusBadge } from './PaymentStatusBadge'
import { PaymentDetailDialog } from './PaymentDetailDialog'
import { PaymentRefundDialog } from './PaymentActionDialogs'
import {
  useSuperAdminPayments,
  useSuperAdminPaymentMutations,
  type PaymentRow,
} from '@/hooks/useSuperAdminPayments'
import { useDataTableUrlState } from '@/hooks/use-data-table-url-state'

export function PaymentHistoryTable() {
  const urlState = useDataTableUrlState({
    sort: 'created_at',
    order: 'desc',
    perPage: 20,
  })

  const {
    payments,
    meta,
    isLoading,
    isError,
    error,
  } = useSuperAdminPayments({
    page: urlState.debounced.page,
    per_page: urlState.debounced.perPage,
    sort: urlState.debounced.sort,
    order: urlState.debounced.order,
    q: urlState.debounced.q,
    status: urlState.debounced.filters.status,
    gateway: urlState.debounced.filters.gateway,
    start_date: urlState.debounced.filters.start_date,
    end_date: urlState.debounced.filters.end_date,
  })

  const { refundPayment, isRefunding } = useSuperAdminPaymentMutations()

  const [selectedDetail, setSelectedDetail] = React.useState<PaymentRow | null>(null)
  const [selectedRefund, setSelectedRefund] = React.useState<PaymentRow | null>(null)

  const columns: ColumnDef<PaymentRow, unknown>[] = React.useMemo(
    () => [
      {
        accessorKey: 'invoice_number',
        header: () => (
          <DataTableColumnHeader
            title="Nomor Invoice"
            columnId="invoice_number"
            sortable={false}
          />
        ),
        cell: ({ row }) => {
          const item = row.original
          return (
            <div className="font-mono text-xs font-bold">
              {item.invoice_number || item.invoice_id.slice(0, 8)}
            </div>
          )
        },
      },
      {
        accessorKey: 'user_email',
        header: () => (
          <DataTableColumnHeader
            title="User / Email"
            columnId="user"
            sortable
            currentSort={urlState.sort}
            currentOrder={urlState.order}
            onSort={(c, o) => urlState.setSort(c, o)}
          />
        ),
        cell: ({ row }) => (
          <div className="max-w-[180px] truncate text-xs" title={row.original.user_email}>
            {row.original.user_email}
          </div>
        ),
      },
      {
        accessorKey: 'amount_idr',
        header: () => (
          <DataTableColumnHeader
            title="Nominal"
            columnId="amount"
            sortable
            currentSort={urlState.sort}
            currentOrder={urlState.order}
            onSort={(c, o) => urlState.setSort(c, o)}
          />
        ),
        cell: ({ row }) => (
          <div className="text-xs font-bold">
            <Money value={row.original.amount_idr} />
          </div>
        ),
      },
      {
        accessorKey: 'gateway',
        header: () => <span className="text-xs font-black uppercase">Gateway / Metode</span>,
        cell: ({ row }) => (
          <div className="text-xs">
            <span className="font-semibold">{row.original.gateway}</span>
            {row.original.payment_method && (
              <span className="text-muted-foreground ml-1">({row.original.payment_method})</span>
            )}
          </div>
        ),
      },
      {
        accessorKey: 'status',
        header: () => (
          <DataTableColumnHeader
            title="Status"
            columnId="status"
            sortable
            currentSort={urlState.sort}
            currentOrder={urlState.order}
            onSort={(c, o) => urlState.setSort(c, o)}
          />
        ),
        cell: ({ row }) => <PaymentStatusBadge status={row.original.status} />,
      },
      {
        accessorKey: 'reviewed_by',
        header: () => <span className="text-xs font-black uppercase">Reviewer</span>,
        cell: ({ row }) => {
          const { reviewed_by, reviewed_at } = row.original
          if (!reviewed_by && !reviewed_at) return <span className="text-xs text-muted-foreground">-</span>
          return (
            <div className="text-xs">
              <p className="font-mono truncate max-w-[120px]">{reviewed_by || 'Sistem'}</p>
              {reviewed_at && (
                <p className="text-[10px] text-muted-foreground">{formatTanggal(reviewed_at)}</p>
              )}
            </div>
          )
        },
      },
      {
        accessorKey: 'refund_amount_idr',
        header: () => <span className="text-xs font-black uppercase">Refund</span>,
        cell: ({ row }) => {
          const { refund_amount_idr, refund_reason } = row.original
          if (!refund_amount_idr || refund_amount_idr <= 0) {
            return <span className="text-xs text-muted-foreground">-</span>
          }
          return (
            <div className="text-xs text-purple-700 dark:text-purple-300">
              <Money value={refund_amount_idr} />
              {refund_reason && (
                <p className="text-[10px] text-muted-foreground truncate max-w-[130px]" title={refund_reason}>
                  {refund_reason}
                </p>
              )}
            </div>
          )
        },
      },
      {
        accessorKey: 'created_at',
        header: () => (
          <DataTableColumnHeader
            title="Tanggal Dibuat"
            columnId="created_at"
            sortable
            currentSort={urlState.sort}
            currentOrder={urlState.order}
            onSort={(c, o) => urlState.setSort(c, o)}
          />
        ),
        cell: ({ row }) => (
          <span className="text-xs text-muted-foreground">
            {formatTanggal(row.original.created_at)}
          </span>
        ),
      },
      {
        id: 'actions',
        header: () => <span className="text-xs font-black uppercase text-right block">Aksi</span>,
        cell: ({ row }) => {
          const p = row.original
          const canRefund =
            (p.status === 'settled' || p.status === 'refunded') &&
            p.amount_idr > (p.refund_amount_idr || 0)

          return (
            <div className="flex items-center justify-end gap-1">
              <Button
                type="button"
                variant="outline"
                size="icon"
                className="h-7 w-7"
                onClick={() => setSelectedDetail(p)}
                title="Lihat detail pembayaran"
                aria-label="Lihat detail pembayaran"
              >
                <Eye className="h-3.5 w-3.5" />
              </Button>
              {canRefund && (
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  className="h-7 w-7 text-purple-700 hover:text-purple-800"
                  onClick={() => setSelectedRefund(p)}
                  title="Refund pembayaran ini"
                  aria-label="Refund pembayaran ini"
                >
                  <Undo2 className="h-3.5 w-3.5" />
                </Button>
              )}
            </div>
          )
        },
      },
    ],
    [urlState]
  )

  const activeStatus = urlState.filters.status || 'all'
  const activeGateway = urlState.filters.gateway || 'all'
  const startDate = urlState.filters.start_date || ''
  const endDate = urlState.filters.end_date || ''

  return (
    <div className="space-y-4">
      <DataTableToolbar
        search={urlState.q}
        onSearch={(v) => urlState.setSearch(v)}
        placeholder="Cari email / nomor invoice..."
        filters={urlState.filters}
        onRemoveFilter={(k) => urlState.setFilter(k, '')}
      >
        <div className="flex flex-wrap items-center gap-2">
          {/* Status filter */}
          <div className="w-[140px]">
            <Select
              value={activeStatus}
              onValueChange={(val) => urlState.setFilter('status', val === 'all' ? '' : val)}
            >
              <SelectTrigger className="h-9 text-xs" aria-label="Filter status">
                <SelectValue placeholder="Semua Status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Semua Status</SelectItem>
                <SelectItem value="pending">Pending</SelectItem>
                <SelectItem value="pending_review">Pending Review</SelectItem>
                <SelectItem value="settled">Settled</SelectItem>
                <SelectItem value="failed">Failed</SelectItem>
                <SelectItem value="expired">Expired</SelectItem>
                <SelectItem value="refunded">Refunded</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Gateway filter */}
          <div className="w-[130px]">
            <Select
              value={activeGateway}
              onValueChange={(val) => urlState.setFilter('gateway', val === 'all' ? '' : val)}
            >
              <SelectTrigger className="h-9 text-xs" aria-label="Filter gateway">
                <SelectValue placeholder="Semua Gateway" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Semua Gateway</SelectItem>
                <SelectItem value="manual">Manual</SelectItem>
                <SelectItem value="xendit">Xendit</SelectItem>
                <SelectItem value="midtrans">Midtrans</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Start date */}
          <div className="flex items-center gap-1">
            <Label htmlFor="filter-start-date" className="text-xs text-muted-foreground sr-only">
              Tanggal Mulai
            </Label>
            <Input
              id="filter-start-date"
              type="date"
              className="h-9 w-[130px] text-xs"
              value={startDate}
              onChange={(e) => urlState.setFilter('start_date', e.target.value)}
              aria-label="Tanggal mulai"
            />
          </div>

          {/* End date */}
          <div className="flex items-center gap-1">
            <Label htmlFor="filter-end-date" className="text-xs text-muted-foreground sr-only">
              Tanggal Akhir
            </Label>
            <Input
              id="filter-end-date"
              type="date"
              className="h-9 w-[130px] text-xs"
              value={endDate}
              onChange={(e) => urlState.setFilter('end_date', e.target.value)}
              aria-label="Tanggal akhir"
            />
          </div>
        </div>
      </DataTableToolbar>

      <DataTable
        columns={columns}
        data={payments}
        isLoading={isLoading}
        error={isError ? (error as Error)?.message : null}
        emptyMessage="Tidak ada transaksi pembayaran yang cocok dengan filter."
        sort={urlState.sort}
        order={urlState.order}
        onSort={(col, ord) => urlState.setSort(col, ord)}
        onRowClick={(row) => setSelectedDetail(row)}
      />

      <DataTablePagination
        page={urlState.page}
        perPage={urlState.perPage}
        total={meta?.total ?? payments.length}
        totalPages={Math.ceil((meta?.total ?? payments.length) / urlState.perPage)}
        onPageChange={(p) => urlState.setPage(p)}
        onPerPageChange={(pp) => urlState.setPerPage(pp)}
      />

      <PaymentDetailDialog
        payment={selectedDetail}
        open={Boolean(selectedDetail)}
        onOpenChange={(open) => !open && setSelectedDetail(null)}
      />

      <PaymentRefundDialog
        payment={selectedRefund}
        open={Boolean(selectedRefund)}
        onOpenChange={(open) => !open && setSelectedRefund(null)}
        onConfirm={async (payload) => {
          await refundPayment(payload)
        }}
        isLoading={isRefunding}
      />
    </div>
  )
}
