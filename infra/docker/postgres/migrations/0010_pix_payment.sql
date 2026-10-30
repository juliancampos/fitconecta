-- ============================================================================
-- Migracao 0010: Pagamento Pix obrigatorio no agendamento (ADR 0010)
-- Idempotente: pode ser executada com seguranca mais de uma vez.
-- ============================================================================

ALTER TYPE appointment_status ADD VALUE IF NOT EXISTS 'expired';
ALTER TYPE payment_status ADD VALUE IF NOT EXISTS 'expired';

ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS provider VARCHAR(40),
    ADD COLUMN IF NOT EXISTS provider_payment_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS provider_fee_cents INTEGER,
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS raw_status VARCHAR(80),
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_provider_payment
    ON payments(provider, provider_payment_id)
    WHERE provider_payment_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_payments_status_expires
    ON payments(status, expires_at);

CREATE INDEX IF NOT EXISTS idx_appointments_pending_expiry
    ON appointments(status, reserved_until)
    WHERE status = 'pending_payment';

CREATE TABLE IF NOT EXISTS payment_webhook_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider VARCHAR(40) NOT NULL,
    provider_event_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(80) NOT NULL,
    payload JSONB NOT NULL,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, provider_event_id)
);
