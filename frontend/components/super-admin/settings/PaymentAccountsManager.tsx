'use client'

import * as React from 'react'
import { toast } from 'sonner'
import { Plus, Trash2, Pencil, X, Building2, Wallet, QrCode } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import {
  useAdminPaymentAccounts,
  useCreatePaymentAccount,
  useDeletePaymentAccount,
  useUpdatePaymentAccount,
  type PaymentAccountPayload,
} from '@/hooks/usePaymentAccounts'
import type { PaymentAccount, PaymentAccountKind } from '@/hooks/use-billing'
import { billingErrorMessage } from '@/hooks/use-billing'

const KIND_OPTIONS: { value: PaymentAccountKind; label: string; icon: typeof Building2 }[] = [
  { value: 'bank', label: 'Transfer Bank', icon: Building2 },
  { value: 'ewallet', label: 'E-Wallet', icon: Wallet },
  { value: 'qris', label: 'QRIS', icon: QrCode },
]

const EMPTY_FORM: PaymentAccountPayload = {
  kind: 'bank',
  name: '',
  account_number: '',
  account_holder: '',
  qris_url: '',
  sort_order: 0,
  is_active: true,
}

/** Create/edit form for one destination. */
function AccountForm({
  initial,
  onSubmit,
  onCancel,
  submitting,
}: {
  initial: PaymentAccountPayload
  onSubmit: (payload: PaymentAccountPayload) => Promise<void>
  onCancel: () => void
  submitting: boolean
}) {
  const [form, setForm] = React.useState<PaymentAccountPayload>(initial)
  const patch = (next: Partial<PaymentAccountPayload>) => setForm((f) => ({ ...f, ...next }))
  const isQRIS = form.kind === 'qris'

  return (
    <div className="space-y-3 border-2 rounded-lg border-foreground bg-muted/40 p-4">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div className="space-y-1.5">
          <Label htmlFor="pa_kind">Jenis</Label>
          <select
            id="pa_kind"
            value={form.kind}
            onChange={(e) => patch({ kind: e.target.value as PaymentAccountKind })}
            className="h-9 w-full border-2 rounded-md border-foreground bg-background px-2 text-sm font-bold"
          >
            {KIND_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>{o.label}</option>
            ))}
          </select>
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="pa_name">Nama</Label>
          <Input
            id="pa_name"
            value={form.name}
            placeholder={isQRIS ? 'QRIS' : 'BCA / DANA / SeaBank'}
            onChange={(e) => patch({ name: e.target.value })}
          />
        </div>
      </div>

      {isQRIS ? (
        <div className="space-y-1.5">
          <Label htmlFor="pa_qris">Tautan Gambar QRIS</Label>
          <Input
            id="pa_qris"
            value={form.qris_url}
            placeholder="https://…/qris.png"
            onChange={(e) => patch({ qris_url: e.target.value })}
          />
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div className="space-y-1.5">
            <Label htmlFor="pa_number">Nomor</Label>
            <Input
              id="pa_number"
              value={form.account_number}
              placeholder="1234567890"
              onChange={(e) => patch({ account_number: e.target.value })}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="pa_holder">Atas Nama</Label>
            <Input
              id="pa_holder"
              value={form.account_holder}
              placeholder="PT FinTrack"
              onChange={(e) => patch({ account_holder: e.target.value })}
            />
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div className="space-y-1.5">
          <Label htmlFor="pa_order">Urutan Tampil</Label>
          <Input
            id="pa_order"
            type="number"
            value={form.sort_order}
            onChange={(e) => patch({ sort_order: Number(e.target.value) })}
          />
        </div>
        <div className="flex items-end gap-3 pb-2">
          <input
            id="pa_active"
            type="checkbox"
            checked={form.is_active ?? true}
            onChange={(e) => patch({ is_active: e.target.checked })}
            className="size-4"
          />
          <Label htmlFor="pa_active" className="text-sm font-bold">Aktif</Label>
        </div>
      </div>

      <div className="flex justify-end gap-2">
        <Button variant="outline" size="sm" className="cursor-pointer" onClick={onCancel} disabled={submitting}>
          <X className="mr-1.5 h-3.5 w-3.5" /> Batal
        </Button>
        <Button size="sm" className="cursor-pointer" onClick={() => onSubmit(form)} disabled={submitting}>
          {submitting ? 'Menyimpan…' : 'Simpan Akun'}
        </Button>
      </div>
    </div>
  )
}

