"use client";

import { useCallback, useEffect, useState } from "react";
import { subscribeDomainEvents } from "@/lib/realtime-bus";
import {
  appendNotification,
  notificationFor,
  unreadCount,
  type AppNotification,
} from "@/lib/notifications";

/**
 * In-memory notification feed for the header bell.
 *
 * Notifications are derived from the single realtime SSE stream (via the bus)
 * and kept in component state — they are a UX accelerator, not durable storage,
 * so they intentionally reset on reload. A durable inbox would be a backend
 * concern (out of scope here).
 */
export function useNotifications() {
  const [items, setItems] = useState<AppNotification[]>([]);

  useEffect(() => {
    return subscribeDomainEvents((event) => {
      const n = notificationFor(event);
      if (!n) return;
      setItems((prev) => appendNotification(prev, n));
    });
  }, []);

  const markAllRead = useCallback(() => {
    setItems((prev) => prev.map((n) => (n.read ? n : { ...n, read: true })));
  }, []);

  const markRead = useCallback((id: string) => {
    setItems((prev) => prev.map((n) => (n.id === id ? { ...n, read: true } : n)));
  }, []);

  const clear = useCallback(() => setItems([]), []);

  return { items, unread: unreadCount(items), markAllRead, markRead, clear };
}
