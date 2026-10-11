'use client'

import * as React from 'react'
import { ExternalLink, ImageIcon } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'

interface PaymentProofDialogProps {
  proofUrl?: string | null
  invoiceNumber?: string | null
}

export function PaymentProofDialog({ proofUrl, invoiceNumber }: PaymentProofDialogProps) {
  const [open, setOpen] = React.useState(false)

  if (!proofUrl) {
    return (
      <span className="inline-flex items-center gap-1.5 text-xs text-muted-foreground italic">
        <ImageIcon className="h-3.5 w-3.5 opacity-60" />
        tanpa bukti
      </span>
    )
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <button
          type="button"
          className="cursor-pointer group relative inline-flex items-center gap-2 overflow-hidden rounded border border-border bg-card p-1 text-left transition hover:border-foreground focus:outline-none focus:ring-2 focus:ring-ring"
          aria-label="Lihat bukti transfer"
        >
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={proofUrl}
            alt={`Bukti pembayaran ${invoiceNumber || ''}`}
            className="h-12 w-12 rounded object-cover transition group-hover:scale-105"
            loading="lazy"
          />
          <span className="text-xs font-semibold text-primary underline underline-offset-2">
            Lihat Bukti
          </span>
        </button>
      </DialogTrigger>
      <DialogContent className="max-w-2xl sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center justify-between pr-6 text-base">
            <span>Bukti Pembayaran {invoiceNumber ? `(${invoiceNumber})` : ''}</span>
            <Button
              variant="outline"
              size="sm"
              asChild
              className="gap-1.5"
            >
              <a className="cursor-pointer" href={proofUrl} target="_blank" rel="noopener noreferrer">
                <ExternalLink className="h-3.5 w-3.5" />
                Buka Tab Baru
              </a>
            </Button>
          </DialogTitle>
        </DialogHeader>
        <div className="flex max-h-[70vh] items-center justify-center overflow-auto rounded border border-border bg-muted/40 p-2">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={proofUrl}
            alt={`Bukti pembayaran penuh ${invoiceNumber || ''}`}
            className="max-h-[65vh] w-auto max-w-full rounded object-contain"
          />
        </div>
      </DialogContent>
    </Dialog>
  )
}
