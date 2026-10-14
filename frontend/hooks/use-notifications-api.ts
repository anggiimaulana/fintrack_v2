"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import axios from "axios";
import { toast } from "sonner";
import api from "@/lib/api";
import { staleTimeConfig } from "@/lib/cache-invalidation";

/**
 * Per-user notification preferences, persisted server-side.
 *
 * The toggles here actually control delivery: the backend writes every event to
 * the in-app inbox, and emails only when the matching preference is on. This
 * replaced a localStorage-only version that no backend ever read.
 */
export interface NotificationPreferences {
  transaction_email: boolean;
  category_reminder: boolean;
  weekly_report: boolean;
}

export interface AppNotification {
  id: string;
  event: string;
  title: string;
  body: string;
  href: string;
  read_at?: string | null;
  created_at: string;
}

export const notificationKeys = {
  preferences: ["me", "notification-preferences"] as const,
  inbox: ["me", "notifications"] as const,
};

export function useNotificationPreferences() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: notificationKeys.preferences,
    queryFn: async (): Promise<NotificationPreferences> => {
      const res = await api.get("/me/notification-preferences");
      return res.data?.data as NotificationPreferences;
    },
    staleTime: staleTimeConfig.profile,
  });

  const mutation = useMutation({
    mutationFn: async (prefs: NotificationPreferences) => {
      const res = await api.put("/me/notification-preferences", prefs);
      return res.data?.data as NotificationPreferences;
    },
    onSuccess: (saved) => {
      // Seed the cache with the server's canonical value so the switches do not
      // flicker back on the next read.
      queryClient.setQueryData(notificationKeys.preferences, saved);
      toast.success("Preferensi notifikasi disimpan.");
    },
    onError: () => toast.error("Gagal menyimpan preferensi notifikasi."),
  });

  return { preferences: query.data, isLoading: query.isLoading, isSaving: mutation.isPending, save: mutation.mutate };
}

/**
 * The durable in-app inbox backing the header bell. Polls lightly so an event
 * that arrived while the tab was backgrounded shows up without a manual reload.
 */
export function useNotificationInbox() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: notificationKeys.inbox,
    queryFn: async (): Promise<{ items: AppNotification[]; unread: number }> => {
      const res = await api.get("/me/notifications");
      return {
        items: (res.data?.data?.items ?? []) as AppNotification[],
        unread: (res.data?.data?.unread ?? 0) as number,
      };
    },
    staleTime: 30_000,
    refetchInterval: 60_000,
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: notificationKeys.inbox });

  const markRead = useMutation({
    mutationFn: async (id: string) => {
      await api.post(`/me/notifications/${id}/read`);
    },
    onSuccess: invalidate,
  });

  const markAllRead = useMutation({
    mutationFn: async () => {
      await api.post("/me/notifications/read-all");
    },
    onSuccess: invalidate,
  });

  const clear = useMutation({
    mutationFn: async () => {
      await api.delete("/me/notifications");
    },
    onSuccess: invalidate,
  });

  return {
    items: query.data?.items ?? [],
    unread: query.data?.unread ?? 0,
    isLoading: query.isLoading,
    markRead: markRead.mutate,
    markAllRead: markAllRead.mutate,
    clear: clear.mutate,
  };
}

/** Sends a test email to the signed-in user (verifies delivery end-to-end). */
export function useSendTestEmail() {
  return useMutation({
    mutationFn: async () => {
      const res = await api.post("/me/notifications/test-email");
      return res.data;
    },
    onSuccess: () => toast.success("Email uji terkirim. Periksa kotak masuk Anda."),
    onError: (err: unknown) => {
      const code = axios.isAxiosError(err) ? err.response?.data?.error : undefined;
      toast.error(
        code === "mail_disabled"
          ? "Email belum dikonfigurasi. Hubungi admin."
          : "Gagal mengirim email uji. Periksa konfigurasi SMTP.",
      );
    },
  });
}
