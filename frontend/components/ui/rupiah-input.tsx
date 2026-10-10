"use client";

import { InputHTMLAttributes, forwardRef, useCallback } from "react";

interface RupiahInputProps extends Omit<InputHTMLAttributes<HTMLInputElement>, "onChange" | "value"> {
  value: number;
  onChange: (value: number) => void;
}

export const RupiahInput = forwardRef<HTMLInputElement, RupiahInputProps>(
  ({ value, onChange, className = "", ...props }, ref) => {
    const formatRupiah = useCallback((num: number) => {
      if (isNaN(num)) return "";
      return new Intl.NumberFormat("id-ID").format(num);
    }, []);

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const raw = e.target.value.replace(/\D/g, "");
      const num = parseInt(raw, 10);
      onChange(isNaN(num) ? 0 : num);
    };

    const displayValue = value > 0 ? formatRupiah(value) : value === 0 ? "" : formatRupiah(value);

    return (
      <div className="relative w-full">
        <span className="absolute left-4 top-1/2 -translate-y-1/2 text-lg font-black text-foreground font-mono pointer-events-none z-10">
          Rp
        </span>
        <input
          ref={ref}
          type="text"
          inputMode="numeric"
          value={displayValue}
          onChange={handleChange}
          className={`w-full font-mono text-lg border-2 border-foreground bg-background pl-14 pr-4 py-2 focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none placeholder:text-muted-foreground/70 ${className}`}
          placeholder="0"
          {...props}
        />
      </div>
    );
  }
);

RupiahInput.displayName = "RupiahInput";
