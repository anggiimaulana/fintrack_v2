import * as React from "react"

import { cn } from "@/lib/utils"

function Textarea({ className, ...props }: React.ComponentProps<"textarea">) {
  return (
    <textarea
      data-slot="textarea"
      className={cn(
        "flex min-h-20 w-full min-w-0 rounded-md border-2 border-foreground bg-background px-3 py-2 text-sm text-foreground transition-all outline-none placeholder:text-muted-foreground/70 focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50 aria-invalid:border-destructive",
        className
      )}
      {...props}
    />
  )
}

export { Textarea }
