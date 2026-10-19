"use client"

import React, { useState } from "react"
import { Input } from "@/components/ui/input"

export interface RupiahInputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "onChange" | "value"> {
  value?: number
  onChange?: (value: number | undefined) => void
  min?: number
  allowZero?: boolean
  errorMessage?: string
  /** Currency symbol shown before the field. Defaults to Rupiah. */
  symbol?: string
  /**
   * Minor-unit precision. 0 (default) keeps the original integer-only Rupiah
   * behaviour; >0 allows a comma decimal separator so currencies like USD can be
   * entered exactly.
   */
  decimals?: number
}

const GROUP_SEP = "."
const DEC_SEP = ","

function groupThousands(intPart: string): string {
  const clean = intPart.replace(/[^0-9]/g, "").replace(/^0+(?=\d)/, "")
  return clean.replace(/\B(?=(\d{3})+(?!\d))/g, GROUP_SEP)
}

function formatValue(value: number | undefined, decimals: number): string {
  if (value == null) return ""
  const fixed = decimals > 0 ? value.toFixed(decimals) : String(Math.trunc(value))
  const [int, frac] = fixed.split(".")
  const grouped = groupThousands(int)
  return frac ? `${grouped}${DEC_SEP}${frac}` : grouped
}

export const RupiahInput = React.forwardRef<HTMLInputElement, RupiahInputProps>(
  (
    {
      className,
      value,
      onChange,
      min = 1,
      allowZero = false,
      errorMessage,
      symbol = "Rp",
      decimals = 0,
      placeholder = "0",
      onBlur,
      ...props
    },
    ref
  ) => {
    const digits = Math.min(4, Math.max(0, Math.trunc(decimals)))
    const [draft, setDraft] = useState<string | null>(null)
    const effectiveMin = allowZero ? 0 : min
    const displayValue = draft ?? formatValue(value, digits)

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const raw = e.target.value.replace(/[^0-9,]/g, "")
      const [intRaw, ...fracParts] = raw.split(DEC_SEP)
      const int = intRaw.replace(/^0+(?=\d)/, "")
      const frac = digits > 0 ? fracParts.join("").slice(0, digits) : ""
      // Keep a trailing separator while the fraction is still empty, otherwise
      // typing "1000," would drop the comma and the next keypress would land in
      // the integer part.
      const hasSep = digits > 0 && raw.includes(DEC_SEP)
      setDraft(hasSep ? `${groupThousands(int)}${DEC_SEP}${frac}` : groupThousands(int))

      if (!int && !frac) {
        onChange?.(undefined)
        return
      }
      const num = Number(`${int || "0"}.${frac || "0"}`)
      onChange?.(Number.isFinite(num) ? num : undefined)
    }

    const isInvalid = value !== undefined && value < effectiveMin

    const handleBlur = (e: React.FocusEvent<HTMLInputElement>) => {
      setDraft(null)
      onBlur?.(e)
    }

    return (
      <div className="space-y-1 w-full">
        <div className="relative">
          <span className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground font-bold text-xs select-none">
            {symbol}
          </span>
          <Input
            {...props}
            ref={ref}
            type="text"
            inputMode={digits > 0 ? "decimal" : "numeric"}
            value={displayValue}
            onChange={handleChange}
            onBlur={handleBlur}
            placeholder={placeholder}
            className={`pl-9 font-mono ${isInvalid ? "border-destructive focus-visible:ring-destructive" : ""} ${className || ""}`}
          />
        </div>
        {isInvalid && (
          <p role="alert" className="text-xs font-semibold text-destructive">
            {errorMessage || `Minimal input adalah ${effectiveMin.toLocaleString("id-ID")}`}
          </p>
        )}
      </div>
    )
  }
)

RupiahInput.displayName = "RupiahInput"
