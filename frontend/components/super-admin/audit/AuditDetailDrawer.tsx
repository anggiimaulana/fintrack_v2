'use client'

import * as React from 'react'
import { format, formatDistanceToNow } from 'date-fns'
import { id as localeId } from 'date-fns/locale'
import { Copy, Check, Info } from 'lucide-react'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { JsonDiffViewer } from './JsonDiffViewer'
import type { AuditLogItem } from '@/hooks/useSuperAdminAudit'

interface AuditDetailDrawerProps {
  log: AuditLogItem | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function AuditDetailDrawer({ log, open, onOpenChange }: AuditDetailDrawerProps) {
  const [copiedKey, setCopiedKey] = React.useState<string | null>(null)

  const handleCopy = (text: string, key: string) => {
    navigator.clipboard.writeText(text)
    setCopiedKey(key)
    setTimeout(() => setCopiedKey(null), 2000)
  }

  if (!log) return null

  const createdAtDate = new Date(log.created_at)
  const formattedAbsolute = !isNaN(createdAtDate.getTime())
    ? format(createdAtDate, 'dd MMMM yyyy, HH:mm:ss', { locale: localeId })
    : log.created_at
  const formattedRelative = !isNaN(createdAtDate.getTime())
    ? formatDistanceToNow(createdAtDate, { addSuffix: true, locale: localeId })
    : '-'

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="w-full sm:max-w-2xl overflow-y-auto bg-card border-border p-6"
      >
        <SheetHeader className="space-y-1 text-left">
          <div className="flex items-center gap-2">
            <Badge variant="outline" className="font-mono text-xs uppercase">
              {log.action}
            </Badge>
            <Badge variant="secondary" className="font-mono text-xs">
              {log.entity}
            </Badge>
          </div>
          <SheetTitle className="text-lg font-bold text-foreground">
            Detail Log Audit
          </SheetTitle>
          <SheetDescription className="text-xs text-muted-foreground">
            Log ID: <span className="font-mono text-foreground">{log.id}</span>
          </SheetDescription>
        </SheetHeader>

        {/* Append-only Compliance Notice */}
        <div className="mt-4 flex items-start gap-2.5 rounded-md border border-sky-500/20 bg-sky-500/10 p-3 text-xs text-sky-800 dark:text-sky-300">
          <Info className="h-4 w-4 shrink-0 mt-0.5" />
          <div className="leading-relaxed">
            <strong className="font-semibold">Append-Only Trail:</strong> Log audit ini bersifat permanen dan tidak dapat diubah atau dihapus untuk kepatuhan tata kelola (UU PDP). Nilai sensitif telah disamarkan (masked).
          </div>
        </div>

        <Separator className="my-4" />

        {/* Metadata info grid */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs">
          <div>
            <span className="font-semibold text-muted-foreground block mb-0.5">Waktu Pencatatan</span>
            <div className="text-foreground font-medium">{formattedAbsolute}</div>
            <div className="text-[11px] text-muted-foreground">({formattedRelative})</div>
          </div>

          <div>
            <span className="font-semibold text-muted-foreground block mb-0.5">Aktor</span>
            <div className="text-foreground font-medium">{log.actor_name || 'Sistem / Anonim'}</div>
            <div className="text-[11px] text-muted-foreground">{log.actor_email || '-'}</div>
            {log.actor_id && (
              <div className="font-mono text-[10px] text-muted-foreground/75 truncate">
                ID: {log.actor_id}
              </div>
            )}
          </div>

          <div>
            <span className="font-semibold text-muted-foreground block mb-0.5">Entitas Target</span>
            <div className="text-foreground font-medium">{log.entity}</div>
            <div className="text-muted-foreground font-mono text-[11px]">
              ID: {log.entity_id || '-'}
            </div>
          </div>

          <div>
            <span className="font-semibold text-muted-foreground block mb-0.5">Alamat IP</span>
            <div className="font-mono text-foreground">{log.ip || '-'}</div>
          </div>

          <div className="sm:col-span-2">
            <span className="font-semibold text-muted-foreground block mb-0.5">Request ID</span>
            {log.request_id ? (
              <div className="flex items-center gap-2">
                <code className="rounded bg-muted px-2 py-0.5 font-mono text-xs text-foreground">
                  {log.request_id}
                </code>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6 text-muted-foreground hover:text-foreground"
                  onClick={() => handleCopy(log.request_id!, 'req_id')}
                  aria-label="Salin Request ID"
                >
                  {copiedKey === 'req_id' ? (
                    <Check className="h-3.5 w-3.5 text-emerald-600 dark:text-emerald-400" />
                  ) : (
                    <Copy className="h-3.5 w-3.5" />
                  )}
                </Button>
              </div>
            ) : (
              <span className="text-muted-foreground">-</span>
            )}
          </div>

          {log.user_agent && (
            <div className="sm:col-span-2">
              <span className="font-semibold text-muted-foreground block mb-0.5">User Agent</span>
              <p className="rounded bg-muted/50 p-2 font-mono text-[11px] text-muted-foreground break-all leading-normal">
                {log.user_agent}
              </p>
            </div>
          )}
        </div>

        <Separator className="my-5" />

        {/* Diff view */}
        <div className="space-y-2">
          <h3 className="text-sm font-semibold text-foreground">
            Perubahan Data (State Diff)
          </h3>
          <JsonDiffViewer before={log.before} after={log.after} />
        </div>
      </SheetContent>
    </Sheet>
  )
}
