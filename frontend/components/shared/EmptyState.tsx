import type { ReactNode } from "react";
import { Inbox } from "lucide-react";

/**
 * Standard empty-state block for lists and tables that returned no rows.
 */
export function EmptyState({
  title = "Belum ada data",
  description = "Data akan muncul di sini setelah Anda menambahkannya.",
  action,
}: {
  title?: string;
  description?: string;
  action?: ReactNode;
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 border-2 rounded-lg border-dashed border-foreground/40 bg-card py-12 px-6 text-center">
      <div className="flex h-12 w-12 items-center justify-center border-2 rounded-md border-foreground bg-muted">
        <Inbox className="h-6 w-6 text-muted-foreground" aria-hidden="true" />
      </div>
      <p className="text-sm font-black uppercase tracking-wide text-foreground">{title}</p>
      <p className="max-w-sm text-xs font-bold text-muted-foreground">{description}</p>
      {action}
    </div>
  );
}
