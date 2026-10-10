import { formatRupiah } from "@/lib/format";
import { cn } from "@/lib/utils";

/**
 * Renders a Rupiah amount in a monospace face so figures align across the UI.
 * Every currency value should go through this component.
 */
export function Money({
  value,
  className,
  signed = false,
}: {
  value: number | string | null | undefined;
  className?: string;
  signed?: boolean;
}) {
  const n = typeof value === "string" ? Number(value) : (value ?? 0);
  const prefix = signed && n > 0 ? "+" : "";
  return (
    <span className={cn("font-mono tabular-nums", className)}>
      {prefix}
      {formatRupiah(n)}
    </span>
  );
}
