import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { Slot } from "radix-ui"

import { cn } from "@/lib/utils"

const buttonVariants = cva(
  "inline-flex shrink-0 items-center justify-center rounded-md border-2 border-foreground font-bold whitespace-nowrap transition-all outline-none select-none shadow-[3px_3px_0px_0px] shadow-foreground cursor-pointer active:shadow-none active:translate-x-[3px] active:translate-y-[3px] disabled:cursor-not-allowed disabled:opacity-50 disabled:shadow-none disabled:active:translate-x-0 disabled:active:translate-y-0",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground hover:bg-primary/90",
        outline: "bg-background text-foreground hover:bg-muted",
        secondary: "bg-secondary text-secondary-foreground hover:bg-secondary/80",
        ghost: "border-transparent shadow-none bg-transparent hover:bg-muted active:shadow-none active:translate-x-0 active:translate-y-0",
        destructive: "bg-destructive text-primary-foreground hover:bg-destructive/90",
        link: "border-transparent shadow-none bg-transparent text-primary underline-offset-4 hover:underline active:shadow-none active:translate-x-0 active:translate-y-0",
      },
      size: {
        default: "h-9 gap-2 px-4 text-sm",
        xs: "h-6 gap-1 px-2 text-xs",
        sm: "h-7 gap-1.5 px-3 text-xs",
        lg: "h-10 gap-2.5 px-6 text-base",
        icon: "h-9 w-9",
        "icon-xs": "h-6 w-6",
        "icon-sm": "h-7 w-7",
        "icon-lg": "h-10 w-10",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

function Button({
  className,
  variant = "default",
  size = "default",
  asChild = false,
  ...props
}: React.ComponentProps<"button"> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean
  }) {
  const Comp = asChild ? Slot.Root : "button"

  return (
    <Comp
      data-slot="button"
      data-variant={variant}
      data-size={size}
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  )
}

export { Button, buttonVariants }
