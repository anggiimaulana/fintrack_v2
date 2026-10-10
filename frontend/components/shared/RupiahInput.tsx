"use client"

import React, { useState, useEffect } from "react"
import { Input } from "@/components/ui/input"

interface RupiahInputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "onChange" | "value"> {
  value?: number
  onChange?: (value: number | undefined) => void
}

export const RupiahInput = React.forwardRef<HTMLInputElement, RupiahInputProps>(
  ({ className, value, onChange, placeholder = "0", ...props }, ref) => {
    const [displayValue, setDisplayValue] = useState<string>("")

    useEffect(() => {
      if (value !== undefined && value !== null) {
        setDisplayValue(formatRupiah(value.toString()))
      } else {
        setDisplayValue("")
      }
    }, [value])

    const formatRupiah = (angka: string) => {
      const number_string = angka.replace(/[^,\d]/g, "").toString()
      const split = number_string.split(",")
      const sisa = split[0].length % 3
      let rupiah = split[0].substr(0, sisa)
      const ribuan = split[0].substr(sisa).match(/\d{3}/gi)

      if (ribuan) {
        const separator = sisa ? "." : ""
        rupiah += separator + ribuan.join(".")
      }

      rupiah = split[1] !== undefined ? rupiah + "," + split[1] : rupiah
      return rupiah
    }

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      let val = e.target.value
      
      // Clean up the string to get just numbers
      const cleanVal = val.replace(/[^0-9]/g, "")
      
      // Update display value
      setDisplayValue(formatRupiah(cleanVal))
      
      // Call onChange with the raw number
      if (onChange) {
        onChange(cleanVal ? parseInt(cleanVal, 10) : undefined)
      }
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
          placeholder={placeholder}
          className={`pl-9 font-mono ${className || ""}`}
        />
      </div>
    )
  }
)

RupiahInput.displayName = "RupiahInput"
