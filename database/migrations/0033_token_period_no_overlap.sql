-- ============================================================
-- 0033_token_period_no_overlap.sql
--
-- Guarantee a user has AT MOST ONE token period covering any instant.
--
-- Problem addressed:
--   The token period is created per billing cycle and keyed by
--   (user_id, period_start). Nothing stopped a SECOND period from being inserted
--   whose [period_start, period_end] window overlapped the first — which
--   happened on every payment that reset current_period_start (10 overlapping
--   periods were found on one live account). The ledger's "current period"
--   lookup then silently picked only the newest row, orphaning the rest and
--   making the user's quota accounting ambiguous.
--
--   The application fix keeps the cycle stable across plan changes (the running
--   period's quota is synced in place instead of a new period being opened). This
--   migration is the database backstop: an exclusion constraint makes overlap
--   impossible even from a future code path or a second app instance.
--
-- Expand-only & idempotent.
-- ============================================================

BEGIN;

-- ------------------------------------------------------------
-- 1. btree_gist enables an exclusion constraint that mixes equality
--    (user_id) with a range overlap test (the period window).
-- ------------------------------------------------------------
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- ------------------------------------------------------------
-- 2. Repair existing overlaps so the constraint can be created.
--
--    The ledger is append-only (DELETE is blocked by trigger) and
--    token_periods is referenced by it with ON DELETE RESTRICT, so an orphaned
--    period cannot simply be deleted. Instead we SHRINK each superseded period
--    so it no longer covers `now` and no longer overlaps the period that
--    replaced it: its window is clamped to end where the successor begins.
--    Usage already recorded on it is preserved (the ledger keeps pointing at
--    it), and the live cycle is the newest period — unchanged.
--
--    A period is "superseded" when another period for the same user starts
--    strictly later but still within its window.
-- ------------------------------------------------------------
WITH ranked AS (
    SELECT id,
           user_id,
           period_start,
           period_end,
           LEAD(period_start) OVER (
               PARTITION BY user_id
               ORDER BY period_start ASC, id ASC
           ) AS next_start
    FROM token_periods
)
UPDATE token_periods p
SET period_end = r.next_start,
    updated_at = now()
FROM ranked r
WHERE p.id = r.id
  AND r.next_start IS NOT NULL
  AND r.next_start < p.period_end
  AND r.next_start > p.period_start;

-- ------------------------------------------------------------
-- 3. The backstop: no two periods for a user may overlap.
--    Uses a half-open interval [start, end) so a period ending exactly when the
--    next begins is allowed (back-to-back cycles are legitimate).
-- ------------------------------------------------------------
ALTER TABLE token_periods
    DROP CONSTRAINT IF EXISTS token_periods_no_overlap;

ALTER TABLE token_periods
    ADD CONSTRAINT token_periods_no_overlap
    EXCLUDE USING gist (
        user_id WITH =,
        tstzrange(period_start, period_end, '[)') WITH &&
    );

COMMIT;
