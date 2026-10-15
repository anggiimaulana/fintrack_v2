import { useQuery, useMutation, useQueryClient, type QueryClient } from "@tanstack/react-query";
import axios from "axios";
import api from "@/lib/api";
import { queryKeys } from "@/lib/cache-keys";
import { staleTimeConfig } from "@/lib/cache-invalidation";
import { subscribeDomainEvents } from "@/lib/realtime-bus";
import { toast } from "sonner";
import { useEffect } from "react";

/**
 * A notification row as returned by the API.
 */
export interface NotificationItem {
  id: string;
  event: string;
  title: string;
  body: string;
  href: string;
  read_at?: string | null;
  created_at: string;
}

export interface NotificationStats {
  total: number;
  unread: number;
  read: number;
}

/** The shape every list cache entry holds: `{ data, total, page, perPage }`. */
export interface NotificationListPage {
  data: NotificationItem[];
  total: number;
  page: number;
  perPage: number;
}

export interface NotificationListFilter {
  page?: number;
  perPage?: number;
  sort?: string;
  order?: "asc" | "desc";
  search?: string;
  status?: "all" | "read" | "unread";
}

/** Stable cache key for a filter set (serialised so it is a plain string). */
export function notificationsListKey(filter: NotificationListFilter): string {
  return JSON.stringify({
    page: filter.page ?? 1,
    perPage: filter.perPage ?? 20,
    sort: filter.sort ?? "created_at",
    order: filter.order ?? "desc",
    search: filter.search ?? "",
    status: filter.status ?? "all",
  });
}

function buildListParams(filter: NotificationListFilter): string {
  const p = new URLSearchParams();
  p.set("page", String(filter.page ?? 1));
  p.set("per_page", String(filter.perPage ?? 20));
  if (filter.sort) p.set("sort", filter.sort);
  if (filter.order) p.set("order", filter.order);
  if (filter.search) p.set("q", filter.search);
  if (filter.status && filter.status !== "all") p.set("filter_status", filter.status);
  return p.toString();
}

// ─── cache patching (the realtime requirement) ────────────────────
//
// Every cached list page is patched in place rather than refetched. This is what
// keeps a realtime "created/read/deleted" event O(1) instead of re-downloading
// the whole table.

type ListSnapshot = [readonly unknown[], NotificationListPage][];

function snapshotLists(queryClient: QueryClient): ListSnapshot {
  return queryClient.getQueriesData<NotificationListPage>({
    queryKey: queryKeys.notifications.all,
  }) as ListSnapshot;
}

function restoreLists(queryClient: QueryClient, snapshot: ListSnapshot) {
  for (const [key, value] of snapshot) queryClient.setQueryData(key, value);
}

/** Mark the given ids read in every cached page (in place). */
function patchRead(queryClient: QueryClient, ids: string[], readAt: string) {
  const idSet = new Set(ids);
  for (const [key, value] of snapshotLists(queryClient)) {
    if (!value?.data) continue;
    queryClient.setQueryData(key, {
      ...value,
      data: value.data.map((n) => (idSet.has(n.id) ? { ...n, read_at: n.read_at ?? readAt } : n)),
    });
  }
}

/** Remove the given ids from every cached page (in place). */
function patchDeleted(queryClient: QueryClient, ids: string[]) {
  const idSet = new Set(ids);
  for (const [key, value] of snapshotLists(queryClient)) {
    if (!value?.data) continue;
    const next = value.data.filter((n) => !idSet.has(n.id));
    queryClient.setQueryData(key, {
      ...value,
      data: next,
      total: Math.max(0, (value.total ?? next.length) - (value.data.length - next.length)),
    });
  }
}

/** Prepend a new notification to page 1 of the default (newest-first) list. */
function patchCreated(queryClient: QueryClient, item: NotificationItem) {
  const defaultKey = notificationsListKey({ page: 1, perPage: 20, sort: "created_at", order: "desc" });
  for (const [key, value] of snapshotLists(queryClient)) {
    if (!value?.data) continue;
    // The cache key is ["notifications","list", <json>]; index 2 is the already
    // serialised filter string, so compare it directly (never re-stringify it).
    const isDefaultFirstPage = key[2] === defaultKey;
    if (!isDefaultFirstPage) continue;
    if (value.data.some((n) => n.id === item.id)) continue;
    queryClient.setQueryData(key, {
      ...value,
      data: [item, ...value.data].slice(0, value.perPage),
      total: (value.total ?? value.data.length) + 1,
    });
  }
}

/** Bump the unread/total counters without a refetch. */
function patchStats(
  queryClient: QueryClient,
  mutate: (s: NotificationStats) => NotificationStats,
) {
  queryClient.setQueryData<NotificationStats>(queryKeys.notifications.stats, (prev) =>
    prev ? mutate(prev) : prev,
  );
}

