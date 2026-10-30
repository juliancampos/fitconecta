-- +goose Up
-- ============================================================================
-- FitConecta+ Initial Migration
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "citext";
CREATE EXTENSION IF NOT EXISTS "btree_gist";

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TYPE user_role AS ENUM ('client', 'professional', 'admin');
CREATE TYPE user_status AS ENUM ('pending', 'active', 'suspended');

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email CITEXT NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role user_role NOT NULL,
    status user_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE IF NOT EXISTS professional_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    display_name VARCHAR(120) NOT NULL,
    bio TEXT,
    city VARCHAR(100) NOT NULL,
    state VARCHAR(2) NOT NULL,
    timezone VARCHAR(50) NOT NULL DEFAULT 'America/Sao_Paulo',
    session_price_cents INTEGER NOT NULL CHECK (session_price_cents > 0),
    cref_number VARCHAR(30),
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_professionals_location ON professional_profiles(city, state);

CREATE TRIGGER trg_professional_profiles_updated_at
    BEFORE UPDATE ON professional_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE IF NOT EXISTS specialties (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS professional_specialties (
    professional_id UUID NOT NULL REFERENCES professional_profiles(id) ON DELETE CASCADE,
    specialty_id INTEGER NOT NULL REFERENCES specialties(id) ON DELETE CASCADE,
    PRIMARY KEY (professional_id, specialty_id)
);

CREATE TABLE IF NOT EXISTS client_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(120) NOT NULL,
    phone VARCHAR(20),
    city VARCHAR(100),
    state VARCHAR(2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trg_client_profiles_updated_at
    BEFORE UPDATE ON client_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE IF NOT EXISTS availability_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    professional_id UUID NOT NULL REFERENCES professional_profiles(id) ON DELETE CASCADE,
    weekday SMALLINT NOT NULL CHECK (weekday BETWEEN 0 AND 6),
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    slot_minutes SMALLINT NOT NULL DEFAULT 60 CHECK (slot_minutes > 0),
    valid_from DATE,
    valid_until DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_rule_times CHECK (start_time < end_time)
);

CREATE INDEX IF NOT EXISTS idx_availability_rules_prof ON availability_rules(professional_id, weekday);

CREATE TABLE IF NOT EXISTS availability_exceptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    professional_id UUID NOT NULL REFERENCES professional_profiles(id) ON DELETE CASCADE,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    reason VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_exception_times CHECK (starts_at < ends_at)
);

CREATE INDEX IF NOT EXISTS idx_availability_exceptions_prof ON availability_exceptions(professional_id, starts_at, ends_at);

CREATE TYPE appointment_status AS ENUM (
    'pending_payment',
    'confirmed',
    'cancelled_by_client',
    'cancelled_by_professional',
    'completed',
    'no_show'
);

CREATE TABLE IF NOT EXISTS appointments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    professional_id UUID NOT NULL REFERENCES professional_profiles(id) ON DELETE RESTRICT,
    client_id UUID NOT NULL REFERENCES client_profiles(id) ON DELETE RESTRICT,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    period TSTZRANGE GENERATED ALWAYS AS (tstzrange(starts_at, ends_at)) STORED,
    status appointment_status NOT NULL DEFAULT 'pending_payment',
    reserved_until TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    cancel_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_appointment_times CHECK (starts_at < ends_at),
    CONSTRAINT no_overlap_professional_appointment EXCLUDE USING gist (
        professional_id WITH =,
        period WITH &&
    ) WHERE (status IN ('pending_payment', 'confirmed'))
);

CREATE INDEX IF NOT EXISTS idx_appointments_client ON appointments(client_id, starts_at);
CREATE INDEX IF NOT EXISTS idx_appointments_prof ON appointments(professional_id, starts_at);
CREATE INDEX IF NOT EXISTS idx_appointments_status ON appointments(status);

CREATE TRIGGER trg_appointments_updated_at
    BEFORE UPDATE ON appointments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TYPE payment_method AS ENUM ('pix', 'credit_card');
CREATE TYPE payment_status AS ENUM ('pending', 'paid', 'failed', 'refunded');

CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    appointment_id UUID NOT NULL UNIQUE REFERENCES appointments(id) ON DELETE RESTRICT,
    client_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    professional_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    platform_fee_cents INTEGER NOT NULL CHECK (platform_fee_cents >= 0),
    professional_net_cents INTEGER NOT NULL CHECK (professional_net_cents >= 0),
    method payment_method NOT NULL,
    status payment_status NOT NULL DEFAULT 'pending',
    gateway_transaction_id VARCHAR(120),
    gateway_payload JSONB,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_split_sum CHECK (platform_fee_cents + professional_net_cents = amount_cents)
);

CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
CREATE INDEX IF NOT EXISTS idx_payments_gateway_tx ON payments(gateway_transaction_id);

CREATE TRIGGER trg_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TYPE device_platform AS ENUM ('android', 'ios', 'web');

CREATE TABLE IF NOT EXISTS device_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform device_platform NOT NULL,
    token TEXT NOT NULL UNIQUE,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_device_tokens_user ON device_tokens(user_id);

INSERT INTO specialties (name) VALUES 
    ('Musculação'),
    ('Treinamento Funcional'),
    ('Pilates'),
    ('Corrida'),
    ('Cross Training'),
    ('Yoga'),
    ('Alongamento / Mobilidade'),
    ('Condicionamento Físico')
ON CONFLICT (name) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS device_tokens CASCADE;
DROP TABLE IF EXISTS payments CASCADE;
DROP TABLE IF EXISTS appointments CASCADE;
DROP TABLE IF EXISTS availability_exceptions CASCADE;
DROP TABLE IF EXISTS availability_rules CASCADE;
DROP TABLE IF EXISTS client_profiles CASCADE;
DROP TABLE IF EXISTS professional_specialties CASCADE;
DROP TABLE IF EXISTS specialties CASCADE;
DROP TABLE IF EXISTS professional_profiles CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TYPE IF EXISTS device_platform;
DROP TYPE IF EXISTS payment_status;
DROP TYPE IF EXISTS payment_method;
DROP TYPE IF EXISTS appointment_status;
DROP TYPE IF EXISTS user_status;
DROP TYPE IF EXISTS user_role;
DROP FUNCTION IF EXISTS update_updated_at_column;
