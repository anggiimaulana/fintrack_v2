'use client'

import { cn } from '@/lib/utils'

const PRESET_COLORS = [
  '#3b5bdb', // primary
  '#10b981', // success
  '#f43f5e', // danger
  '#f59e0b', // warning
  '#06b6d4', // cyan
  '#8b5cf6', // violet
  '#ec4899', // pink
  '#84cc16', // lime
  '#f97316', // orange
  '#475569', // slate
]

interface ColorPickerProps {
  value: string
  onChange: (value: string) => void
}

export function ColorPicker({ value, onChange }: ColorPickerProps) {
  return (
    <div className="flex flex-wrap gap-2 p-1 border rounded-md">
      {PRESET_COLORS.map((color) => (
        <button
          key={color}
          type="button"
          className={cn(
            "h-8 w-8 rounded-full border-2 transition-transform hover:scale-110",
            value === color ? "border-neutral-900 scale-110" : "border-transparent"
          )}
          style={{ backgroundColor: color }}
          onClick={() => onChange(color)}
        />
      ))}
    </div>
  )
}
