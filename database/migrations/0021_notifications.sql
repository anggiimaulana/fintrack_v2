-- ============================================================
-- 0020_notifications.sql
-- Per-user notification preferences + a durable in-app inbox.
--
-- Replaces the localStorage-only toggles (which no backend read) with real,
-- server-side preferences, and persists notifications so the bell survives a
-- reload. Email delivery honours these preferences; the in-app entry is always
-- written so nothing is silently lost when email is unconfigured.
-- ============================================================

-- ------------------------------------------------------------
-- notification_preferences: one row per user, created lazily.
-- Defaults mirror the previous UI defaults (transaction email + category
-- reminder on, weekly report off).
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id            uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    transaction_email  boolean NOT NULL DEFAULT true,
    category_reminder  boolean NOT NULL DEFAULT true,
    weekly_report      boolean NOT NULL DEFAULT false,
    updated_at         timestamptz NOT NULL DEFAULT now()
);

-- ------------------------------------------------------------
-- notifications: durable in-app inbox, read state included.
-- The composite (user_id, created_at DESC) index serves the inbox query.
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notifications (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event      varchar(64)  NOT NULL,
    title      text         NOT NULL,
    body       text         NOT NULL,
    href       text         NOT NULL DEFAULT '',
    read_at    timestamptz,
    created_at timestamptz  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_created
    ON notifications (user_id, created_at DESC);

-- Unread count is a hot path (bell badge); a partial index keeps it cheap.
CREATE INDEX IF NOT EXISTS idx_notifications_user_unread
    ON notifications (user_id) WHERE read_at IS NULL;