// ─── queries ──────────────────────────────────────────────────────

export function useNotifications(filter: NotificationListFilter = {}) {
  const key = notificationsListKey(filter);
  return useQuery<NotificationListPage>({
    queryKey: queryKeys.notifications.list(key),
    queryFn: async () => {
      const res = await api.get(`/me/notifications?${buildListParams(filter)}`);
      return {
        data: (res.data?.data ?? []) as NotificationItem[],
        total: res.data?.meta?.total ?? 0,
        page: res.data?.meta?.page ?? 1,
        perPage: res.data?.meta?.per_page ?? 20,
      };
    },
    staleTime: staleTimeConfig.profile,
    placeholderData: (prev) => prev,
  });
}

export function useNotificationStats() {
  return useQuery<NotificationStats>({
    queryKey: queryKeys.notifications.stats,
    queryFn: async () => {
      const res = await api.get("/me/notifications/stats");
      return (res.data?.data ?? { total: 0, unread: 0, read: 0 }) as NotificationStats;
    },
    staleTime: staleTimeConfig.profile,
  });
}

/** How many rows the header bell dropdown renders. */
export const BELL_PREVIEW_LIMIT = 8;

/**
 * Per-user notification preferences, persisted server-side.
 *
 * The toggles control EMAIL delivery: the backend writes every event to the
 * in-app inbox, and emails only when the matching preference is on.
 */
export interface NotificationPreferences {
  transaction_email: boolean;
  category_reminder: boolean;
  weekly_report: boolean;
}

export function useNotificationPreferences() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: queryKeys.notifications.preferences,
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
      queryClient.setQueryData(queryKeys.notifications.preferences, saved);
      toast.success("Preferensi notifikasi disimpan.");
    },
    onError: () => toast.error("Gagal menyimpan preferensi notifikasi."),
  });

  return { preferences: query.data, isLoading: query.isLoading, isSaving: mutation.isPending, save: mutation.mutate };
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

/**
 * useNotificationInbox backs the header bell.
 *
 * It reads the SAME cache entries as the notifications page (the default
 * newest-first first page and the stats query) so a single SSE event updates
 * both, and it never polls: freshness comes from `useNotificationsRealtime`,
 * which is mounted once at the provider level. The list is sliced for display,
 * so no extra request is issued just to render the dropdown.
 */
export function useNotificationInbox() {
  const filter: NotificationListFilter = { page: 1, perPage: 20, sort: "created_at", order: "desc" };
  const { data, isLoading } = useNotifications(filter);
  const { data: stats } = useNotificationStats();

  const markReadMutation = useMarkNotificationRead();
  const markAllReadMutation = useMarkAllNotificationsRead();
  const clearMutation = useClearNotifications();

  return {
    items: (data?.data ?? []).slice(0, BELL_PREVIEW_LIMIT),
    unread: stats?.unread ?? 0,
    isLoading,
    markRead: (id: string) => markReadMutation.mutate(id),
    markAllRead: () => markAllReadMutation.mutate(),
    clear: () => clearMutation.mutate(),
  };
}

// ─── realtime subscription ────────────────────────────────────────
//
// Mounted once by the notifications page. Patches the cache in place for the
// affected rows only; never triggers a blanket refetch.

export function useNotificationsRealtime() {
  const queryClient = useQueryClient();
  useEffect(() => {
    return subscribeDomainEvents((event) => {
      const payload = (event.payload ?? {}) as {
        ids?: string[];
        item?: NotificationItem;
        unread?: number;
      };
      const ids = payload.ids ?? [];

      switch (event.event) {
        case "notification_created":
          if (payload.item) patchCreated(queryClient, payload.item);
          patchStats(queryClient, (s) => ({
            total: s.total + 1,
            unread: typeof payload.unread === "number" ? payload.unread : s.unread + 1,
            read: s.read,
          }));
          break;

        case "notification_read": {
          const now = new Date().toISOString();
          if (ids.length > 0) {
            patchRead(queryClient, ids, now);
            patchStats(queryClient, (s) => ({
              total: s.total,
              unread: typeof payload.unread === "number" ? payload.unread : Math.max(0, s.unread - ids.length),
              read: s.read + ids.length,
            }));
          } else {
            // "all" was marked read: unread collapses to 0.
            for (const [key, value] of snapshotLists(queryClient)) {
              if (!value?.data) continue;
              queryClient.setQueryData(key, {
                ...value,
                data: value.data.map((n) => ({ ...n, read_at: n.read_at ?? now })),
              });
            }
            patchStats(queryClient, (s) => ({ total: s.total, unread: 0, read: s.total }));
          }
          break;
        }

        case "notification_deleted": {
          if (ids.length > 0) {
            patchDeleted(queryClient, ids);
            patchStats(queryClient, (s) => ({
              total: Math.max(0, s.total - ids.length),
              unread: typeof payload.unread === "number" ? payload.unread : s.unread,
              read: Math.max(0, s.read - ids.length),
            }));
          } else {
            // Everything was cleared.
            for (const [key, value] of snapshotLists(queryClient)) {
              queryClient.setQueryData(key, { ...value, data: [], total: 0 });
            }
            patchStats(queryClient, () => ({ total: 0, unread: 0, read: 0 }));
          }
          break;
        }
        default:
          break;
      }
    });
  }, [queryClient]);
}

