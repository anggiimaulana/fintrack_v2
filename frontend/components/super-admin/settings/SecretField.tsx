'use client'

import * as React from 'react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import type { ConfigSource } from '@/hooks/use-config-status'

interface SecretFieldProps {
  id: string
  label: string
  /** Current plaintext entry (empty until the operator types). */
  value: string
  onChange: (value: string) => void
  /** Masked view of the stored secret (e.g. "SB-M••••1234"), "" when unset. */
  masked?: string
  /** Whether a value is configured at all. */
  hasSecret?: boolean
  /** Where the active value came from. */
  source?: ConfigSource
  placeholder?: string
}

const SOURCE_LABEL: Record<ConfigSource, string> = {
  db: 'dari DB',
  env: 'dari ENV',
  none: 'belum diatur',
}

/**
 * A write-only secret input. The stored value is never sent to the browser; we
 * show a masked hint plus where it came from (DB override vs env default).
 * Leaving the field empty keeps the existing value unchanged.
 */
export function SecretField({ id, label, value, onChange, masked, hasSecret, source, placeholder }: SecretFieldProps) {
  return (
    <div className="space-y-1.5">
      <div className="flex items-center gap-2">
        <Label htmlFor={id}>{label}</Label>
        {source && (
          <Badge variant={source === 'none' ? 'secondary' : 'default'} className="text-[10px]">
            {SOURCE_LABEL[source]}
          </Badge>
        )}
      </div>
      <Input
        id={id}
        type="password"
        value={value}
        placeholder={hasSecret && masked ? `${masked} (biarkan kosong untuk tidak mengubah)` : placeholder}
        onChange={(e) => onChange(e.target.value)}
        autoComplete="new-password"
      />
    </div>
  )
}
