"use client";

import { useState } from "react";
import { Bell, Shield, Eye, EyeOff, Palette, Globe } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { useMutation } from "@tanstack/react-query";
import api from "@/lib/api";

export default function SettingsPage() {
  const [showPasswordForm, setShowPasswordForm] = useState(false);
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showPasswords, setShowPasswords] = useState(false);
  // Notification toggles are local UI state (no backend endpoint yet) — kept so
  // the switches are real, labelled, keyboard-operable controls.
  const [notificationPrefs, setNotificationPrefs] = useState([true, true, false]);
  const setNotificationPref = (index: number, value: boolean) =>
    setNotificationPrefs((prev) => prev.map((v, i) => (i === index ? value : v)));

  const updatePasswordMutation = useMutation({
    mutationFn: async (data: { current_password: string; new_password: string }) => {
      const res = await api.put("/auth/password", data);
      return res.data;
    },
  });

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    if (newPassword !== confirmPassword) {
      toast.error("Password baru dan konfirmasi tidak cocok.");
      return;
    }
    if (newPassword.length < 6) {
      toast.error("Password minimal 6 karakter.");
      return;
    }
    try {
      await updatePasswordMutation.mutateAsync({ current_password: currentPassword, new_password: newPassword });
      toast.success("Password berhasil diubah!");
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      setShowPasswordForm(false);
    } catch {
      toast.error("Gagal mengubah password. Pastikan password saat ini benar.");
    }
  };

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div>
        <h2 className="text-xl font-black uppercase tracking-wide text-foreground">Pengaturan</h2>
        <p className="text-xs font-bold text-muted-foreground mt-1">Kelola preferensi akun Anda.</p>
      </div>

      <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground divide-y-2 divide-foreground">
        <div className="p-5">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 border-2 border-foreground bg-primary/10 flex items-center justify-center">
              <Shield className="w-5 h-5 text-primary" />
            </div>
            <div>
              <h3 className="text-sm font-black uppercase tracking-wide text-foreground">Keamanan</h3>
              <p className="text-xs font-bold text-muted-foreground">Kelola password akun Anda</p>
            </div>
          </div>

          {!showPasswordForm ? (
            <Button variant="outline" onClick={() => setShowPasswordForm(true)}>
              <Eye className="w-4 h-4 mr-2" /> Ubah Password
            </Button>
          ) : (
            <form onSubmit={handleChangePassword} className="space-y-4">
              <div className="space-y-2">
                <label htmlFor="settings-current-password" className="text-xs font-bold uppercase tracking-wide text-foreground">Password Saat Ini</label>
                <div className="relative">
                  <input
                    type={showPasswords ? "text" : "password"}
                    id="settings-current-password"
                    value={currentPassword}
                    onChange={(e) => setCurrentPassword(e.target.value)}
                    placeholder="Masukkan password saat ini"
                    className="w-full border-2 border-foreground bg-background px-4 py-2.5 pr-10 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none placeholder:text-muted-foreground/70"
                    required
                  />
                </div>
              </div>
              <div className="space-y-2">
                <label htmlFor="settings-new-password" className="text-xs font-bold uppercase tracking-wide text-foreground">Password Baru</label>
                <input
                  type={showPasswords ? "text" : "password"}
                  id="settings-new-password"
                    value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  placeholder="Minimal 6 karakter"
                  className="w-full border-2 border-foreground bg-background px-4 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none placeholder:text-muted-foreground/70"
                  required
                  minLength={6}
                />
              </div>
              <div className="space-y-2">
                <label htmlFor="settings-confirm-password" className="text-xs font-bold uppercase tracking-wide text-foreground">Konfirmasi Password Baru</label>
                <input
                  type={showPasswords ? "text" : "password"}
                  id="settings-confirm-password"
                    value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  placeholder="Ulangi password baru"
                  className="w-full border-2 border-foreground bg-background px-4 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none placeholder:text-muted-foreground/70"
                  required
                />
              </div>
              <button
                type="button"
                onClick={() => setShowPasswords(!showPasswords)}
                className="text-xs font-bold text-primary hover:text-primary/80 flex items-center gap-1 cursor-pointer"
              >
                {showPasswords ? <EyeOff className="w-3 h-3" /> : <Eye className="w-3 h-3" />}
                {showPasswords ? "Sembunyikan" : "Tampilkan"} password
              </button>
              <div className="flex gap-3 pt-2">
                <Button type="button" variant="outline" onClick={() => setShowPasswordForm(false)}>Batal</Button>
                <Button type="submit" disabled={updatePasswordMutation.isPending}>
                  {updatePasswordMutation.isPending ? "Menyimpan..." : "Simpan Password"}
                </Button>
              </div>
            </form>
          )}
        </div>

        <div className="p-5">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 border-2 border-foreground bg-primary/10 flex items-center justify-center">
              <Bell className="w-5 h-5 text-primary" />
            </div>
            <div>
              <h3 className="text-sm font-black uppercase tracking-wide text-foreground">Notifikasi</h3>
              <p className="text-xs font-bold text-muted-foreground">Preferensi notifikasi</p>
            </div>
          </div>
          <div className="space-y-3">
            {[
              { label: "Email Notifikasi Transaksi", desc: "Dapatkan notifikasi email setiap ada transaksi baru" },
              { label: "Pengingat Kategori", desc: "Pengingat jika pengeluaran mendekati batas kategori" },
              { label: "Laporan Mingguan", desc: "Ringkasan keuangan mingguan via email" },
            ].map((item, i) => (
              <label key={i} className="flex items-center justify-between py-2 cursor-pointer">
                <div>
                  <p className="text-sm font-bold text-foreground">{item.label}</p>
                  <p className="text-xs font-bold text-muted-foreground">{item.desc}</p>
                </div>
                {/* A real checkbox: gives the label a control to associate with
                    and makes the toggle keyboard-operable (the visual switch is
                    a styled sibling, so the input is visually hidden). */}
                <input
                  type="checkbox"
                  className="sr-only"
                  aria-label={item.label}
                  checked={notificationPrefs[i]}
                  onChange={(e) => setNotificationPref(i, e.target.checked)}
                />
                <div aria-hidden="true" className="w-10 h-6 border-2 border-foreground bg-muted relative cursor-pointer">
                  <div className="absolute left-0.5 top-0.5 w-4 h-4 bg-muted-foreground" />
                </div>
              </label>
            ))}
          </div>
        </div>

        <div className="p-5">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 border-2 border-foreground bg-primary/10 flex items-center justify-center">
              <Palette className="w-5 h-5 text-primary" />
            </div>
            <div>
              <h3 className="text-sm font-black uppercase tracking-wide text-foreground">Tampilan</h3>
              <p className="text-xs font-bold text-muted-foreground">Preferensi tema</p>
            </div>
          </div>
          <div className="flex gap-3">
            {["Terang", "Gelap", "Sistem"].map((mode) => (
              <button
                key={mode}
                className={`flex-1 py-2.5 text-sm font-bold border-2 border-foreground cursor-pointer transition-all ${
                  mode === "Terang"
                    ? "bg-card shadow-[2px_2px_0px_0px] shadow-foreground"
                    : "bg-muted text-muted-foreground hover:text-foreground"
                }`}
              >
                {mode}
              </button>
            ))}
          </div>
        </div>

        <div className="p-5">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 border-2 border-foreground bg-primary/10 flex items-center justify-center">
              <Globe className="w-5 h-5 text-primary" />
            </div>
            <div>
              <h3 className="text-sm font-black uppercase tracking-wide text-foreground">Bahasa & Regional</h3>
              <p className="text-xs font-bold text-muted-foreground">Preferensi bahasa dan format regional</p>
            </div>
          </div>
          <select className="w-full border-2 border-foreground bg-background px-4 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none">
            <option>Indonesia (ID)</option>
            <option>English (US)</option>
          </select>
        </div>
      </div>
    </div>
  );
}
