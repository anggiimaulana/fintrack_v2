"use client"

import React, { useState } from "react"
import { Input } from "@/components/ui/input"

export interface RupiahInputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "onChange" | "value"> {
  value?: number
  onChange?: (value: number | undefined) => void
  min?: number
  allowZero?: boolean
  errorMessage?: string
}

function groupThousands(digits: string): string {
  const clean = digits.replace(/[^0-9]/g, "").replace(/^0+(?=\d)/, "")
  return clean.replace(/\B(?=(\d{3})+(?!\d))/g, ".")
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
      placeholder = "0",
      onBlur,
      ...props
    },
    ref
  ) => {
    const [draft, setDraft] = useState<string | null>(null)
    const effectiveMin = allowZero ? 0 : min
    const displayValue = draft ?? (value != null ? groupThousands(String(value)) : "")

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const cleanVal = e.target.value.replace(/[^0-9]/g, "").replace(/^0+(?=\d)/, "")
      setDraft(groupThousands(cleanVal))
      const num = cleanVal ? parseInt(cleanVal, 10) : undefined
      onChange?.(num)
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
            Rp
          </span>
          <Input
            {...props}
            ref={ref}
            type="text"
            inputMode="numeric"
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
