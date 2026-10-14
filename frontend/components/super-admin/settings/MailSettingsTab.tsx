'use client'

import * as React from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Mail, Send, CheckCircle2, AlertCircle } from 'lucide-react'
import { useMailStatus, useSendTestMail } from '@/hooks/use-mail-admin'

interface MailSettingsTabProps {
  /** Raw settings rows (from useSuperAdminSettings) so we can prefill values. */
  settings: { key: string; value: unknown }[]
  isLoading: boolean
  onSave: (dirtySettings: Record<string, unknown>) => Promise<unknown>
  isSaving: boolean
}

/** Reads a saved setting as a string ("" when absent). */
function readValue(settings: MailSettingsTabProps['settings'], key: string): string {
  const row = settings.find((s) => s.key === key)
  if (!row || row.value === undefined || row.value === null) return ''
  return String(row.value)
}

/** The form shape, derived from the saved settings. */
interface MailForm {
  mail_enabled: boolean
  mail_host: string
  mail_port: string
  mail_username: string
  mail_password: string
  mail_from: string
  mail_from_name: string
  mail_use_tls: boolean
}

/** Seed the form from the saved rows (used as the pre-edit baseline). */
function formFromSettings(settings: MailSettingsTabProps['settings']): MailForm {
  return {
    mail_enabled: readValue(settings, 'mail_enabled') === 'true',
    mail_host: readValue(settings, 'mail_host'),
    mail_port: readValue(settings, 'mail_port') || '587',
    mail_username: readValue(settings, 'mail_username'),
    // Never prefill the password: the server only reports whether one is set.
    mail_password: '',
    mail_from: readValue(settings, 'mail_from'),
    mail_from_name: readValue(settings, 'mail_from_name') || 'FinTrack',
    mail_use_tls: readValue(settings, 'mail_use_tls') === 'true',
  }
}

/**
 * Transactional-email configuration.
 *
 * Values saved here take precedence over the process environment, so SMTP can be
 * changed without a redeploy. The password is write-only: an empty field leaves
 * the stored (encrypted) value untouched. A test button verifies delivery with
 * the effective config before you rely on it.
 */
export function MailSettingsTab({ settings, isLoading, onSave, isSaving }: MailSettingsTabProps) {
  const { data: status, isLoading: statusLoading, refetch } = useMailStatus()
  const sendTest = useSendTestMail()

  // Only user edits are stored as state; the saved rows provide the baseline.
  // This avoids writing state from an effect just to mirror props.
  const [edits, setEdits] = React.useState<Partial<MailForm>>({})
  const [testTo, setTestTo] = React.useState('')
  const form: MailForm = { ...formFromSettings(settings), ...edits }

  const patch = (next: Partial<MailForm>) => setEdits((prev) => ({ ...prev, ...next }))

  const handleSave = async () => {
    const payload: Record<string, unknown> = {
      mail_enabled: form.mail_enabled,
      mail_host: form.mail_host.trim(),
      mail_port: Number(form.mail_port) || 587,
      mail_username: form.mail_username.trim(),
      mail_from: form.mail_from.trim(),
      mail_from_name: form.mail_from_name.trim(),
      mail_use_tls: form.mail_use_tls,
    }
    // Only send the password when the operator typed a new one.
    if (form.mail_password.trim() !== '') payload.mail_password = form.mail_password
    await onSave(payload)
    setEdits({})
    refetch()
  }

  if (isLoading) {
    return (
      <div className="space-y-3">
        {[1, 2, 3].map((i) => (
          <Skeleton key={i} className="h-12" />
        ))}
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Effective status banner: is email actually wired up right now? */}
      <Card>
        <CardContent className="flex items-center gap-3 p-4">
          {statusLoading ? (
            <Skeleton className="h-6 w-40" />
          ) : status?.configured ? (
            <>
              <CheckCircle2 className="size-5 text-emerald-600" />
              <span className="text-sm font-bold">
                Email aktif — {status.host}:{status.port} (dari {status.from})
              </span>
            </>
          ) : (
            <>
              <AlertCircle className="size-5 text-amber-600" />
              <span className="text-sm font-bold">
                Email belum aktif. Notifikasi tetap tercatat di inbox aplikasi, tetapi email tidak dikirim.
              </span>
            </>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Mail className="size-4" /> Konfigurasi SMTP
          </CardTitle>
          <CardDescription>
            Nilai di sini menimpa default dari environment, jadi bisa diubah tanpa deploy ulang.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-3">
            <input
              id="mail_enabled"
              type="checkbox"
              checked={form.mail_enabled}
              onChange={(e) => patch({ mail_enabled: e.target.checked })}
              className="size-4"
            />
            <Label htmlFor="mail_enabled" className="text-sm font-bold">Aktifkan pengiriman email</Label>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <div className="space-y-1.5 sm:col-span-2">
              <Label htmlFor="mail_host">SMTP Host</Label>
              <Input id="mail_host" value={form.mail_host} placeholder="smtp.example.com"
                onChange={(e) => patch({ mail_host: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="mail_port">Port</Label>
              <Input id="mail_port" type="number" value={form.mail_port} placeholder="587"
                onChange={(e) => patch({ mail_port: e.target.value })} />
            </div>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="mail_username">Username</Label>
              <Input id="mail_username" value={form.mail_username}
                onChange={(e) => patch({ mail_username: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="mail_password">Password (biarkan kosong untuk tidak mengubah)</Label>
              <Input id="mail_password" type="password" value={form.mail_password}
                placeholder={status?.has_password ? '••••••••' : ''}
                onChange={(e) => patch({ mail_password: e.target.value })} />
            </div>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="mail_from">Alamat Pengirim (From)</Label>
              <Input id="mail_from" type="email" value={form.mail_from} placeholder="no-reply@fintrack.id"
                onChange={(e) => patch({ mail_from: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="mail_from_name">Nama Pengirim</Label>
              <Input id="mail_from_name" value={form.mail_from_name}
                onChange={(e) => patch({ mail_from_name: e.target.value })} />
            </div>
          </div>

          <div className="flex items-center gap-3">
            <input
              id="mail_use_tls"
              type="checkbox"
              checked={form.mail_use_tls}
              onChange={(e) => patch({ mail_use_tls: e.target.checked })}
              className="size-4"
            />
            <Label htmlFor="mail_use_tls" className="text-sm font-bold">
              Gunakan TLS langsung (port 465). Nonaktif = STARTTLS (port 587).
            </Label>
          </div>

          <div className="flex flex-wrap items-center gap-3 pt-2">
            <Button onClick={handleSave} disabled={isSaving}>
              {isSaving ? 'Menyimpan...' : 'Simpan Konfigurasi'}
            </Button>
            <Input
              value={testTo}
              onChange={(e) => setTestTo(e.target.value)}
              placeholder="Email tujuan uji (kosong = alamat From)"
              className="max-w-xs"
            />
            <Button variant="outline" onClick={() => sendTest.mutate(testTo || undefined)} disabled={sendTest.isPending}>
              <Send className="size-4 mr-2" />
              {sendTest.isPending ? 'Mengirim...' : 'Kirim Email Uji'}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
