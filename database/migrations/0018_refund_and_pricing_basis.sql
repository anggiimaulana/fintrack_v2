-- ============================================================
-- 0018_refund_and_pricing_basis.sql
--
-- Two additive changes:
--
-- 1) token_ledger.entry_type gains 'refund'. A refund returns tokens that were
--    already charged (used_tokens) when a feature fails AFTER a commit — e.g. an
--    OCR scan whose provider call returned no usable result. Without it the user
--    pays for work that never completed.
--
-- 2) pricing_rules gains an explicit cost basis:
--      - provider_cost_per_1k_idr : what WE pay the provider per 1k tokens
--      - price_unit               : 'per_1k' | 'per_10k' (how base price reads)
--    This lets a Super Admin enter "Rp 3.000 / 10.000 token" directly and see the
--    margin (sell − buy) per modality.
--
-- Additive & idempotent (expand only; nothing is dropped or rewritten).
-- ============================================================

-- 1) Allow the new ledger entry type. The original CHECK (0008) enumerated the
--    five existing types; replace it with one that also permits 'refund'.
ALTER TABLE token_ledger DROP CONSTRAINT IF EXISTS token_ledger_entry_type_check;
ALTER TABLE token_ledger
    ADD CONSTRAINT token_ledger_entry_type_check
    CHECK (entry_type IN ('reserve','commit','release','grant','adjust','refund'));

-- 2) Pricing cost basis (nullable/defaulted so existing rows stay valid).
ALTER TABLE pricing_rules
    ADD COLUMN IF NOT EXISTS provider_cost_per_1k_idr bigint NOT NULL DEFAULT 0;
ALTER TABLE pricing_rules
    ADD COLUMN IF NOT EXISTS price_unit varchar(8) NOT NULL DEFAULT 'per_1k';

ALTER TABLE pricing_rules DROP CONSTRAINT IF EXISTS pricing_rules_price_unit_check;
ALTER TABLE pricing_rules
    ADD CONSTRAINT pricing_rules_price_unit_check
    CHECK (price_unit IN ('per_1k','per_10k'));

-- down:
--   ALTER TABLE pricing_rules DROP CONSTRAINT IF EXISTS pricing_rules_price_unit_check;
--   ALTER TABLE pricing_rules DROP COLUMN IF EXISTS price_unit;
--   ALTER TABLE pricing_rules DROP COLUMN IF EXISTS provider_cost_per_1k_idr;
--   ALTER TABLE token_ledger DROP CONSTRAINT IF EXISTS token_ledger_entry_type_check;
--   ALTER TABLE token_ledger ADD CONSTRAINT token_ledger_entry_type_check
--       CHECK (entry_type IN ('reserve','commit','release','grant','adjust'));
