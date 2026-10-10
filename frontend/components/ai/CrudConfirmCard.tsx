import { Button } from "@/components/ui/button";

interface CrudConfirmCardProps {
  type: string;
  category: string;
  description: string;
  amount: number;
  date: string;
  onConfirm: () => void;
  onCancel: () => void;
  isLoading?: boolean;
}

export function CrudConfirmCard({
  type,
  category,
  description,
  amount,
  date,
  onConfirm,
  onCancel,
  isLoading,
}: CrudConfirmCardProps) {
  const formatRupiah = (val: number) => {
    return new Intl.NumberFormat("id-ID", {
      style: "currency",
      currency: "IDR",
      minimumFractionDigits: 0,
    }).format(val || 0);
  };

  return (
    <div className="flex w-full justify-start mt-2">
      <div className="bg-white border border-primary-200 rounded-lg p-4 shadow-sm w-full max-w-sm">
        <div className="flex justify-between items-start mb-2">
          <h3 className="font-semibold text-neutral-900">Konfirmasi Transaksi</h3>
          <button onClick={onCancel} className="text-neutral-400 hover:text-neutral-600">
            ✕
          </button>
        </div>
        <div className="text-sm text-neutral-600 mb-4">
          <p className="mb-1">
            <span className="font-medium capitalize">{type}</span> · {category}
          </p>
          <p className="font-medium text-neutral-900">{description}</p>
          <p>
            {formatRupiah(amount)} · {date}
          </p>
        </div>
        <div className="flex justify-end gap-2">
          <Button variant="outline" size="sm" onClick={onCancel} disabled={isLoading}>
            Batalkan
          </Button>
          <Button size="sm" onClick={onConfirm} disabled={isLoading}>
            {isLoading ? "Menyimpan..." : "Konfirmasi & Simpan"}
          </Button>
        </div>
      </div>
    </div>
  );
}
