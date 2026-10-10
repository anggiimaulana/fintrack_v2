'use client'

import * as Icons from 'lucide-react'
import { LucideIcon } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

const RELEVANT_ICONS = [
  'Wallet', 'Banknote', 'Coins', 'CreditCard', 'PiggyBank',
  'ShoppingBag', 'ShoppingCart', 'Utensils', 'Car', 'Bus',
  'Home', 'Wrench', 'Lightbulb', 'Phone', 'Wifi',
  'Monitor', 'Gamepad2', 'Music', 'Film', 'Plane',
  'HeartPulse', 'Stethoscope', 'Pill', 'GraduationCap', 'Book',
  'Briefcase', 'TrendingUp', 'TrendingDown', 'Gift', 'Coffee'
]

interface IconPickerProps {
  value: string
  onChange: (value: string) => void
}

export function IconPicker({ value, onChange }: IconPickerProps) {
  return (
    <div className="grid grid-cols-5 gap-2 p-1 border rounded-md">
      {RELEVANT_ICONS.map((iconName) => {
        const Icon = Icons[iconName as keyof typeof Icons] as LucideIcon
        return (
          <Button
            key={iconName}
            type="button"
            variant="ghost"
            size="icon"
            className={cn(
              "h-10 w-10 p-2",
              value === iconName ? "bg-primary-50 text-primary-600 border-primary-200 border" : "text-neutral-500"
            )}
            onClick={() => onChange(iconName)}
          >
            {Icon && <Icon size={20} />}
          </Button>
        )
      })}
    </div>
  )
}
