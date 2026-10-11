'use client'

import * as React from 'react'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'

interface JsonDiffViewerProps {
  before?: Record<string, unknown> | null
  after?: Record<string, unknown> | null
}

export function JsonDiffViewer({ before, after }: JsonDiffViewerProps) {
  const b = before && typeof before === 'object' ? before : null
  const a = after && typeof after === 'object' ? after : null

  const allKeys = React.useMemo(() => {
    const keys = new Set<string>()
    if (b) Object.keys(b).forEach((k) => keys.add(k))
    if (a) Object.keys(a).forEach((k) => keys.add(k))
    return Array.from(keys).sort()
  }, [b, a])

  if (!b && !a) {
    return (
      <div className="rounded-md border border-border bg-muted/40 p-4 text-center text-xs text-muted-foreground">
        Tidak ada data perubahan (before &amp; after kosong).
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Overview badges */}
      <div className="space-y-2">
        <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          Ringkasan Field Berubah
        </h4>
        <div className="flex flex-wrap gap-1.5">
          {allKeys.length === 0 ? (
            <span className="text-xs text-muted-foreground">Tidak ada field</span>
          ) : (
            allKeys.map((key) => {
              const inBefore = b && key in b
              const inAfter = a && key in a
              const valBefore = inBefore ? JSON.stringify(b[key]) : undefined
              const valAfter = inAfter ? JSON.stringify(a[key]) : undefined

              const label = key
              let statusText = 'tidak berubah'
              let colorClass = 'border-border bg-muted text-foreground'

              if (!inBefore && inAfter) {
                statusText = 'ditambahkan'
                colorClass = 'border-emerald-600/30 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
              } else if (inBefore && !inAfter) {
                statusText = 'dihapus'
                colorClass = 'border-rose-600/30 bg-rose-500/10 text-rose-600 dark:text-rose-400'
              } else if (valBefore !== valAfter) {
                statusText = 'diubah'
                colorClass = 'border-amber-600/30 bg-amber-500/10 text-amber-600 dark:text-amber-400'
              }

              return (
                <span
                  key={key}
                  className={`inline-flex items-center gap-1 rounded border px-2 py-0.5 text-xs font-medium font-mono ${colorClass}`}
                >
                  <span>{label}</span>
                  <span className="text-[10px] opacity-75 font-sans">({statusText})</span>
                </span>
              )
            })
          )}
        </div>
      </div>

      {/* Side-by-side or stacked JSON view */}
      <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
        {/* Before */}
        <div className="flex flex-col rounded-md border border-border bg-card">
          <div className="flex items-center justify-between border-b border-border bg-muted/40 px-3 py-2">
            <span className="text-xs font-semibold text-muted-foreground">Sebelum (Before)</span>
            {!b && <Badge variant="outline" className="text-[10px]">Kosong</Badge>}
          </div>
          <ScrollArea className="h-72 w-full">
            <pre className="overflow-auto p-3 text-xs font-mono leading-relaxed text-foreground whitespace-pre">
              {b ? JSON.stringify(b, null, 2) : '// Tidak ada data awal'}
            </pre>
          </ScrollArea>
        </div>

        {/* After */}
        <div className="flex flex-col rounded-md border border-border bg-card">
          <div className="flex items-center justify-between border-b border-border bg-muted/40 px-3 py-2">
            <span className="text-xs font-semibold text-muted-foreground">Sesudah (After)</span>
            {!a && <Badge variant="outline" className="text-[10px]">Kosong</Badge>}
          </div>
          <ScrollArea className="h-72 w-full">
            <pre className="overflow-auto p-3 text-xs font-mono leading-relaxed text-foreground whitespace-pre">
              {a ? JSON.stringify(a, null, 2) : '// Tidak ada data akhir'}
            </pre>
          </ScrollArea>
        </div>
      </div>
    </div>
  )
}
