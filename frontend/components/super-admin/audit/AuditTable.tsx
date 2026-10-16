'use client'

import * as React from 'react'
import { format, formatDistanceToNow } from 'date-fns'
import { id as localeId } from 'date-fns/locale'
import { Copy, Check, Eye } from 'lucide-react'
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
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { DataTableColumnHeader } from '@/components/data-table/data-table-column-header'
import { DataTablePagination } from '@/components/data-table/data-table-pagination'
import { DataTableSkeleton } from '@/components/data-table/data-table-skeleton'
import { EmptyState } from '@/components/shared/EmptyState'
import type { AuditLogItem } from '@/hooks/useSuperAdminAudit'
import type { SortOrder } from '@/lib/table-params'

interface AuditTableProps {
  data: AuditLogItem[]
  isLoading: boolean
  total: number
  page: number
  perPage: number
  sort?: string
  order?: SortOrder
  onPageChange: (page: number) => void
  onPerPageChange: (perPage: number) => void
  onSort: (columnId: string, order: SortOrder) => void
  onSelectRow: (log: AuditLogItem) => void
}

export function AuditTable({
  data,
  isLoading,
  total,
  page,
  perPage,
  sort,
  order,
  onPageChange,
  onPerPageChange,
  onSort,
  onSelectRow,
}: AuditTableProps) {
  const [copiedId, setCopiedId] = React.useState<string | null>(null)

  const handleCopy = (e: React.MouseEvent, text: string, id: string) => {
    e.stopPropagation()
    navigator.clipboard.writeText(text)
    setCopiedId(id)
    setTimeout(() => setCopiedId(null), 2000)
  }

  const totalPages = Math.ceil(total / perPage)

  const getActionBadgeVariant = (action: string): 'default' | 'secondary' | 'destructive' | 'outline' => {
    const act = action.toLowerCase()
    if (act.includes('delete') || act.includes('cancel') || act.includes('reject')) return 'destructive'
    if (act.includes('create') || act.includes('approve') || act.includes('extend')) return 'default'
    if (act.includes('update') || act.includes('adjust') || act.includes('reset')) return 'secondary'
    return 'outline'
  }

  return (
    <div className="space-y-4">
      <div className="rounded-lg border border-border bg-card overflow-hidden">
        <TooltipProvider delayDuration={200}>
          <Table>
            <TableHeader className="bg-muted/50">
              <TableRow className="border-border hover:bg-transparent">
                {/* Nomor / No */}
                <TableHead className="w-12 text-center font-bold">No</TableHead>
                {/* Waktu / created_at */}
                <TableHead className="w-[180px]">
                  <DataTableColumnHeader
                    title="Waktu"
                    columnId="created_at"
                    sortable
                    currentSort={sort}
                    currentOrder={order}
                    onSort={onSort}
                  />
                </TableHead>

                {/* Aktor / actor */}
                <TableHead className="min-w-[180px]">
                  <DataTableColumnHeader
                    title="Aktor"
                    columnId="actor"
                    sortable
                    currentSort={sort}
                    currentOrder={order}
                    onSort={onSort}
                  />
                </TableHead>

                {/* Aksi / action */}
                <TableHead className="w-[120px]">
                  <DataTableColumnHeader
                    title="Aksi"
                    columnId="action"
                    sortable
                    currentSort={sort}
                    currentOrder={order}
                    onSort={onSort}
                  />
                </TableHead>

                {/* Entitas / entity */}
                <TableHead className="min-w-[160px]">
                  <DataTableColumnHeader
                    title="Entitas"
                    columnId="entity"
                    sortable
                    currentSort={sort}
                    currentOrder={order}
                    onSort={onSort}
                  />
                </TableHead>

                {/* IP Address */}
                <TableHead className="w-[120px] font-black uppercase tracking-wide text-xs">
                  IP
                </TableHead>

                {/* Request ID */}
                <TableHead className="w-[130px] font-black uppercase tracking-wide text-xs">
                  Request ID
                </TableHead>

                {/* Detail action */}
                <TableHead className="w-[80px] text-right font-black uppercase tracking-wide text-xs">
                  Aksi
                </TableHead>
              </TableRow>
            </TableHeader>

            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={7} className="p-0">
                    <DataTableSkeleton columns={7} rows={perPage > 10 ? 10 : perPage} />
                  </TableCell>
                </TableRow>
              ) : data.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="p-8">
                    <EmptyState
                      title="tidak ada aktivitas"
                      description="Belum ada catatan log aktivitas yang cocok dengan kriteria filter saat ini."
                    />
                  </TableCell>
                </TableRow>
              ) : (
                data.map((row) => {
                  const createdAtDate = new Date(row.created_at)
                  const isValidDate = !isNaN(createdAtDate.getTime())
                  const relativeTime = isValidDate
                    ? formatDistanceToNow(createdAtDate, { addSuffix: true, locale: localeId })
                    : row.created_at
                  const absoluteTime = isValidDate
                    ? format(createdAtDate, 'dd MMM yyyy, HH:mm:ss', { locale: localeId })
                    : row.created_at

                  const reqIdShort = row.request_id
                    ? row.request_id.length > 8
                      ? `${row.request_id.slice(0, 8)}…`
                      : row.request_id
                    : '-'

                  return (
                    <TableRow
                      key={row.id}
                      onClick={() => onSelectRow(row)}
                      className="cursor-pointer border-border hover:bg-muted/40 transition-colors"
                    >
                      {/* Created at (relative + absolute on hover tooltip) */}
                      <TableCell className="text-center font-mono text-xs text-muted-foreground w-12">
                        {(page - 1) * perPage + data.indexOf(row) + 1}
                      </TableCell>
                      <TableCell className="text-xs">
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <span className="font-medium text-foreground underline decoration-dotted decoration-muted-foreground/50 underline-offset-4 cursor-help">
                              {relativeTime}
                            </span>
                          </TooltipTrigger>
                          <TooltipContent side="top">
                            <span className="font-mono text-xs">{absoluteTime}</span>
                          </TooltipContent>
                        </Tooltip>
                      </TableCell>

                      {/* Actor (name + email) */}
                      <TableCell className="text-xs">
                        <div className="flex flex-col">
                          <span className="font-semibold text-foreground leading-tight">
                            {row.actor_name || 'Sistem / Anonim'}
                          </span>
                          <span className="text-[11px] text-muted-foreground truncate max-w-[200px]">
                            {row.actor_email || '-'}
                          </span>
                        </div>
                      </TableCell>

                      {/* Action Badge */}
                      <TableCell className="text-xs">
                        <Badge
                          variant={getActionBadgeVariant(row.action)}
                          className="font-mono text-[10px] uppercase font-semibold px-1.5 py-0.5"
                        >
                          {row.action}
                        </Badge>
                      </TableCell>

                      {/* Entity + Entity ID */}
                      <TableCell className="text-xs">
                        <div className="flex flex-col">
                          <span className="font-mono font-medium text-foreground">
                            {row.entity}
                          </span>
                          {row.entity_id && (
                            <span className="font-mono text-[10px] text-muted-foreground truncate max-w-[160px]">
                              {row.entity_id}
                            </span>
                          )}
                        </div>
                      </TableCell>

                      {/* IP */}
                      <TableCell className="text-xs font-mono text-muted-foreground">
                        {row.ip || '-'}
                      </TableCell>

                      {/* Request ID (truncated/copyable) */}
                      <TableCell className="text-xs">
                        {row.request_id ? (
                          <div className="flex items-center gap-1">
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="font-mono text-[11px] text-muted-foreground">
                                  {reqIdShort}
                                </span>
                              </TooltipTrigger>
                              <TooltipContent side="top">
                                <span className="font-mono text-xs">{row.request_id}</span>
                              </TooltipContent>
                            </Tooltip>
                            <Button
                              type="button"
                              variant="ghost"
                              size="icon"
                              className="h-5 w-5 text-muted-foreground hover:text-foreground"
                              onClick={(e) => handleCopy(e, row.request_id!, row.id)}
                              aria-label="Salin Request ID"
                            >
                              {copiedId === row.id ? (
                                <Check className="h-3 w-3 text-emerald-600 dark:text-emerald-400" />
                              ) : (
                                <Copy className="h-3 w-3" />
                              )}
                            </Button>
                          </div>
                        ) : (
                          <span className="text-muted-foreground">-</span>
                        )}
                      </TableCell>

                      {/* View row detail button */}
                      <TableCell className="text-right">
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          className="h-7 w-7 text-muted-foreground hover:text-foreground"
                          onClick={(e) => {
                            e.stopPropagation()
                            onSelectRow(row)
                          }}
                          aria-label="Lihat detail audit"
                        >
                          <Eye className="h-4 w-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  )
                })
              )}
            </TableBody>
          </Table>
        </TooltipProvider>
      </div>

      {/* Pagination */}
      <DataTablePagination
        page={page}
        perPage={perPage}
        total={total}
        totalPages={totalPages}
        onPageChange={onPageChange}
        onPerPageChange={onPerPageChange}
      />
    </div>
  )
}
