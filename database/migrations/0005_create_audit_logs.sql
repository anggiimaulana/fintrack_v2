-- ============================================================
-- 0005_create_audit_logs.sql
-- Append-only audit trail for administrative & billing actions.
-- Aligned with UU PDP No. 27/2022 (non-repudiation / accountability).
--
-- Guarantees:
--   * Immutable: a BEFORE UPDATE/DELETE trigger rejects any mutation.
--   * Append-only: only INSERT is permitted.
--
-- Idempotent: safe to re-run.
-- ============================================================

CREATE TABLE IF NOT EXISTS audit_logs (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Actor performing the action. NULL for system/background actions.
    actor_id    uuid REFERENCES users(id) ON DELETE SET NULL,
    -- Verb, e.g. 'create', 'update', 'delete', 'approve', 'adjust'.
    action      text NOT NULL,
    -- Logical entity/table affected, e.g. 'plan', 'subscription', 'payment'.
    entity      text NOT NULL,
    -- Primary key of the affected row (uuid or opaque id as text).
    entity_id   text,
    -- Snapshot before/after the change (NULL when not applicable).
    before      jsonb,
    after       jsonb,
    -- Request metadata.
    ip          text,
    user_agent  text,
    request_id  text,
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- ------------------------------------------------------------
-- Indexes for the common query patterns of the audit viewer.
-- ------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor      ON audit_logs (actor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity     ON audit_logs (entity, entity_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action     ON audit_logs (action, created_at DESC);

-- ------------------------------------------------------------
-- Append-only enforcement.
-- Any UPDATE or DELETE raises an exception, preserving the trail.
-- ------------------------------------------------------------
CREATE OR REPLACE FUNCTION audit_logs_block_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'table audit_logs is append-only (no % allowed)', TG_OP
        USING ERRCODE = 'restrict_violation';
END;
$$;

DROP TRIGGER IF EXISTS audit_logs_no_update ON audit_logs;
CREATE TRIGGER audit_logs_no_update
    BEFORE UPDATE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION audit_logs_block_mutation();

DROP TRIGGER IF EXISTS audit_logs_no_delete ON audit_logs;
CREATE TRIGGER audit_logs_no_delete
    BEFORE DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION audit_logs_block_mutation();
