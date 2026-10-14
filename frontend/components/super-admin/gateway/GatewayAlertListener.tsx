'use client'

import { useEffect } from 'react'
import { toast } from 'sonner'
import { subscribeDomainEvents } from '@/lib/realtime-bus'

/**
 * Surfaces AI-gateway operational alerts as toasts for Super Admins.
 *
 * Why this exists: gateway events (`ai_route_quarantined`, `ai_endpoint_down`)
 * are published on the ADMIN SSE channel and are deliberately NOT written to the
 * per-user notification inbox (they are operational, not user-facing). Without a
 * listener like this the event reaches the browser but has no visible surface —
 * the only effect would be a silent query refetch.
 *
 * Mounted in the super-admin layout so an alert is visible on any admin page, not
 * just the gateway control plane.
 */
export function GatewayAlertListener() {
  useEffect(() => {
    const seen = new Set<string>()

    return subscribeDomainEvents((event) => {
      // De-duplicate: an SSE reconnect can replay recent events.
      const key = event.id || `${event.event}-${event.timestamp}`
      if (seen.has(key)) return
      seen.add(key)

      const payload = (event.payload ?? {}) as {
        use_case?: string
        model?: string
        endpoint?: string
        endpoint_id?: string
        reason?: string
        recovered?: boolean
      }

      if (event.event === 'ai_route_quarantined') {
        if (payload.recovered) {
          toast.success('Rute AI pulih', {
            description: `Model ${payload.model ?? '?'} kembali normal dan siap melayani.`,
          })
          return
        }
        const where = [payload.use_case, payload.model].filter(Boolean).join(' / ')
        toast.warning('Model AI dikarantina', {
          description: `${where || 'Sebuah rute'} gagal berturut-turut; trafik dialihkan otomatis.${payload.reason ? ` (${payload.reason})` : ''}`,
        })
        return
      }

      if (event.event === 'ai_endpoint_down') {
        toast.error('Endpoint AI terganggu', {
          description: `Endpoint ${payload.endpoint_id ?? ''} tidak sehat; trafik dialihkan otomatis.`.trim(),
        })
      }
    })
  }, [])

  return null
}
