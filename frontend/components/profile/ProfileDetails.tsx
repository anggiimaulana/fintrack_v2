"use client";

import { useRef, useState } from "react";
import { User, Mail, Phone, Save, Camera, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
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

/**
 * Account details: avatar, name, phone, and the (read-only) email.
 *
 * Extracted from the former standalone profile page so it can live inside the
 * merged "Profil & Pengaturan" accordion.
 */
export function ProfileDetails() {
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
  const [isDragging, setIsDragging] = useState(false);

  const fullName = formOverride.fullName ?? profile?.full_name ?? "";
  const phone = formOverride.phone ?? profile?.phone ?? "";
  const avatarUrl: string = profile?.avatar_url ?? "";
  const email = sessionUser?.email ?? profile?.email ?? "";

  const initials = (fullName || email || "US")
    .split(" ")
    .map((w: string) => w[0])
    .filter(Boolean)
    .slice(0, 2)
    .join("")
    .toUpperCase();

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
      await updateMutation.mutateAsync({ full_name: fullName, phone, avatar_url: avatarUrl });
      toast.success("Profil berhasil diperbarui!");
    } catch (err) {
      toast.error(apiErrorMessage(err, "Gagal memperbarui profil."));
    }
  };

  const isSaving = updateMutation.isPending;

  if (isLoading) {
    return (
      <div className="space-y-4">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="h-10 bg-muted border-2 border-foreground animate-pulse" />
        ))}
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      <div className="flex items-center gap-4">
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

      <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
        <div className="space-y-2">
          <label htmlFor="profile-full-name" className="text-xs font-bold uppercase tracking-wide text-foreground flex items-center gap-2">
            <User className="w-4 h-4" /> Nama Lengkap
          </label>
          <input
            id="profile-full-name"
            type="text"
            value={fullName}
            onChange={(e) => setFormOverride((s) => ({ ...s, fullName: e.target.value }))}
            placeholder="Nama lengkap Anda"
            className="w-full border-2 border-foreground bg-background px-4 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none placeholder:text-muted-foreground/70"
          />
        </div>

        <div className="space-y-2">
          <label htmlFor="profile-phone" className="text-xs font-bold uppercase tracking-wide text-foreground flex items-center gap-2">
            <Phone className="w-4 h-4" /> Nomor Telepon
          </label>
          <input
            id="profile-phone"
            type="tel"
            value={phone}
            onChange={(e) => setFormOverride((s) => ({ ...s, phone: e.target.value }))}
            placeholder="08xxxxxxxxxx"
            className="w-full border-2 border-foreground bg-background px-4 py-2.5 text-sm font-bold focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground focus:outline-none placeholder:text-muted-foreground/70"
          />
        </div>
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

      <div className="flex justify-end pt-1">
        <Button type="submit" disabled={isSaving} className="w-full sm:w-auto sm:min-w-52">
          {isSaving ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : <Save className="w-4 h-4 mr-2" />}
          {isSaving ? "Menyimpan..." : "Simpan Perubahan"}
        </Button>
      </div>
    </form>
  );
}
