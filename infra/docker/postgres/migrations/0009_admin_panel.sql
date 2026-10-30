-- ============================================================================
-- Migracao 0009: Painel administrativo (ADR 0008)
-- Idempotente: pode ser executada com seguranca mais de uma vez.
-- ============================================================================

-- suspended deixa de existir como estado de conta; inactive cobre o mesmo caso.
DO $$ BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_enum e
        JOIN pg_type t ON t.oid = e.enumtypid
        WHERE t.typname = 'user_status' AND e.enumlabel = 'suspended'
    ) AND NOT EXISTS (
        SELECT 1
        FROM pg_enum e
        JOIN pg_type t ON t.oid = e.enumtypid
        WHERE t.typname = 'user_status' AND e.enumlabel = 'inactive'
    ) THEN
        ALTER TYPE user_status RENAME VALUE 'suspended' TO 'inactive';
    END IF;
END $$;

ALTER TYPE user_status ADD VALUE IF NOT EXISTS 'blocked';
ALTER TYPE appointment_status ADD VALUE IF NOT EXISTS 'cancelled_by_admin';
ALTER TYPE payment_status ADD VALUE IF NOT EXISTS 'refund_pending';

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS status_reason TEXT,
    ADD COLUMN IF NOT EXISTS status_changed_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS admin_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    admin_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action VARCHAR(80) NOT NULL,
    target_type VARCHAR(40) NOT NULL,
    target_id UUID NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    ip VARCHAR(64),
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_audit_target
    ON admin_audit_log(target_type, target_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_admin_audit_admin
    ON admin_audit_log(admin_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_users_role_status ON users(role, status);

CREATE INDEX IF NOT EXISTS idx_appointments_starts_at
    ON appointments(starts_at DESC);
CREATE INDEX IF NOT EXISTS idx_appointments_status_starts
    ON appointments(status, starts_at DESC);
CREATE INDEX IF NOT EXISTS idx_appointments_client_starts
    ON appointments(client_id, starts_at DESC);
CREATE INDEX IF NOT EXISTS idx_appointments_prof_starts
    ON appointments(professional_id, starts_at DESC);

CREATE INDEX IF NOT EXISTS idx_payments_client_created
    ON payments(client_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payments_prof_created
    ON payments(professional_id, created_at DESC);
