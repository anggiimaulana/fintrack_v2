"use client"

import React, { useState } from "react"
import { Input } from "@/components/ui/input"

interface RupiahInputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "onChange" | "value"> {
  value?: number
  onChange?: (value: number | undefined) => void
}

function groupThousands(digits: string): string {
  const clean = digits.replace(/[^0-9]/g, "")
  return clean.replace(/\B(?=(\d{3})+(?!\d))/g, ".")
}

export const RupiahInput = React.forwardRef<HTMLInputElement, RupiahInputProps>(
  ({ className, value, onChange, placeholder = "0", ...props }, ref) => {
    // `draft` holds what the user is typing; when absent the display follows
    // the controlled `value` prop directly (no effect-driven state sync).
    const [draft, setDraft] = useState<string | null>(null)
    const displayValue = draft ?? (value != null ? groupThousands(String(value)) : "")

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const cleanVal = e.target.value.replace(/[^0-9]/g, "")
      setDraft(groupThousands(cleanVal))
      onChange?.(cleanVal ? parseInt(cleanVal, 10) : undefined)
    }

    return (
      <div className="relative">
        <span className="absolute left-3 top-1/2 -translate-y-1/2 text-neutral-500 font-medium">Rp</span>
        <Input
          {...props}
          ref={ref}
          type="text"
          value={displayValue}
          onChange={handleChange}
          onBlur={() => setDraft(null)}
          placeholder={placeholder}
          className={`pl-9 font-mono ${className || ""}`}
        />
      </div>
    )
  }
)

RupiahInput.displayName = "RupiahInput"
