"use client";

import { useMutation, useQuery } from "@tanstack/react-query";
import axios from "axios";
import { toast } from "sonner";
import api from "@/lib/api";

/** Effective mail configuration as reported by the backend (password redacted). */
export interface MailStatus {
  enabled: boolean;
  configured: boolean;
  host?: string;
  port?: number;
  username?: string;
  from?: string;
  from_name?: string;
  use_tls?: boolean;
  has_password?: boolean;
}

export const mailStatusKey = ["admin", "mail-status"] as const;

export function useMailStatus() {
  return useQuery({
    queryKey: mailStatusKey,
    queryFn: async (): Promise<MailStatus> => {
      const res = await api.get("/admin/settings/mail");
      return res.data?.data as MailStatus;
    },
    staleTime: 30_000,
  });
}

/** Sends a test email using the effective config; `to` defaults to the From address. */
export function useSendTestMail() {
  return useMutation({
    mutationFn: async (to?: string) => {
      const res = await api.post<{ message?: string }>("/admin/settings/mail/test", to ? { to } : {});
      return res.data;
    },
    onSuccess: (res) => {
      toast.success(res?.message ?? "Email uji terkirim.");
    },
    onError: (err: unknown) => {
      const code = axios.isAxiosError(err) ? err.response?.data?.error : undefined;
      toast.error(
        code === "mail_disabled"
          ? "Email belum diaktifkan atau konfigurasi belum lengkap."
          : "Gagal mengirim email uji. Periksa host/port/kredensial.",
      );
    },
  });
}
