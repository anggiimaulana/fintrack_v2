-- 0035_payment_card_tokens.sql
-- Store tokenized payment cards for recurring and auto-charge billing.
-- Never stores PAN, CVV, or raw card data (PCI-DSS SAQ-A compliant).

CREATE TABLE IF NOT EXISTS payment_card_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    gateway VARCHAR(32) NOT NULL DEFAULT 'midtrans',
    token_id TEXT NOT NULL,
    masked_card VARCHAR(32) NOT NULL,
    card_type VARCHAR(32),
    bank VARCHAR(32),
    is_default BOOLEAN NOT NULL DEFAULT false,
    expires_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_payment_card_tokens_user_gateway_token UNIQUE (user_id, gateway, token_id)
);

CREATE INDEX IF NOT EXISTS idx_payment_card_tokens_user_id ON payment_card_tokens(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_payment_card_tokens_user_default ON payment_card_tokens(user_id) WHERE is_default = true;

DROP TRIGGER IF EXISTS on_payment_card_tokens_updated ON payment_card_tokens;
CREATE TRIGGER on_payment_card_tokens_updated
    BEFORE UPDATE ON payment_card_tokens
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- Grant permissions to application role if it exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'fintrack_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE payment_card_tokens TO fintrack_app;
    END IF;
END $$;
