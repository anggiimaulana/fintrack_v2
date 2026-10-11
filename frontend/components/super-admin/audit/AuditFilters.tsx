'use client'

import * as React from 'react'
import { Search, RotateCcw, AlertCircle } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

const ENTITY_OPTIONS = [
  { value: 'all', label: 'Semua Entitas' },
  { value: 'plan', label: 'Plan' },
  { value: 'subscription', label: 'Subscription' },
  { value: 'payment', label: 'Payment' },
  { value: 'token', label: 'Token' },
  { value: 'ai_route', label: 'AI Route' },
  { value: 'settings', label: 'Settings' },
]

const UUID_REGEX = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i

interface AuditFiltersProps {
  startDate: string
  endDate: string
  email: string
  userId: string
  entity: string
  action: string
  entityId: string
  userIdError?: string | null
  onFilterChange: (key: string, value: string) => void
  onReset: () => void
}

export function AuditFilters({
  startDate,
  endDate,
  email,
  userId,
  entity,
  action,
  entityId,
  userIdError,
  onFilterChange,
  onReset,
}: AuditFiltersProps) {
  const [localEmail, setLocalEmail] = React.useState(email)
  const [localUserId, setLocalUserId] = React.useState(userId)
  const [localAction, setLocalAction] = React.useState(action)
  const [localEntityId, setLocalEntityId] = React.useState(entityId)

  // Adopt external (URL) filter changes without a setState-in-effect: React's
  // "adjust state during render" pattern. Signature compares the incoming
  // values, so typing in a field is not clobbered by its own echo.
  const externalSignature = `${email}\u0000${userId}\u0000${action}\u0000${entityId}`
  const [adoptedSignature, setAdoptedSignature] = React.useState(externalSignature)
  if (adoptedSignature !== externalSignature) {
    setAdoptedSignature(externalSignature)
    setLocalEmail(email)
    setLocalUserId(userId)
    setLocalAction(action)
    setLocalEntityId(entityId)
  }

  // Debounced input updates (300ms)
  React.useEffect(() => {
    const timer = setTimeout(() => {
      if (localEmail !== email) onFilterChange('email', localEmail)
    }, 300)
    return () => clearTimeout(timer)
  }, [localEmail, email, onFilterChange])

  React.useEffect(() => {
    const timer = setTimeout(() => {
      if (localUserId !== userId) onFilterChange('user_id', localUserId)
    }, 300)
    return () => clearTimeout(timer)
  }, [localUserId, userId, onFilterChange])

  React.useEffect(() => {
    const timer = setTimeout(() => {
      if (localAction !== action) onFilterChange('action', localAction)
    }, 300)
    return () => clearTimeout(timer)
  }, [localAction, action, onFilterChange])

  React.useEffect(() => {
    const timer = setTimeout(() => {
      if (localEntityId !== entityId) onFilterChange('entity_id', localEntityId)
    }, 300)
    return () => clearTimeout(timer)
  }, [localEntityId, entityId, onFilterChange])

  // Local UUID validation check for immediate inline warning (before backend 400 validation returns)
  const isInvalidUuid = localUserId.trim() !== '' && !UUID_REGEX.test(localUserId.trim())

  return (
    <div className="rounded-lg border border-border bg-card p-4 space-y-4">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
        {/* Tanggal Mulai */}
        <div className="space-y-1.5">
          <Label htmlFor="start_date" className="text-xs font-semibold text-muted-foreground">
            Dari Tanggal
          </Label>
          <Input
            id="start_date"
            type="date"
            value={startDate}
            onChange={(e) => onFilterChange('start_date', e.target.value)}
            className="h-9 text-xs"
          />
        </div>

        {/* Tanggal Akhir */}
        <div className="space-y-1.5">
          <Label htmlFor="end_date" className="text-xs font-semibold text-muted-foreground">
            Sampai Tanggal
          </Label>
          <Input
            id="end_date"
            type="date"
            value={endDate}
            onChange={(e) => onFilterChange('end_date', e.target.value)}
            className="h-9 text-xs"
          />
        </div>

        {/* Filter Email Actor */}
        <div className="space-y-1.5">
          <Label htmlFor="filter_email" className="text-xs font-semibold text-muted-foreground">
            Email Aktor
          </Label>
          <div className="relative">
            <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              id="filter_email"
              type="text"
              placeholder="Cari email..."
              value={localEmail}
              onChange={(e) => setLocalEmail(e.target.value)}
              className="h-9 pl-8 text-xs"
            />
          </div>
        </div>

        {/* Filter Entitas */}
        <div className="space-y-1.5">
          <Label htmlFor="filter_entity" className="text-xs font-semibold text-muted-foreground">
            Tipe Entitas
          </Label>
          <Select
            value={entity || 'all'}
            onValueChange={(val) => onFilterChange('entity', val === 'all' ? '' : val)}
          >
            <SelectTrigger id="filter_entity" className="h-9 text-xs">
              <SelectValue placeholder="Semua Entitas" />
            </SelectTrigger>
            <SelectContent>
              {ENTITY_OPTIONS.map((opt) => (
                <SelectItem key={opt.value} value={opt.value} className="text-xs">
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Filter User ID (UUID) */}
        <div className="space-y-1.5">
          <Label htmlFor="filter_user_id" className="text-xs font-semibold text-muted-foreground">
            User ID (UUID)
          </Label>
          <Input
            id="filter_user_id"
            type="text"
            placeholder="UUID aktor/user..."
            value={localUserId}
            onChange={(e) => setLocalUserId(e.target.value)}
            className={`h-9 font-mono text-xs ${
              isInvalidUuid || userIdError ? 'border-destructive focus-visible:ring-destructive' : ''
            }`}
          />
          {(isInvalidUuid || userIdError) && (
            <p className="flex items-center gap-1 text-[11px] text-destructive">
              <AlertCircle className="h-3 w-3 shrink-0" />
              <span>{userIdError || 'Format UUID tidak valid (harus standard 36-karakter)'}</span>
            </p>
          )}
        </div>

        {/* Filter Action */}
        <div className="space-y-1.5">
          <Label htmlFor="filter_action" className="text-xs font-semibold text-muted-foreground">
            Aksi (Action)
          </Label>
          <Input
            id="filter_action"
            type="text"
            placeholder="create, update, refund..."
            value={localAction}
            onChange={(e) => setLocalAction(e.target.value)}
            className="h-9 text-xs font-mono"
          />
        </div>

        {/* Filter Entity ID */}
        <div className="space-y-1.5">
          <Label htmlFor="filter_entity_id" className="text-xs font-semibold text-muted-foreground">
            ID Entitas
          </Label>
          <Input
            id="filter_entity_id"
            type="text"
            placeholder="ID target..."
            value={localEntityId}
            onChange={(e) => setLocalEntityId(e.target.value)}
            className="h-9 text-xs font-mono"
          />
        </div>

        {/* Action Button: Reset */}
        <div className="flex items-end">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={onReset}
            className="h-9 text-xs w-full gap-1.5"
          >
            <RotateCcw className="h-3.5 w-3.5" />
            Reset Filter
          </Button>
        </div>
      </div>
    </div>
  )
}
