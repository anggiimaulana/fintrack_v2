import { AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui/button";

/**
 * Standard error-state block with a retry action, shown when a query fails.
 */
export function ErrorState({
  title = "Gagal memuat data",
  description = "Terjadi kesalahan saat mengambil data. Silakan coba lagi.",
  onRetry,
}: {
  title?: string;
  description?: string;
  onRetry?: () => void;
}) {
  return (
    <div
      role="alert"
      className="flex flex-col items-center justify-center gap-3 border-2 border-destructive bg-destructive/10 py-12 px-6 text-center"
    >
      <div className="flex h-12 w-12 items-center justify-center border-2 border-destructive bg-background">
        <AlertTriangle className="h-6 w-6 text-destructive" aria-hidden="true" />
      </div>
      <p className="text-sm font-black uppercase tracking-wide text-foreground">{title}</p>
      <p className="max-w-sm text-xs font-bold text-muted-foreground">{description}</p>
      {onRetry && (
        <Button variant="outline" onClick={onRetry}>
          Coba lagi
        </Button>
      )}
    </div>
  );
}