// ─── mutations (optimistic, so the UI never waits) ────────────────

export function useMarkNotificationRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.post(`/me/notifications/${id}/read`);
    },
    onMutate: async (id: string) => {
      await queryClient.cancelQueries({ queryKey: queryKeys.notifications.all });
      const snapshot = snapshotLists(queryClient);
      patchRead(queryClient, [id], new Date().toISOString());
      patchStats(queryClient, (s) => ({ total: s.total, unread: Math.max(0, s.unread - 1), read: s.read + 1 }));
      return { snapshot };
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.snapshot) restoreLists(queryClient, ctx.snapshot);
      toast.error("Gagal menandai notifikasi.");
    },
  });
}

export function useMarkNotificationsRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (ids: string[]) => {
      await api.post("/me/notifications/read", { ids });
    },
    onMutate: async (ids: string[]) => {
      await queryClient.cancelQueries({ queryKey: queryKeys.notifications.all });
      const snapshot = snapshotLists(queryClient);
      const now = new Date().toISOString();
      patchRead(queryClient, ids, now);
      patchStats(queryClient, (s) => ({
        total: s.total,
        unread: Math.max(0, s.unread - ids.length),
        read: s.read + ids.length,
      }));
      return { snapshot };
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.snapshot) restoreLists(queryClient, ctx.snapshot);
      toast.error("Gagal menandai notifikasi.");
    },
  });
}

export function useMarkAllNotificationsRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.post("/me/notifications/read-all");
    },
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: queryKeys.notifications.all });
      const snapshot = snapshotLists(queryClient);
      const now = new Date().toISOString();
      for (const [key, value] of snapshotLists(queryClient)) {
        if (!value?.data) continue;
        queryClient.setQueryData(key, {
          ...value,
          data: value.data.map((n) => ({ ...n, read_at: n.read_at ?? now })),
        });
      }
      patchStats(queryClient, (s) => ({ total: s.total, unread: 0, read: s.total }));
      return { snapshot };
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.snapshot) restoreLists(queryClient, ctx.snapshot);
      toast.error("Gagal menandai semua notifikasi.");
    },
  });
}

export function useDeleteNotification() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/me/notifications/${id}`);
    },
    onMutate: async (id: string) => {
      await queryClient.cancelQueries({ queryKey: queryKeys.notifications.all });
      const snapshot = snapshotLists(queryClient);
      patchDeleted(queryClient, [id]);
      patchStats(queryClient, (s) => ({ total: Math.max(0, s.total - 1), unread: s.unread, read: Math.max(0, s.read - 1) }));
      return { snapshot };
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.snapshot) restoreLists(queryClient, ctx.snapshot);
      toast.error("Gagal menghapus notifikasi.");
    },
  });
}

export function useDeleteNotifications() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (ids: string[]) => {
      await api.post("/me/notifications/delete", { ids });
    },
    onMutate: async (ids: string[]) => {
      await queryClient.cancelQueries({ queryKey: queryKeys.notifications.all });
      const snapshot = snapshotLists(queryClient);
      patchDeleted(queryClient, ids);
      patchStats(queryClient, (s) => ({
        total: Math.max(0, s.total - ids.length),
        unread: s.unread,
        read: Math.max(0, s.read - ids.length),
      }));
      return { snapshot };
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.snapshot) restoreLists(queryClient, ctx.snapshot);
      toast.error("Gagal menghapus notifikasi.");
    },
  });
}

/**
 * useClearNotifications empties the whole inbox (DELETE /me/notifications). Used
 * by the bell's "clear" action. Patches every cached page to empty and zeroes
 * the counters optimistically, restoring the snapshot on failure.
 */
export function useClearNotifications() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.delete("/me/notifications");
    },
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: queryKeys.notifications.all });
      const snapshot = snapshotLists(queryClient);
      for (const [key, value] of snapshotLists(queryClient)) {
        queryClient.setQueryData(key, { ...value, data: [], total: 0 });
      }
      patchStats(queryClient, () => ({ total: 0, unread: 0, read: 0 }));
      return { snapshot };
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.snapshot) restoreLists(queryClient, ctx.snapshot);
      toast.error("Gagal membersihkan notifikasi.");
    },
  });
}
