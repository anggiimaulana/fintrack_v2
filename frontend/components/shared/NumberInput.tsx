"use client"

import React, { useState } from "react"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"

export interface NumberInputProps
  extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "onChange" | "value"> {
  value?: number | null
  onChange?: (value: number | undefined) => void
  allowNegative?: boolean
  allowZero?: boolean
  suffix?: string
}

function groupThousands(digits: string): string {
  const clean = digits.replace(/[^0-9]/g, "").replace(/^0+(?=\d)/, "")
  if (!clean) return "0"
  return clean.replace(/\B(?=(\d{3})+(?!\d))/g, ".")
}

export function formatNumberWithDots(val: number | string | null | undefined): string {
  if (val === undefined || val === null || val === "") return ""
  const str = String(val).trim()
  if (str === "-1") return "-1"
  const isNeg = str.startsWith("-")
  const rawDigits = str.replace(/[^0-9]/g, "")
  if (!rawDigits) return isNeg ? "-" : ""
  const formatted = groupThousands(rawDigits)
  return isNeg ? `-${formatted}` : formatted
}

export const NumberInput = React.forwardRef<HTMLInputElement, NumberInputProps>(
  (
    {
      className,
      value,
      onChange,
      allowNegative = false,
      allowZero = true,
      suffix,
      placeholder = "0",
      onBlur,
      disabled,
      ...props
    },
    ref
  ) => {
    const [draft, setDraft] = useState<string | null>(null)

    const displayValue =
      draft ??
      (value !== undefined && value !== null
        ? value === 0 && !allowZero
          ? ""
          : formatNumberWithDots(value)
        : "")

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const raw = e.target.value
      const isNeg = allowNegative && raw.trim().startsWith("-")
      const digits = raw.replace(/[^0-9]/g, "").replace(/^0+(?=\d)/, "")

      if (!digits) {
        if (isNeg) {
          setDraft("-")
          onChange?.(-1)
        } else {
          setDraft("")
          onChange?.(allowZero ? 0 : undefined)
        }
        return
      }

      const formatted = isNeg ? `-${groupThousands(digits)}` : groupThousands(digits)
      setDraft(formatted)

      let num = parseInt(digits, 10)
      if (isNeg) num = -num
      onChange?.(num)
    }

    const handleBlur = (e: React.FocusEvent<HTMLInputElement>) => {
      setDraft(null)
      onBlur?.(e)
    }

    if (suffix) {
      return (
        <div className="relative w-full">
          <Input
            {...props}
            ref={ref}
            type="text"
            inputMode="numeric"
            value={displayValue}
            onChange={handleChange}
            onBlur={handleBlur}
            placeholder={placeholder}
            disabled={disabled}
            className={cn("font-mono pr-12", className)}
          />
          <span className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground text-xs font-bold pointer-events-none select-none">
            {suffix}
          </span>
        </div>
      )
    }

    return (
      <Input
        {...props}
        ref={ref}
        type="text"
        inputMode="numeric"
        value={displayValue}
        onChange={handleChange}
        onBlur={handleBlur}
        placeholder={placeholder}
        disabled={disabled}
        className={cn("font-mono", className)}
      />
    )
  }
)

NumberInput.displayName = "NumberInput"
