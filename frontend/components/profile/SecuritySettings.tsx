"use client";

import { useState } from "react";
import { Eye, EyeOff } from "lucide-react";
import { toast } from "sonner";
import axios from "axios";
import { Button } from "@/components/ui/button";
import { useMutation } from "@tanstack/react-query";
import api from "@/lib/api";

/**
 * Change-password form. The backend (`PUT /auth/password`) verifies the current
 * password, rejects reuse, and enforces a minimum of 8 characters — the same
 * rule as registration. Server error codes are surfaced verbatim so the user
 * knows exactly what to fix.
 */
export function SecuritySettings() {
  const [showForm, setShowForm] = useState(false);
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showPasswords, setShowPasswords] = useState(false);

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
    if (newPassword.length < 8) {
      toast.error("Password baru minimal 8 karakter.");
      return;
    }
    if (newPassword === currentPassword) {
      toast.error("Password baru tidak boleh sama dengan password saat ini.");
      return;
    }
    try {
      await updatePasswordMutation.mutateAsync({ current_password: currentPassword, new_password: newPassword });
      toast.success("Password berhasil diubah. Sesi di perangkat lain telah dikeluarkan.");
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      setShowForm(false);
    } catch (err) {
      // Map the backend's machine-readable codes to a precise message rather
      // than a generic failure (mirrors how the login form handles errors).
      const code = axios.isAxiosError(err) ? err.response?.data?.error : undefined;
      const byCode: Record<string, string> = {
        wrong_password: "Password saat ini salah.",
        password_same: "Password baru tidak boleh sama dengan password saat ini.",
        weak_password: "Password baru minimal 8 karakter.",
      };
      toast.error((code && byCode[code]) || "Gagal mengubah password. Coba lagi.");
    }
  };

  if (!showForm) {
    return (
      <Button variant="outline" onClick={() => setShowForm(true)}>
        <Eye className="w-4 h-4 mr-2" /> Ubah Password
      </Button>
    );
  }

  return (
    <form onSubmit={handleChangePassword} className="space-y-4">
      <div className="space-y-2">
        <label htmlFor="settings-current-password" className="text-xs font-bold uppercase tracking-wide text-foreground">Password Saat Ini</label>
        <input
          type={showPasswords ? "text" : "password"}
          id="settings-current-password"
          value={currentPassword}
          onChange={(e) => setCurrentPassword(e.target.value)}
          placeholder="Masukkan password saat ini"
          className="w-full border-2 border-foreground bg-background px-4 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none placeholder:text-muted-foreground/70"
          required
        />
      </div>
      <div className="space-y-2">
        <label htmlFor="settings-new-password" className="text-xs font-bold uppercase tracking-wide text-foreground">Password Baru</label>
        <input
          type={showPasswords ? "text" : "password"}
          id="settings-new-password"
          value={newPassword}
          onChange={(e) => setNewPassword(e.target.value)}
          placeholder="Minimal 8 karakter"
          className="w-full border-2 border-foreground bg-background px-4 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none placeholder:text-muted-foreground/70"
          required
          minLength={8}
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
        <Button type="button" variant="outline" onClick={() => setShowForm(false)}>Batal</Button>
        <Button type="submit" disabled={updatePasswordMutation.isPending}>
          {updatePasswordMutation.isPending ? "Menyimpan..." : "Simpan Password"}
        </Button>
      </div>
    </form>
  );
}
