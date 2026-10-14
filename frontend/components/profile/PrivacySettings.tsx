"use client";

import { useState } from "react";
import { Download, Trash2, ShieldAlert } from "lucide-react";
import { toast } from "sonner";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import api from "@/lib/api";

/**
 * Data-subject controls required by UU PDP No. 27/2022: export a copy of the
 * user's data (Pasal 21) and request account/data destruction (Pasal 43).
 */
export function PrivacySettings() {
  const router = useRouter();
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [deletePassword, setDeletePassword] = useState("");
  const [deleteReason, setDeleteReason] = useState("");
  const [isExporting, setIsExporting] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  const handleExportData = async () => {
    try {
      setIsExporting(true);
      const res = await api.get("/user/export-data");
      const blob = new Blob([JSON.stringify(res.data?.data, null, 2)], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `fintrack-data-export-${new Date().toISOString().slice(0, 10)}.json`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
      toast.success("Salinan data pribadi berhasil diunduh.");
    } catch {
      toast.error("Gagal mengunduh salinan data pribadi.");
    } finally {
      setIsExporting(false);
    }
  };

  const handleDeleteAccount = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!deletePassword) {
      toast.error("Password konfirmasi wajib diisi.");
      return;
    }
    try {
      setIsDeleting(true);
      await api.delete("/me", { data: { password: deletePassword, reason: deleteReason || "Permintaan Pengguna (UU PDP)" } });
      toast.success("Akun dan data pribadi Anda telah dihapus secara permanen.");
      setDeleteModalOpen(false);
      router.push("/login");
    } catch {
      toast.error("Gagal menghapus akun. Pastikan password Anda benar.");
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-start gap-2">
        <ShieldAlert className="w-5 h-5 text-primary shrink-0 mt-0.5" />
        <p className="text-xs font-bold text-muted-foreground leading-relaxed">
          Sebagai subjek data, Anda berhak memperoleh salinan data pribadi Anda (Pasal 21) dan mengajukan pemusnahan data akun (Pasal 43).
        </p>
      </div>

      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pt-3 border-t-2 border-foreground/10">
        <div>
          <p className="text-sm font-black text-foreground">Ekspor Salinan Data Pribadi</p>
          <p className="text-xs font-bold text-muted-foreground">Unduh seluruh profil, transaksi, histori AI, dan invoice dalam format JSON.</p>
        </div>
        <Button type="button" variant="outline" onClick={handleExportData} disabled={isExporting} className="shrink-0">
          <Download className="w-4 h-4 mr-2" />
          {isExporting ? "Mengunduh..." : "Ekspor Data (JSON)"}
        </Button>
      </div>

      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pt-3 border-t-2 border-foreground/10">
        <div>
          <p className="text-sm font-black text-destructive">Hapus Akun &amp; Musnahkan Data</p>
          <p className="text-xs font-bold text-muted-foreground">
            Hapus chat AI &amp; vektor permanen, pseudonymisasi akun (faktur pajak disimpan anonim 10 thn sesuai UU KUP).
          </p>
        </div>
        <Button type="button" variant="destructive" onClick={() => setDeleteModalOpen(true)} className="shrink-0">
          <Trash2 className="w-4 h-4 mr-2" />
          Hapus Akun
        </Button>
      </div>

      <Dialog open={deleteModalOpen} onOpenChange={setDeleteModalOpen}>
        <DialogContent className="border-2 border-foreground shadow-[6px_6px_0px_0px] shadow-foreground bg-card sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-lg font-black text-destructive flex items-center gap-2">
              <ShieldAlert className="w-5 h-5 text-destructive" /> Konfirmasi Penghapusan Akun
            </DialogTitle>
            <DialogDescription className="text-xs font-bold text-muted-foreground mt-2">
              Tindakan ini permanen sesuai UU PDP No. 27/2022 Pasal 43. Riwayat chat dan vektor AI akan dihapus tuntas. Masukkan password Anda untuk mengonfirmasi.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleDeleteAccount} className="space-y-4 py-2">
            <div className="space-y-1.5">
              <label htmlFor="delete-account-password" className="text-xs font-black uppercase text-foreground">
                Password Konfirmasi
              </label>
              <input
                id="delete-account-password"
                type="password"
                value={deletePassword}
                onChange={(e) => setDeletePassword(e.target.value)}
                placeholder="Masukkan password Anda"
                required
                className="w-full border-2 border-foreground bg-background px-3 py-2 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none"
              />
            </div>
            <div className="space-y-1.5">
              <label htmlFor="delete-account-reason" className="text-xs font-black uppercase text-foreground">
                Alasan (Opsional)
              </label>
              <input
                id="delete-account-reason"
                type="text"
                value={deleteReason}
                onChange={(e) => setDeleteReason(e.target.value)}
                placeholder="Alasan penutupan akun"
                className="w-full border-2 border-foreground bg-background px-3 py-2 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none"
              />
            </div>
            <DialogFooter className="gap-2 sm:gap-0 mt-4">
              <Button type="button" variant="outline" onClick={() => setDeleteModalOpen(false)}>
                Batal
              </Button>
              <Button type="submit" variant="destructive" disabled={isDeleting}>
                {isDeleting ? "Memproses..." : "Ya, Hapus Akun Permanen"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
