"use client";

import { useState } from "react";
import { User, Mail, Phone, Save, Download, Trash2, ShieldAlert } from "lucide-react";
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
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { useSession } from "@/lib/session-context";
export default function ProfilePage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { user: sessionUser } = useSession();
  const { data: profile, isLoading } = useQuery({
    queryKey: ["profile", "me"],
    queryFn: async () => {
      const res = await api.get("/me");
      return res.data?.data;
    },
  });

  const [formOverride, setFormOverride] = useState<{ fullName?: string; phone?: string; avatarUrl?: string }>({});

  const fullName = formOverride.fullName ?? profile?.full_name ?? "";
  const phone = formOverride.phone ?? profile?.phone ?? "";
  const avatarUrl = formOverride.avatarUrl ?? profile?.avatar_url ?? "";

  const setFullName = (v: string) => setFormOverride((s) => ({ ...s, fullName: v }));
  const setPhone = (v: string) => setFormOverride((s) => ({ ...s, phone: v }));

  const email = sessionUser?.email ?? profile?.email ?? "";


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
  const updateMutation = useMutation({
    mutationFn: async (data: Record<string, unknown>) => {
      const res = await api.put("/me", data);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["profile", "me"] });
      queryClient.invalidateQueries({ queryKey: ["session", "me"] });
    },
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await updateMutation.mutateAsync({ full_name: fullName, phone: phone || null, avatar_url: avatarUrl || null });
      toast.success("Profil berhasil diperbarui!");
    } catch {
      toast.error("Gagal memperbarui profil.");
    }
  };

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div>
        <h2 className="text-xl font-black uppercase tracking-wide text-foreground">Profil Saya</h2>
        <p className="text-xs font-bold text-muted-foreground mt-1">Kelola informasi profil akun Anda.</p>
      </div>

      <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-6">
        {isLoading ? (
          <div className="space-y-4">
            {[1,2,3,4].map(i => (
              <div key={i} className="h-10 bg-muted border-2 border-foreground animate-pulse" />
            ))}
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-5">
            <div className="flex items-center gap-4 mb-6">
              <div className="w-16 h-16 border-2 border-foreground bg-primary/10 flex items-center justify-center">
                <User className="w-8 h-8 text-primary" />
              </div>
              <div>
                <p className="font-black text-lg text-foreground">{fullName || "User"}</p>
                <p className="text-sm font-bold text-muted-foreground">{email}</p>
              </div>
            </div>

            <div className="space-y-2">
              <label htmlFor="profile-full-name" className="text-xs font-bold uppercase tracking-wide text-foreground flex items-center gap-2">
                <User className="w-4 h-4" /> Nama Lengkap
              </label>
              <input
                id="profile-full-name"
                type="text"
                value={fullName}
                onChange={(e) => setFullName(e.target.value)}
                placeholder="Nama lengkap Anda"
                className="w-full border-2 border-foreground bg-background px-4 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none placeholder:text-muted-foreground/70"
              />
            </div>

            <div className="space-y-2">
              <label htmlFor="profile-email" className="text-xs font-bold uppercase tracking-wide text-foreground flex items-center gap-2">
                <Mail className="w-4 h-4" /> Email
              </label>
              <input
                id="profile-email"
                type="email"
                value={email}
                disabled
                className="w-full border-2 border-foreground bg-muted px-4 py-2.5 text-sm font-bold text-muted-foreground cursor-not-allowed"
              />
              <p className="text-xs font-bold text-muted-foreground">Email tidak dapat diubah. Hubungi admin jika perlu perubahan.</p>
            </div>

            <div className="space-y-2">
              <label htmlFor="profile-phone" className="text-xs font-bold uppercase tracking-wide text-foreground flex items-center gap-2">
                <Phone className="w-4 h-4" /> Nomor Telepon
              </label>
              <input
                id="profile-phone"
                type="tel"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                placeholder="08xxxxxxxxxx"
                className="w-full border-2 border-foreground bg-background px-4 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none placeholder:text-muted-foreground/70"
              />
            </div>

            <Button type="submit" disabled={updateMutation.isPending} className="w-full">
              <Save className="w-4 h-4 mr-2" />
              {updateMutation.isPending ? "Menyimpan..." : "Simpan Perubahan"}
            </Button>
          </form>
        )}
      </div>

      {/* Kepatuhan Regulasi UU PDP No. 27/2022 */}
      <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-6 space-y-4">
        <div className="flex items-center gap-2">
          <ShieldAlert className="w-5 h-5 text-primary" />
          <h3 className="text-base font-black uppercase tracking-wide text-foreground">
            Privasi &amp; Kepatuhan UU PDP (No. 27/2022)
          </h3>
        </div>
        <p className="text-xs font-bold text-muted-foreground leading-relaxed">
          Sebagai subjek data, Anda berhak memperoleh salinan data pribadi Anda (Pasal 21) dan mengajukan pemusnahan data akun (Pasal 43).
        </p>

        <div className="pt-2 border-t-2 border-foreground/10 space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div>
              <p className="text-sm font-black text-foreground">Ekspor Salinan Data Pribadi</p>
              <p className="text-xs font-bold text-muted-foreground">Unduh seluruh profil, transaksi, histori AI, dan invoice dalam format JSON.</p>
            </div>
            <Button
              type="button"
              variant="outline"
              onClick={handleExportData}
              disabled={isExporting}
              className="shrink-0"
            >
              <Download className="w-4 h-4 mr-2" />
              {isExporting ? "Mengunduh..." : "Ekspor Data (JSON)"}
            </Button>
          </div>

          <div className="pt-3 border-t-2 border-foreground/10 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div>
              <p className="text-sm font-black text-destructive">Hapus Akun &amp; Musnahkan Data</p>
              <p className="text-xs font-bold text-muted-foreground">
                Hapus chat AI &amp; vektor permanen, pseudonymisasi akun (faktur pajak disimpan anonim 10 thn sesuai UU KUP).
              </p>
            </div>
            <Button
              type="button"
              variant="destructive"
              onClick={() => setDeleteModalOpen(true)}
              className="shrink-0"
            >
              <Trash2 className="w-4 h-4 mr-2" />
              Hapus Akun
            </Button>
          </div>
        </div>
      </div>

      {/* Modal Konfirmasi Hapus Akun */}
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
