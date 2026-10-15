import { LucideIcon } from 'lucide-react'
import { Skeleton } from '@/components/ui/skeleton'

export function AdminStatCardSkeleton() {
  return (
    <div className="border-2 rounded-lg border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-6">
      <div className="flex items-center gap-4">
        <Skeleton className="p-3 border-2 rounded-md border-foreground w-[56px] h-[56px]" />
        <div>
          <Skeleton className="h-3 w-20 border-2 rounded-md border-foreground mb-1" />
          <Skeleton className="h-7 w-24 border-2 rounded-md border-foreground" />
        </div>
      </div>
    </div>
  )
}

interface AdminStatCardProps {
  label: string
  value: string | number
  icon: LucideIcon
  colorClassName?: string
}

export function AdminStatCard({ label, value, icon: Icon, colorClassName = 'text-primary' }: AdminStatCardProps) {
  return (
    <div className="border-2 rounded-lg border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-6">
      <div className="flex items-center gap-4">
        <div className={`p-3 border-2 rounded-md border-foreground bg-background ${colorClassName}`}>
          <Icon size={24} />
        </div>
        <div>
          <p className="text-xs font-black uppercase tracking-wide text-muted-foreground">{label}</p>
          <h3 className="text-2xl font-black tracking-tight text-foreground font-mono">
            {value}
          </h3>
        </div>
      </div>
    </div>
  )
}
