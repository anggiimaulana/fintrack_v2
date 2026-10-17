import Link from "next/link";
import { cn } from "@/lib/utils";

/**
 * Neobrutalist action link for the marketing surface. A plain `Link` (not the
 * app `Button`) so the accent tones can replace the button background entirely
 * — Tailwind class conflicts would otherwise be order-dependent.
 */

export type ActionTone = "acid" | "outline" | "outlineOnDark";
export type ActionSize = "sm" | "md" | "lg";

const BASE =
  "inline-flex shrink-0 items-center justify-center gap-2 rounded-md border-[3px] font-bold transition-transform " +
  "active:translate-x-[3px] active:translate-y-[3px] active:shadow-none " +
  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background";

/** Geometry only; the tone supplies the shadow colour. */
const SIZES: Record<ActionSize, string> = {
  sm: "h-9 px-4 text-sm shadow-[3px_3px_0px_0px]",
  md: "h-11 px-6 text-sm shadow-[4px_4px_0px_0px]",
  lg: "h-12 px-7 text-base shadow-[4px_4px_0px_0px]",
};

const TONES: Record<ActionTone, string> = {
  acid: "border-foreground bg-accent-acid text-accent-acid-foreground shadow-foreground hover:bg-accent-acid-strong",
  outline: "border-foreground bg-background text-foreground shadow-foreground hover:bg-muted",
  outlineOnDark: "border-white bg-transparent text-white shadow-white hover:bg-white/10",
};

export function ActionLink({
  href,
  children,
  tone = "acid",
  size = "md",
  className,
}: {
  href: string;
  children: React.ReactNode;
  tone?: ActionTone;
  size?: ActionSize;
  className?: string;
}) {
  return (
    <Link href={href} className={cn(BASE, SIZES[size], TONES[tone], className)}>
      {children}
    </Link>
  );
}
