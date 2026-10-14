"use client";

import { useRef, useState } from "react";
import { User, Mail, Phone, Save, Download, Trash2, ShieldAlert, Camera, Loader2 } from "lucide-react";
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
import type { AxiosError } from "axios";
import api from "@/lib/api";
import { useSession } from "@/lib/session-context";

const MAX_AVATAR_BYTES = 2 * 1024 * 1024;
const ACCEPTED_AVATAR = "image/png,image/jpeg,image/webp";

/** Extracts a human-readable message from an Axios error, falling back safely. */
function apiErrorMessage(err: unknown, fallback: string): string {
  const ax = err as AxiosError<{ message?: string }>;
  return ax?.response?.data?.message || fallback;
}

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

  const fileInputRef = useRef<HTMLInputElement>(null);

  const [formOverride, setFormOverride] = useState<{ fullName?: string; phone?: string }>({});

  const fullName = formOverride.fullName ?? profile?.full_name ?? "";
  const phone = formOverride.phone ?? profile?.phone ?? "";
  const avatarUrl: string = profile?.avatar_url ?? "";

  const setFullName = (v: string) => setFormOverride((s) => ({ ...s, fullName: v }));
  const setPhone = (v: string) => setFormOverride((s) => ({ ...s, phone: v }));

  const email = sessionUser?.email ?? profile?.email ?? "";

  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [deletePassword, setDeletePassword] = useState("");
  const [deleteReason, setDeleteReason] = useState("");
  const [isExporting, setIsExporting] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [isDragging, setIsDragging] = useState(false);

  const initials = (fullName || email || "US")
    .split(" ")
    .map((w: string) => w[0])
    .filter(Boolean)
    .slice(0, 2)
    .join("")
    .toUpperCase();

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
      // Refresh both the profile card and the session mirror so the Header /
      // Sidebar reflect a new name or avatar immediately.
      queryClient.invalidateQueries({ queryKey: ["profile", "me"] });
      queryClient.invalidateQueries({ queryKey: ["session", "me"] });
    },
  });

  const avatarMutation = useMutation({
    mutationFn: async (file: File) => {
      const fd = new FormData();
      fd.append("file", file);
      const res = await api.post("/me/avatar", fd);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["profile", "me"] });
      queryClient.invalidateQueries({ queryKey: ["session", "me"] });
      toast.success("Foto profil berhasil diperbarui!");
    },
    onError: (err) => toast.error(apiErrorMessage(err, "Gagal mengunggah foto profil.")),
  });

  const submitAvatar = (file: File | undefined) => {
    if (!file) return;
    if (!ACCEPTED_AVATAR.split(",").includes(file.type)) {
      toast.error("Format tidak didukung. Gunakan JPG, PNG, atau WEBP.");
      return;
    }
    if (file.size > MAX_AVATAR_BYTES) {
      toast.error("Ukuran foto melebihi 2 MB.");
      return;
    }
    avatarMutation.mutate(file);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      // Empty string (not null) signals "clear this field": the backend treats a
      // blank phone as NULL, which is what lets the UI remove a number.
      await updateMutation.mutateAsync({ full_name: fullName, phone: phone, avatar_url: avatarUrl });
      toast.success("Profil berhasil diperbarui!");
    } catch (err) {
      toast.error(apiErrorMessage(err, "Gagal memperbarui profil."));
    }
  };

  const isSaving = updateMutation.isPending;

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div>
        <h2 className="text-xl font-black uppercase tracking-wide text-foreground">Profil Saya</h2>
        <p className="text-xs font-bold text-muted-foreground mt-1">Kelola informasi profil akun Anda.</p>
      </div>

      <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-6">
        {isLoading ? (
          <div className="space-y-4">
            {[1, 2, 3, 4].map((i) => (
              <div key={i} className="h-10 bg-muted border-2 border-foreground animate-pulse" />
            ))}
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-5">
            {/* Avatar */}
            <div className="flex items-center gap-4 mb-2">
              <input
                ref={fileInputRef}
                type="file"
                accept={ACCEPTED_AVATAR}
                className="hidden"
                onChange={(e) => {
                  submitAvatar(e.target.files?.[0]);
                  e.target.value = "";
                }}
              />
              <button
                type="button"
                aria-label="Unggah foto profil"
                disabled={avatarMutation.isPending}
                onClick={() => fileInputRef.current?.click()}
                onDragOver={(e) => {
                  e.preventDefault();
                  setIsDragging(true);
                }}
                onDragLeave={() => setIsDragging(false)}
                onDrop={(e) => {
                  e.preventDefault();
                  setIsDragging(false);
                  submitAvatar(e.dataTransfer.files?.[0]);
                }}
                className={`group relative w-20 h-20 shrink-0 border-2 border-foreground overflow-hidden flex items-center justify-center transition-all cursor-pointer ${
                  isDragging ? "bg-primary/20 ring-4 ring-primary/30" : "bg-primary/10"
                }`}
              >
                {avatarUrl ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img src={avatarUrl} alt="Foto profil" className="w-full h-full object-cover" />
                ) : (
                  <span className="text-xl font-black text-primary">{initials}</span>
                )}
                <span className="absolute inset-0 bg-foreground/60 text-background opacity-0 group-hover:opacity-100 flex items-center justify-center transition-opacity">
                  {avatarMutation.isPending ? (
                    <Loader2 className="w-5 h-5 animate-spin" />
                  ) : (
                    <Camera className="w-5 h-5" />
                  )}
                </span>
              </button>
              <div className="min-w-0">
                <p className="font-black text-lg text-foreground truncate">{fullName || "User"}</p>
                <p className="text-sm font-bold text-muted-foreground truncate">{email}</p>
                <p className="text-[11px] font-bold text-muted-foreground mt-1">
                  Klik atau tarik foto untuk mengganti (JPG/PNG/WEBP, maks 2 MB).
                </p>
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

            <Button type="submit" disabled={isSaving} className="w-full">
              {isSaving ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : <Save className="w-4 h-4 mr-2" />}
              {isSaving ? "Menyimpan..." : "Simpan Perubahan"}
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
