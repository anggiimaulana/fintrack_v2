"use client"

import * as React from "react"
import { Input } from "@/components/ui/input"
import { NumberInput } from "@/components/shared/NumberInput"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import type { PlanLimitCycle } from "@/hooks/usePlanCycles"

interface DegradedParamsFormProps {
  cycle: PlanLimitCycle
  onChange: (updated: PlanLimitCycle) => void
}

const FALLBACK_MODELS = [
  { value: "gpt-4o-mini", label: "GPT-4o Mini (Hemat)" },
  { value: "gemini-1.5-flash", label: "Gemini 1.5 Flash (Cepat)" },
  { value: "claude-3-haiku", label: "Claude 3 Haiku (Ringan)" },
]

export function DegradedParamsForm({ cycle, onChange }: DegradedParamsFormProps) {
  if (cycle.cycle_type !== "weekly_monthly") return null

  return (
    <div className="mt-3 p-3 bg-muted/50 rounded-md border text-xs space-y-3">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        <div>
          <Label className="text-xs font-medium">Perilaku Saat Mingguan Habis</Label>
          <Select
            value={cycle.degraded_mode || "hard_block"}
            onValueChange={(val: "hard_block" | "throttle" | "model_downgrade") => {
              onChange({
                ...cycle,
                degraded_mode: val,
                fallback_model: val === "model_downgrade" ? (cycle.fallback_model || "gpt-4o-mini") : null,
                throttle_rate_amount: val === "throttle" ? (cycle.throttle_rate_amount || 5) : null,
                throttle_rate_window_secs: val === "throttle" ? (cycle.throttle_rate_window_secs || 60) : null,
              })
            }}
          >
            <SelectTrigger className="h-8 text-xs mt-1 bg-background">
              <SelectValue placeholder="Pilih perilaku" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="hard_block">Blokir Total (Hard Block)</SelectItem>
              <SelectItem value="throttle">Batasi Kecepatan (Rate Limit)</SelectItem>
              <SelectItem value="model_downgrade">Turunkan Model AI (Downgrade)</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {cycle.degraded_mode === "model_downgrade" && (
          <div>
            <Label className="text-xs font-medium">Model AI Fallback</Label>
            <Select
              value={cycle.fallback_model || "gpt-4o-mini"}
              onValueChange={(val) => onChange({ ...cycle, fallback_model: val })}
            >
              <SelectTrigger className="h-8 text-xs mt-1 bg-background">
                <SelectValue placeholder="Pilih model" />
              </SelectTrigger>
              <SelectContent>
                {FALLBACK_MODELS.map((m) => (
                  <SelectItem key={m.value} value={m.value}>
                    {m.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}
      </div>

      {cycle.degraded_mode === "throttle" && (
        <div className="grid grid-cols-2 gap-3 pt-1">
          <div>
            <Label className="text-xs font-medium">Maks Permintaan</Label>
            <NumberInput
              min={1}
              className="h-8 text-xs mt-1 bg-background border-2 border-foreground"
              value={cycle.throttle_rate_amount ?? 5}
              onChange={(val) =>
                onChange({ ...cycle, throttle_rate_amount: val ?? 1 })
              }
            />
          </div>
          <div>
            <Label className="text-xs font-medium">Jendela Waktu (Detik)</Label>
            <NumberInput
              min={1}
              className="h-8 text-xs mt-1 bg-background border-2 border-foreground"
              value={cycle.throttle_rate_window_secs ?? 60}
              onChange={(val) =>
                onChange({ ...cycle, throttle_rate_window_secs: val ?? 1 })
              }
            />
          </div>
        </div>
      )}

      {cycle.degraded_mode !== "hard_block" && (
        <div>
          <Label className="text-xs font-medium">Maks Penggunaan Mode Degraded (Opsional)</Label>
          <NumberInput
            placeholder="Kosongkan untuk tanpa batas"
            className="h-8 text-xs mt-1 bg-background border-2 border-foreground"
            value={cycle.max_degraded_usage ?? undefined}
            onChange={(val) => {
              onChange({ ...cycle, max_degraded_usage: val ?? null })
            }}
          />
        </div>
      )}
    </div>
  )
}
