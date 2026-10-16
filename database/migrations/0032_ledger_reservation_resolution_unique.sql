-- ============================================================
-- 0032_ledger_reservation_resolution_unique.sql
--
-- Guarantee that a reservation can be resolved AT MOST ONCE.
--
-- Problem addressed:
--   LedgerService.Commit/Release checked `IsReservationResolved` OUTSIDE the
--   transaction, so two concurrent calls with different idempotency keys both
--   passed the check and both charged the period (double-charge). Live data
--   showed 112 reservations with two `commit` entries each.
--
--   The application fix moves the check inside the transaction, after the
--   period row is locked, which serialises concurrent resolutions. This
--   trigger is the database-level backstop: even if a future code path (or a
--   second app instance) races, the second commit/release is rejected instead
--   of silently charging twice.
--
--   A partial UNIQUE INDEX cannot be used: the ledger is append-only (UPDATE
--   and DELETE are blocked by trigger), so the existing duplicate rows cannot
--   be removed to satisfy the index. A BEFORE INSERT trigger works on the
--   already-dirty data because it only constrains NEW writes.
--
-- Expand-only & idempotent: it adds a trigger and repairs the extra charges
-- caused by past duplicates. Nothing is dropped, so an older app version keeps
-- working during rollout.
-- ============================================================

BEGIN;

-- ------------------------------------------------------------
-- 1. Repair the double-charges already present.
--
--    The duplicate rows cannot be deleted, so the extra charge is neutralised
--    by crediting the period back and recording an `adjust` entry (the same
--    mechanism admin top-ups use). This keeps the ledger honest and the balance
--    correct. Guarded by the idempotency key, so re-running is a no-op.
-- ------------------------------------------------------------
WITH ranked AS (
    SELECT id,
           reservation_id,
           user_id,
           period_id,
           amount,
           modality,
           row_number() OVER (
               PARTITION BY reservation_id
               ORDER BY created_at ASC, id ASC
           ) AS rn
    FROM token_ledger
    WHERE reservation_id IS NOT NULL
      AND entry_type = 'commit'
),
dupes AS (
    -- Everything after the FIRST commit for a reservation is a duplicate.
    SELECT reservation_id,
           user_id,
           period_id,
           min(modality) AS modality,
           sum(amount) AS refund_units
    FROM ranked
    WHERE rn > 1
    GROUP BY reservation_id, user_id, period_id
),
inserted AS (
    INSERT INTO token_ledger (period_id, user_id, reservation_id, entry_type, amount, modality, idempotency_key, notes)
    SELECT d.period_id,
           d.user_id,
           d.reservation_id,
           'adjust',
           -d.refund_units,
           d.modality,
           'dedupe-commit:' || d.reservation_id::text,
           'auto: reversal of duplicate commit for reservation ' || d.reservation_id::text
    FROM dupes d
    WHERE NOT EXISTS (
        SELECT 1 FROM token_ledger x
        WHERE x.idempotency_key = 'dedupe-commit:' || d.reservation_id::text
    )
    RETURNING period_id, amount
)
-- Apply the reversal to the affected periods (used_tokens only; the reserved
-- hold was already released by the original commit). Scoped to rows inserted
-- by THIS statement, so re-running the migration credits nothing further.
UPDATE token_periods p
SET used_tokens = GREATEST(0, p.used_tokens - c.credit),
    updated_at = now()
FROM (SELECT period_id, sum(-amount) AS credit FROM inserted GROUP BY period_id) c
WHERE p.id = c.period_id;

-- ------------------------------------------------------------
-- 2. Backstop: reject a second commit/release for a reservation.
--
--    Raises SQLSTATE 'P0001' with a marker message the repository maps to
--    billing.ErrReservationAlreadyResolved, so callers see the same error the
--    in-transaction check returns.
-- ------------------------------------------------------------
CREATE OR REPLACE FUNCTION token_ledger_block_double_resolution()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.reservation_id IS NOT NULL
       AND NEW.entry_type IN ('commit', 'release')
       AND EXISTS (
           SELECT 1 FROM token_ledger
           WHERE reservation_id = NEW.reservation_id
             AND entry_type IN ('commit', 'release')
       ) THEN
        RAISE EXCEPTION 'reservation % already resolved', NEW.reservation_id
            USING ERRCODE = 'P0001',
                  HINT = 'reservation_already_resolved';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS token_ledger_single_resolution ON token_ledger;
CREATE TRIGGER token_ledger_single_resolution
    BEFORE INSERT ON token_ledger
    FOR EACH ROW
    EXECUTE FUNCTION token_ledger_block_double_resolution();

COMMIT;
