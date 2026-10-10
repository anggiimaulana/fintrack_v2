import { cn } from "@/lib/utils"

function Skeleton({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="skeleton"
      className={cn("animate-pulse bg-muted border-2 border-foreground", className)}
      {...props}
    />
  )
}

export { Skeleton }
