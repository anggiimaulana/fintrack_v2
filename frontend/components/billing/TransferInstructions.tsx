'use client'

import * as React from 'react'
import { Copy, Check, Building2, Wallet, QrCode } from 'lucide-react'
import { toast } from 'sonner'
import { usePaymentInstructions, type PaymentAccount, type PaymentAccountKind } from '@/hooks/use-billing'

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

const KIND_ICON: Record<PaymentAccountKind, typeof Building2> = {
  bank: Building2,
  ewallet: Wallet,
  qris: QrCode,
}

const KIND_LABEL: Record<PaymentAccountKind, string> = {
  bank: 'Transfer Bank',
  ewallet: 'E-Wallet',
  qris: 'QRIS',
}

/** One payment destination card: account details (copyable) or a QRIS image. */
function AccountCard({ account }: { account: PaymentAccount }) {
  const Icon = KIND_ICON[account.kind] ?? Building2
  return (
    <div className="border-2 border-foreground bg-card p-3">
      <div className="mb-2 flex items-center gap-2">
        <Icon className="h-4 w-4 text-primary" aria-hidden="true" />
        <span className="text-sm font-black">{account.name}</span>
        <span className="ml-auto border-2 border-foreground px-1.5 py-0.5 text-[10px] font-black uppercase text-muted-foreground">
          {KIND_LABEL[account.kind] ?? account.kind}
        </span>
      </div>

      {account.kind === 'qris' ? (
        account.qris_url ? (
          <div className="space-y-2">
            {/* QRIS images are uploaded by the operator; a plain <img> avoids
                next/image's remote-domain config for an admin-supplied URL. */}
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src={account.qris_url}
              alt={`Kode QRIS ${account.name}`}
              className="mx-auto h-48 w-48 border-2 border-foreground bg-white object-contain p-1"
            />
            <p className="text-center text-[11px] font-bold text-muted-foreground">
              Pindai kode di atas dengan aplikasi pembayaran Anda.
            </p>
          </div>
        ) : (
          <p className="text-xs font-bold text-muted-foreground">Kode QRIS belum tersedia.</p>
        )
      ) : (
        <>
          {account.account_number && <InstructionRow label="Nomor" value={account.account_number} />}
          {account.account_holder && <InstructionRow label="Atas Nama" value={account.account_holder} />}
        </>
      )}
    </div>
  )
}

/**
 * Payment instructions for a manual payment: the amount to transfer, every
 * destination the operator configured (bank / e-wallet / QRIS), and — for a
 * legacy deployment with no accounts — the single gateway-configured account.
 *
 * Fetched lazily by payment id, so the same component works in the checkout
 * modal and on a payment row the user re-opens later.
 */
export function TransferInstructions({ paymentId, amountIdr }: { paymentId: string; amountIdr: number }) {
  const { instructions, isLoading } = usePaymentInstructions(paymentId)

  if (isLoading) {
    return <p className="text-xs font-bold text-muted-foreground">Memuat instruksi…</p>
  }

  const accounts = instructions?.accounts ?? []
  const legacyBank = instructions?.bank_name
  const legacyAccount = instructions?.account_number
  const legacyHolder = instructions?.account_holder
  const hasLegacy = Boolean(legacyBank || legacyAccount || legacyHolder)

  return (
    <div className="space-y-3">
      {/* Amount to pay — the one thing every method needs. */}
      <div className="border-2 border-foreground bg-muted/40 p-3">
        <p className="mb-2 text-xs font-black uppercase tracking-wide text-foreground">Jumlah yang Harus Dibayar</p>
        <InstructionRow label="Jumlah" value={`Rp ${amountIdr.toLocaleString('id-ID')}`} />
        {instructions?.order_id && <InstructionRow label="Kode Order" value={instructions.order_id} />}
      </div>

      {/* All configured destinations. */}
      {accounts.length > 0 ? (
        <div className="space-y-2">
          <p className="text-xs font-black uppercase tracking-wide text-foreground">
            Pilih Salah Satu Tujuan Pembayaran
          </p>
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
            {accounts.map((acc, i) => (
              <AccountCard key={acc.id ?? `${acc.kind}-${acc.name}-${i}`} account={acc} />
            ))}
          </div>
        </div>
      ) : hasLegacy ? (
        <div className="border-2 border-foreground bg-muted/40 p-3">
          <p className="mb-2 flex items-center gap-1.5 text-xs font-black uppercase tracking-wide text-foreground">
            <Building2 className="h-3.5 w-3.5" aria-hidden="true" /> Instruksi Transfer
          </p>
          {legacyBank && <InstructionRow label="Bank" value={legacyBank} />}
          {legacyAccount && <InstructionRow label="No. Rekening" value={legacyAccount} />}
          {legacyHolder && <InstructionRow label="Atas Nama" value={legacyHolder} />}
        </div>
      ) : (
        <p className="border-2 border-dashed border-foreground/40 p-3 text-xs font-bold text-muted-foreground">
          Belum ada tujuan pembayaran yang dikonfigurasi. Hubungi admin untuk detail rekening, lalu unggah bukti
          transfer Anda.
        </p>
      )}
    </div>
  )
}
