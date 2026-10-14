'use client'

import * as React from 'react'
import { createPortal } from 'react-dom'
import { Clock, Ban } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Money } from '@/components/shared/Money'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { useCancelInvoice, type Invoice } from '@/hooks/use-billing'
import { formatTanggal } from '@/lib/format'
import { formatCountdown, isCancellable, remainingMs, urgencyFor } from '@/lib/invoice-expiry'

const STATUS_LABEL: Record<Invoice['status'], string> = {
  draft: 'Draf',
  issued: 'Terbit',
  paid: 'Lunas',
  past_due: 'Jatuh tempo',
  voided: 'Dibatalkan',
  expired: 'Kedaluwarsa',
}

/** Explains why a closed invoice is closed (so "Dibatalkan" is unambiguous). */
function closedNote(inv: Invoice): string | null {
  if (inv.status !== 'voided' && inv.status !== 'expired') return null
  switch (inv.closed_reason) {
    case 'cancelled_by_user':
      return 'Dibatalkan oleh Anda'
    case 'expired':
      return 'Kedaluwarsa otomatis'
    case 'superseded_by_unique_open_invoice_constraint':
      return 'Digantikan tagihan terbaru'
    default:
      return null
  }
}

/**
 * One invoice row: number, dates, total, status, a live countdown for an open
 * invoice, and a cancel action. The countdown ticks once a second and the row
 * greys out when the window lapses, so a user always knows where they stand.
 */
export function InvoiceRow({ invoice }: { invoice: Invoice }) {
  const cancel = useCancelInvoice()
  const [confirmOpen, setConfirmOpen] = React.useState(false)
  const [, forceTick] = React.useState(0)

  const cancellable = isCancellable(invoice.status)

  // Tick once a second while the invoice is still open, so the countdown is
  // live. The interval is cleared as soon as the invoice is no longer open.
  React.useEffect(() => {
    if (!cancellable || !invoice.due_at) return
    const id = setInterval(() => forceTick((n) => n + 1), 1000)
    return () => clearInterval(id)
  }, [cancellable, invoice.due_at])

  const ms = remainingMs(invoice.due_at)
  const urgency = urgencyFor(ms)
  const countdown = cancellable ? formatCountdown(ms) : ''

  const urgencyClass =
    urgency === 'critical'
      ? 'text-red-600'
      : urgency === 'warning'
        ? 'text-amber-600'
        : 'text-muted-foreground'

  const handleCancel = async () => {
    await cancel.mutateAsync(invoice.id)
    setConfirmOpen(false)
  }

  const note = closedNote(invoice)

  return (
    <>
      <tr className="border-b border-foreground/20 last:border-b-0">
        <td className="p-3 font-mono font-bold">{invoice.invoice_number}</td>
        <td className="p-3 text-muted-foreground">{formatTanggal(invoice.issued_at)}</td>
        <td className="p-3 text-muted-foreground">
          <div>{invoice.due_at ? formatTanggal(invoice.due_at) : '—'}</div>
          {countdown && (
            <span className={`flex items-center gap-1 text-xs font-bold ${urgencyClass}`}>
              <Clock className="h-3 w-3" aria-hidden="true" />
              {countdown} lagi
            </span>
          )}
        </td>
        <td className="p-3 text-right">
          <Money value={invoice.total_idr} />
          {invoice.tax_idr > 0 && (
            <span className="block text-[10px] font-bold text-muted-foreground">
              incl. pajak <Money value={invoice.tax_idr} />
            </span>
          )}
        </td>
        <td className="p-3">
          <Badge
            variant={
              invoice.status === 'paid'
                ? 'default'
                : invoice.status === 'voided' || invoice.status === 'expired'
                  ? 'destructive'
                  : 'secondary'
            }
          >
            {STATUS_LABEL[invoice.status]}
          </Badge>
          {note && <p className="mt-1 text-[10px] font-bold text-muted-foreground">{note}</p>}
        </td>
        <td className="p-3 text-right">
          {cancellable ? (
            <Button
              variant="outline"
              size="sm"
              className="cursor-pointer text-red-600"
              onClick={() => setConfirmOpen(true)}
              disabled={cancel.isPending}
            >
              <Ban className="mr-1.5 h-3.5 w-3.5" /> Batalkan
            </Button>
          ) : (
            <span className="text-xs text-muted-foreground">—</span>
          )}
        </td>
      </tr>

      {/* The dialog is portalled to <body>: it renders a <div>, which is not a
          valid child of <tbody>. A portal keeps the markup valid while leaving
          the component self-contained. */}
      {typeof document !== 'undefined' &&
        createPortal(
          <ConfirmDialog
            open={confirmOpen}
            onOpenChange={setConfirmOpen}
            onConfirm={handleCancel}
            title="Batalkan Tagihan?"
            message={`Tagihan ${invoice.invoice_number} akan dibatalkan dan tidak dapat dibayar lagi. Anda dapat membuat tagihan baru setelahnya.`}
            confirmText="Ya, Batalkan"
            variant="danger"
            loading={cancel.isPending}
          />,
          document.body
        )}
    </>
  )
}
