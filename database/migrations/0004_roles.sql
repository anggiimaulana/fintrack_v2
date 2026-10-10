-- ============================================================
-- 0004_roles.sql
-- Least-privilege application role. Idempotent.
-- Invoked via: psql -v ON_ERROR_STOP=1 -v app_password=... -f 0004_roles.sql
--
-- fintrack_app: DML only (no DDL). Statement timeouts set as a
-- safety net for the API.
-- ============================================================

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'fintrack_app') THEN
        CREATE ROLE fintrack_app LOGIN;
    END IF;
END $$;

ALTER ROLE fintrack_app WITH LOGIN PASSWORD :'app_password';

-- Grant on the database this script is currently connected to
-- (works for both `fintrack` and `fintrack_test`).
DO $$
BEGIN
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO fintrack_app', current_database());
END $$;

GRANT USAGE ON SCHEMA public TO fintrack_app;

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO fintrack_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO fintrack_app;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO fintrack_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO fintrack_app;

ALTER ROLE fintrack_app SET statement_timeout = '10s';
ALTER ROLE fintrack_app SET idle_in_transaction_session_timeout = '30s';

-- Remove default CREATE privilege from PUBLIC on public schema.
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
