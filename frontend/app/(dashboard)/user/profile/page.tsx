"use client";

import { useState, useEffect } from "react";
import { User, Mail, Phone, Calendar, Save } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { RupiahInput } from "@/components/ui/rupiah-input";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { useSession } from "@/lib/session-context";

export default function ProfilePage() {
  const queryClient = useQueryClient();
  const { user: sessionUser } = useSession();
  const [fullName, setFullName] = useState("");
  const [phone, setPhone] = useState("");
  const [avatarUrl, setAvatarUrl] = useState("");

  const { data: profile, isLoading } = useQuery({
    queryKey: ["profile", "me"],
    queryFn: async () => {
      try {
        const res = await api.get("/me");
        return res.data?.data;
      } catch {
        return null;
      }
    },
  });

  useEffect(() => {
    if (profile) {
      setFullName(profile.full_name || "");
      setPhone(profile.phone || "");
      setAvatarUrl(profile.avatar_url || "");
    }
  }, [profile]);

  const email = sessionUser?.email ?? profile?.email ?? "";

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
              <label className="text-xs font-bold uppercase tracking-wide text-foreground flex items-center gap-2">
                <User className="w-4 h-4" /> Nama Lengkap
              </label>
              <input
                type="text"
                value={fullName}
                onChange={(e) => setFullName(e.target.value)}
                placeholder="Nama lengkap Anda"
                className="w-full border-2 border-foreground bg-background px-4 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none placeholder:text-muted-foreground/70"
              />
            </div>

            <div className="space-y-2">
              <label className="text-xs font-bold uppercase tracking-wide text-foreground flex items-center gap-2">
                <Mail className="w-4 h-4" /> Email
              </label>
              <input
                type="email"
                value={email}
                disabled
                className="w-full border-2 border-foreground bg-muted px-4 py-2.5 text-sm font-bold text-muted-foreground cursor-not-allowed"
              />
              <p className="text-xs font-bold text-muted-foreground">Email tidak dapat diubah. Hubungi admin jika perlu perubahan.</p>
            </div>

            <div className="space-y-2">
              <label className="text-xs font-bold uppercase tracking-wide text-foreground flex items-center gap-2">
                <Phone className="w-4 h-4" /> Nomor Telepon
              </label>
              <input
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
    </div>
  );
}
