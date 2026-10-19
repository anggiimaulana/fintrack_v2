-- Dunning now follows the subscription period instead of the invoice due date,
-- so the default "soon" reminder is H-7 (168h) rather than H-3 (72h).
UPDATE settings
SET value = to_jsonb(168),
    description = 'Ingatkan (segera) saat langganan berakhir <= N jam'
WHERE key = 'dunning_hours_soon' AND value = to_jsonb(72);

UPDATE settings
SET description = 'Ingatkan (mendesak) saat langganan berakhir <= N jam'
WHERE key = 'dunning_hours_urgent';

-- Persistent dedupe for notifications: restarts and multi-replica deployments
-- must not re-send the same reminder. NULL keys are unaffected (all existing
-- rows), so only callers that opt in (dunning) are deduplicated.
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS dedupe_key text;

CREATE UNIQUE INDEX IF NOT EXISTS uq_notifications_user_dedupe
    ON notifications (user_id, dedupe_key)
    WHERE dedupe_key IS NOT NULL;

-- Backfill one "created" event for subscriptions that predate the activator
-- recording history. Without this the "Riwayat Paket" tab stays empty until the
-- next renewal even though the user has an active subscription.
INSERT INTO subscription_events (subscription_id, user_id, event_type, previous_state, new_state, created_at)
SELECT s.id,
       s.user_id,
       'created',
       '{}'::jsonb,
       jsonb_build_object('plan_code', p.code, 'status', s.status),
       COALESCE(s.current_period_start, s.created_at)
FROM subscriptions s
JOIN plans p ON p.id = s.plan_id
WHERE NOT EXISTS (
    SELECT 1 FROM subscription_events e WHERE e.subscription_id = s.id
);