/** Row summary for one configured destination. */
function AccountRow({
  account,
  onEdit,
  onDelete,
  deleting,
}: {
  account: PaymentAccount
  onEdit: () => void
  onDelete: () => void
  deleting: boolean
}) {
  const Icon = KIND_OPTIONS.find((o) => o.value === account.kind)?.icon ?? Building2
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 border-2 rounded-md border-foreground bg-card p-3">
      <div className="flex min-w-0 items-center gap-3">
        <Icon className="h-4 w-4 shrink-0 text-primary" aria-hidden="true" />
        <div className="min-w-0">
          <p className="truncate text-sm font-black">
            {account.name}
            {!account.is_active && (
              <span className="ml-2 border-2 rounded-md border-foreground px-1.5 py-0.5 text-[10px] font-black uppercase text-muted-foreground">
                Nonaktif
              </span>
            )}
          </p>
          <p className="truncate font-mono text-xs font-bold text-muted-foreground">
            {account.kind === 'qris'
              ? account.qris_url || '—'
              : [account.account_number, account.account_holder].filter(Boolean).join(' · ') || '—'}
          </p>
        </div>
      </div>
      <div className="flex items-center gap-2">
        <Button variant="outline" size="sm" className="cursor-pointer" onClick={onEdit}>
          <Pencil className="h-3.5 w-3.5" />
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="cursor-pointer text-red-600"
          onClick={onDelete}
          disabled={deleting}
          aria-label={`Hapus akun ${account.name}`}
        >
          <Trash2 className="h-3.5 w-3.5" />
        </Button>
      </div>
    </div>
  )
}

/**
 * Manage the manual payment destinations users can pay to: bank accounts,
 * e-wallets, and QRIS. These appear at checkout and on a pending payment's
 * "Bayar Sekarang" modal.
 */
export function PaymentAccountsManager() {
  const { data: accounts = [], isLoading, isError } = useAdminPaymentAccounts()
  const create = useCreatePaymentAccount()
  const update = useUpdatePaymentAccount()
  const remove = useDeletePaymentAccount()

  const [creating, setCreating] = React.useState(false)
  const [editingId, setEditingId] = React.useState<string | null>(null)

  const handleCreate = async (payload: PaymentAccountPayload) => {
    try {
      await create.mutateAsync(payload)
      toast.success('Akun pembayaran ditambahkan')
      setCreating(false)
    } catch (e) {
      toast.error(billingErrorMessage(e, 'Gagal menambah akun'))
    }
  }

  const handleUpdate = async (id: string, payload: PaymentAccountPayload) => {
    try {
      await update.mutateAsync({ id, payload })
      toast.success('Akun pembayaran diperbarui')
      setEditingId(null)
    } catch (e) {
      toast.error(billingErrorMessage(e, 'Gagal memperbarui akun'))
    }
  }

  const handleDelete = async (account: PaymentAccount) => {
    if (!account.id) return
    try {
      await remove.mutateAsync(account.id)
      toast.success('Akun pembayaran dihapus')
    } catch (e) {
      toast.error(billingErrorMessage(e, 'Gagal menghapus akun'))
    }
  }

  const editing = accounts.find((a) => a.id === editingId) ?? null

  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between gap-4">
        <div>
          <CardTitle className="text-base">Tujuan Pembayaran</CardTitle>
          <CardDescription>
            Beberapa akun bank, e-wallet, dan QRIS yang ditampilkan ke pengguna saat membayar.
          </CardDescription>
        </div>
        {!creating && !editingId && (
          <Button size="sm" className="cursor-pointer" onClick={() => setCreating(true)}>
            <Plus className="mr-1.5 h-4 w-4" /> Tambah
          </Button>
        )}
      </CardHeader>
      <CardContent className="space-y-3">
        {isLoading ? (
          <div className="space-y-2">
            {[1, 2].map((i) => <Skeleton key={i} className="h-14 w-full" />)}
          </div>
        ) : isError ? (
          <p className="text-sm font-bold text-red-600">Gagal memuat akun pembayaran.</p>
        ) : (
          <>
            {creating && (
              <AccountForm
                initial={EMPTY_FORM}
                onSubmit={handleCreate}
                onCancel={() => setCreating(false)}
                submitting={create.isPending}
              />
            )}
            {editing && editing.id && (
              <AccountForm
                initial={{
                  kind: editing.kind,
                  name: editing.name,
                  account_number: editing.account_number ?? '',
                  account_holder: editing.account_holder ?? '',
                  qris_url: editing.qris_url ?? '',
                  sort_order: editing.sort_order ?? 0,
                  is_active: editing.is_active ?? true,
                }}
                onSubmit={(payload) => handleUpdate(editing.id as string, payload)}
                onCancel={() => setEditingId(null)}
                submitting={update.isPending}
              />
            )}

            {accounts.length === 0 && !creating ? (
              <p className="py-6 text-center text-sm font-bold text-muted-foreground">
                Belum ada tujuan pembayaran. Tambahkan minimal satu agar pengguna dapat membayar.
              </p>
            ) : (
              accounts.map((a) => (
                <AccountRow
                  key={a.id ?? `${a.kind}-${a.name}`}
                  account={a}
                  onEdit={() => { setCreating(false); setEditingId(a.id ?? null) }}
                  onDelete={() => handleDelete(a)}
                  deleting={remove.isPending}
                />
              ))
            )}
          </>
        )}
      </CardContent>
    </Card>
  )
}
