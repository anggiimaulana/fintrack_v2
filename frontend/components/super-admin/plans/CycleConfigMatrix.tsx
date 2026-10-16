"use client"

import * as React from "react"
import { Input } from "@/components/ui/input"
import { NumberInput } from "@/components/shared/NumberInput"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { DegradedParamsForm } from "./DegradedParamsForm"
import type { PlanLimitCycle } from "@/hooks/usePlanCycles"
import type { CatalogLimit } from "@/hooks/useEntitlementCatalog"

interface CycleConfigMatrixProps {
  limits: CatalogLimit[]
  cycles: PlanLimitCycle[]
  onChange: (updatedCycles: PlanLimitCycle[]) => void
}

export function CycleConfigMatrix({ limits, cycles, onChange }: CycleConfigMatrixProps) {
  // Map existing cycles by limit_key
  const cyclesByKey = React.useMemo(() => {
    const record: Record<string, PlanLimitCycle> = {}
    for (const c of cycles) {
      record[c.limit_key] = c
    }
    return record
  }, [cycles])

  const handleCycleChange = (limitKey: string, updated: PlanLimitCycle) => {
    const next = [...cycles]
    const idx = next.findIndex((c) => c.limit_key === limitKey)
    if (idx >= 0) {
      next[idx] = updated
    } else {
      next.push(updated)
    }
    onChange(next)
  }

  return (
    <div className="space-y-4">
      <div className="text-sm text-muted-foreground">
        Konfigurasi siklus reset kuota mingguan dan perilaku pembatasan (degraded mode) saat kuota mingguan habis.
      </div>

      <div className="space-y-3">
        {limits.map((l) => {
          const currentCycle: PlanLimitCycle = cyclesByKey[l.key] || {
            limit_key: l.key,
            cycle_type: "monthly_only",
          }
          const isWeekly = currentCycle.cycle_type === "weekly_monthly"

          return (
            <div
              key={l.key}
              className="p-3 border rounded-md bg-card space-y-2 transition-colors hover:border-foreground/40"
            >
              <div className="flex flex-col md:flex-row md:items-center justify-between gap-3">
                <div className="space-y-0.5">
                  <div className="font-semibold text-sm">{l.label}</div>
                  <div className="text-xs text-muted-foreground font-mono">
                    {l.key} {l.unit ? `(${l.unit})` : ""}
                  </div>
                </div>

                <div className="flex items-center gap-3">
                  <div>
                    <Label className="text-xs sr-only">Tipe Siklus</Label>
                    <Select
                      value={currentCycle.cycle_type}
                      onValueChange={(val: "monthly_only" | "weekly_monthly") => {
                        const isToken = l.key.includes("token");
                        const defaultLimit = isToken ? 50000 : l.key.includes("scan") ? 25 : 500;
                        const defaultDegraded = isToken ? "model_downgrade" : "hard_block";
                        const defaultModel = isToken ? "gpt-4o-mini" : null;

                        handleCycleChange(l.key, {
                          ...currentCycle,
                          cycle_type: val,
                          weekly_limit:
                            val === "weekly_monthly"
                              ? currentCycle.weekly_limit && currentCycle.weekly_limit > 0
                                ? currentCycle.weekly_limit
                                : defaultLimit
                              : null,
                          degraded_mode:
                            val === "weekly_monthly"
                              ? currentCycle.degraded_mode || defaultDegraded
                              : null,
                          fallback_model:
                            val === "weekly_monthly" &&
                            (currentCycle.degraded_mode || defaultDegraded) === "model_downgrade"
                              ? currentCycle.fallback_model || defaultModel
                              : null,
                        });
                      }}
                    >
                      <SelectTrigger className="h-8 text-xs w-[170px]">
                        <SelectValue placeholder="Pilih siklus" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="monthly_only">Bulanan Saja</SelectItem>
                        <SelectItem value="weekly_monthly">Mingguan + Bulanan</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  {isWeekly && (
                    <div className="flex items-center gap-1.5">
                      <Label className="text-xs whitespace-nowrap">Batas/Minggu:</Label>
                      <NumberInput
                        min={1}
                        className="h-8 text-xs w-28 border-2 border-foreground"
                        value={currentCycle.weekly_limit ?? undefined}
                        onChange={(val) => {
                          handleCycleChange(l.key, { ...currentCycle, weekly_limit: val ?? 0 })
                        }}
                        placeholder="0"
                      />
                    </div>
                  )}
                </div>
              </div>

              {isWeekly && (
                <DegradedParamsForm
                  cycle={currentCycle}
                  onChange={(upd) => handleCycleChange(l.key, upd)}
                />
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
