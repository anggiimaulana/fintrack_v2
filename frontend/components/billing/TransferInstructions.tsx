'use client'

import * as React from 'react'
import { Copy, Check, Building2 } from 'lucide-react'
import { toast } from 'sonner'
import { usePaymentInstructions } from '@/hooks/use-billing'

/** A copyable label/value row. */
function InstructionRow({ label, value }: { label: string; value: string }) {
  const [copied, setCopied] = React.useState(false)
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      toast.error('Gagal menyalin ke papan klip')
    }
  }
  return (
    <div className="flex items-center justify-between gap-3 border-b-2 border-foreground/20 py-2 last:border-b-0">
      <span className="text-xs font-bold uppercase tracking-wide text-muted-foreground">{label}</span>
      <span className="flex items-center gap-2">
        <span className="font-mono text-sm font-black text-foreground">{value}</span>
        <button
          type="button"
          onClick={copy}
          aria-label={`Salin ${label}`}
          className="cursor-pointer text-muted-foreground transition hover:text-primary"
        >
          {copied ? <Check className="h-3.5 w-3.5 text-emerald-600" /> : <Copy className="h-3.5 w-3.5" />}
        </button>
      </span>
    </div>
  )
}

/**
 * Bank-transfer instructions for a manual payment, fetched lazily by payment id.
 * Falls back to a "contact admin" hint when no bank account is configured, so
 * the UI never shows an empty "transfer to ''" instruction.
 */
export function TransferInstructions({ paymentId, amountIdr }: { paymentId: string; amountIdr: number }) {
  const { instructions, isLoading } = usePaymentInstructions(paymentId)
  if (isLoading) {
    return <p className="text-xs font-bold text-muted-foreground">Memuat instruksi…</p>
  }
  const bank = instructions?.bank_name
  const account = instructions?.account_number
  const holder = instructions?.account_holder
  const hasBank = Boolean(bank || account || holder)
  return (
    <div className="border-2 border-foreground bg-muted/40 p-3">
      <p className="mb-2 flex items-center gap-1.5 text-xs font-black uppercase tracking-wide text-foreground">
        <Building2 className="h-3.5 w-3.5" aria-hidden="true" />
        Instruksi Transfer
      </p>
      <InstructionRow label="Jumlah" value={`Rp ${amountIdr.toLocaleString('id-ID')}`} />
      {hasBank ? (
        <>
          {bank && <InstructionRow label="Bank" value={bank} />}
          {account && <InstructionRow label="No. Rekening" value={account} />}
          {holder && <InstructionRow label="Atas Nama" value={holder} />}
        </>
      ) : (
        <p className="pt-1 text-xs font-bold text-muted-foreground">
          Hubungi admin untuk detail rekening tujuan, lalu unggah bukti transfer Anda.
        </p>
      )}
    </div>
  )
}
