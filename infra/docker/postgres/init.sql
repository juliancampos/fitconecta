-- ============================================================================
-- FitConecta+ Database Initialization Script
-- Based on ADR 0001: Modelagem e Estrutura do Banco de Dados Relacional
-- ============================================================================

-- Habilitar extensoes necessarias
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "citext";
CREATE EXTENSION IF NOT EXISTS "btree_gist";

-- Funcao trigger para atualizacao automatica do campo updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 1. USUARIOS E AUTENTICACAO
-- ============================================================================
CREATE TYPE user_role AS ENUM ('client', 'professional', 'admin');
CREATE TYPE user_status AS ENUM ('pending', 'active', 'blocked', 'inactive');

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email CITEXT NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role user_role NOT NULL,
    status user_status NOT NULL DEFAULT 'pending',
    status_reason TEXT,
    status_changed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_role_status ON users(role, status);

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- 2. PERFIS DOS PROFISSIONAIS (EXCLUSIVO AUTONOMOS - PESSOA FISICA)
-- ============================================================================
CREATE TABLE professional_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    display_name VARCHAR(120) NOT NULL,
    bio TEXT,
    audience_other TEXT,       -- texto do "Outro publico especifico" (ADR 0011)
    audience_experience TEXT,  -- experiencia opcional com os publicos especificos
    city VARCHAR(100) NOT NULL,
    state VARCHAR(2) NOT NULL,
    timezone VARCHAR(50) NOT NULL DEFAULT 'America/Sao_Paulo',
    session_price_cents INTEGER NOT NULL CHECK (session_price_cents > 0),
    cref_number VARCHAR(30),
    cref_verified_at TIMESTAMPTZ, -- ultima validacao positiva no portal do CREF6 (ADR 0013)
    avatar_url TEXT,
    rating_avg NUMERIC(2,1) NOT NULL DEFAULT 0,
    reviews_count INTEGER NOT NULL DEFAULT 0,
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_rating_range CHECK (rating_avg >= 0 AND rating_avg <= 5)
);

CREATE INDEX idx_professionals_location ON professional_profiles(city, state);

CREATE TRIGGER trg_professional_profiles_updated_at
    BEFORE UPDATE ON professional_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Galeria de fotos do profissional (ate 3 imagens exibidas no perfil)
CREATE TABLE professional_photos (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    professional_id UUID NOT NULL REFERENCES professional_profiles(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    position SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_professional_photos_prof ON professional_photos(professional_id, position);

-- Especialidades / Modalidades
CREATE TABLE specialties (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE
);

CREATE TABLE professional_specialties (
    professional_id UUID NOT NULL REFERENCES professional_profiles(id) ON DELETE CASCADE,
    specialty_id INTEGER NOT NULL REFERENCES specialties(id) ON DELETE CASCADE,
    PRIMARY KEY (professional_id, specialty_id)
);

-- Bairros de atuacao (ver ADR 0004)
CREATE TABLE neighborhoods (
    id SERIAL PRIMARY KEY,
    state VARCHAR(2) NOT NULL,
    city VARCHAR(100) NOT NULL,
    name CITEXT NOT NULL,
    slug VARCHAR(140) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_neighborhoods_name UNIQUE (state, city, name),
    CONSTRAINT uq_neighborhoods_slug UNIQUE (state, city, slug)
);

CREATE INDEX idx_neighborhoods_city ON neighborhoods(state, city);
CREATE INDEX idx_neighborhoods_slug ON neighborhoods(slug);

CREATE TRIGGER trg_neighborhoods_updated_at
    BEFORE UPDATE ON neighborhoods
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE professional_neighborhoods (
    professional_id UUID NOT NULL REFERENCES professional_profiles(id) ON DELETE CASCADE,
    neighborhood_id INTEGER NOT NULL REFERENCES neighborhoods(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (professional_id, neighborhood_id)
);

CREATE INDEX idx_prof_neighborhoods_neighborhood ON professional_neighborhoods(neighborhood_id);

-- Objetivos, publicos e locais de atendimento (ADR 0011)
CREATE TABLE objectives (
    id SERIAL PRIMARY KEY,
    name VARCHAR(80) NOT NULL UNIQUE
);

CREATE TABLE audiences (
    id SERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL UNIQUE,
    is_specific BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE service_locations (
    id SERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL UNIQUE
);

CREATE TABLE professional_objectives (
    professional_id UUID NOT NULL REFERENCES professional_profiles(id) ON DELETE CASCADE,
    objective_id INTEGER NOT NULL REFERENCES objectives(id) ON DELETE RESTRICT,
    PRIMARY KEY (professional_id, objective_id)
);

CREATE TABLE professional_audiences (
    professional_id UUID NOT NULL REFERENCES professional_profiles(id) ON DELETE CASCADE,
    audience_id INTEGER NOT NULL REFERENCES audiences(id) ON DELETE RESTRICT,
    PRIMARY KEY (professional_id, audience_id)
);

CREATE TABLE professional_locations (
    professional_id UUID NOT NULL REFERENCES professional_profiles(id) ON DELETE CASCADE,
    location_id INTEGER NOT NULL REFERENCES service_locations(id) ON DELETE RESTRICT,
    PRIMARY KEY (professional_id, location_id)
);

-- ============================================================================
-- 3. PERFIL DO CLIENTE
-- ============================================================================
CREATE TYPE client_sex AS ENUM ('male', 'female', 'other');
CREATE TYPE client_activity_level AS ENUM ('sedentary', 'casual', 'frequent');

CREATE TABLE client_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(120) NOT NULL,
    phone VARCHAR(20),
    city VARCHAR(100),
    state VARCHAR(2),
    birth_date DATE,              -- ADR 0012: dados opcionais, preenchidos na pagina de perfil
    sex client_sex,
    height_cm SMALLINT,
    weight_kg NUMERIC(5,1),
    activity_level client_activity_level,
    objective TEXT,
    cpf VARCHAR(11),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_client_birth_date CHECK (birth_date IS NULL OR birth_date >= DATE '1900-01-01'),
    CONSTRAINT chk_client_height CHECK (height_cm IS NULL OR height_cm BETWEEN 100 AND 250),
    CONSTRAINT chk_client_weight CHECK (weight_kg IS NULL OR weight_kg BETWEEN 30 AND 300),
    CONSTRAINT chk_client_objective_length CHECK (objective IS NULL OR char_length(objective) <= 500),
    CONSTRAINT chk_client_cpf CHECK (cpf IS NULL OR cpf ~ '^[0-9]{11}$')
);

CREATE TRIGGER trg_client_profiles_updated_at
    BEFORE UPDATE ON client_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- 4. DISPONIBILIDADE E AGENDA DO PROFISSIONAL
-- ============================================================================
CREATE TABLE availability_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    professional_id UUID NOT NULL REFERENCES professional_profiles(id) ON DELETE CASCADE,
    weekday SMALLINT NOT NULL CHECK (weekday BETWEEN 0 AND 6), -- 0=Domingo, 6=Sabado
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    slot_minutes SMALLINT NOT NULL DEFAULT 60 CHECK (slot_minutes > 0),
    valid_from DATE,
    valid_until DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_rule_times CHECK (start_time < end_time)
);

CREATE INDEX idx_availability_rules_prof ON availability_rules(professional_id, weekday);

CREATE TABLE availability_exceptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    professional_id UUID NOT NULL REFERENCES professional_profiles(id) ON DELETE CASCADE,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    reason VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_exception_times CHECK (starts_at < ends_at)
);

CREATE INDEX idx_availability_exceptions_prof ON availability_exceptions(professional_id, starts_at, ends_at);

-- ============================================================================
-- 5. AGENDAMENTOS (APPOINTMENTS) COM CONSTRAINT DE EXCLUSAO
-- ============================================================================
CREATE TYPE appointment_status AS ENUM (
    'pending_payment',
    'confirmed',
    'cancelled_by_client',
    'cancelled_by_professional',
    'cancelled_by_admin',
    'completed',
    'no_show',
    'expired'
);

CREATE TYPE appointment_location AS ENUM ('home', 'studio', 'online');

CREATE TABLE appointments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    professional_id UUID NOT NULL REFERENCES professional_profiles(id) ON DELETE RESTRICT,
    client_id UUID NOT NULL REFERENCES client_profiles(id) ON DELETE RESTRICT,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    period TSTZRANGE GENERATED ALWAYS AS (tstzrange(starts_at, ends_at)) STORED,
    specialty_id INTEGER REFERENCES specialties(id) ON DELETE RESTRICT, -- tipo de aula (ADR 0005)
    location_type appointment_location, -- tipo de local (ADR 0005)
    location_notes TEXT,                -- descricao livre (ADR 0005)
    status appointment_status NOT NULL DEFAULT 'pending_payment',
    reserved_until TIMESTAMPTZ, -- expiracao da pre-reserva caso o pagamento nao seja concluido
    cancelled_at TIMESTAMPTZ,
    cancel_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_appointment_times CHECK (starts_at < ends_at),
    -- Garante que um profissional nao possa ter dois horarios sobrepostos em status ativos
    CONSTRAINT no_overlap_professional_appointment EXCLUDE USING gist (
        professional_id WITH =,
        period WITH &&
    ) WHERE (status IN ('pending_payment', 'confirmed'))
);

CREATE INDEX idx_appointments_client ON appointments(client_id, starts_at);
CREATE INDEX idx_appointments_prof ON appointments(professional_id, starts_at);
CREATE INDEX idx_appointments_status ON appointments(status);
CREATE INDEX idx_appointments_specialty ON appointments(specialty_id);
CREATE INDEX idx_appointments_starts_at ON appointments(starts_at DESC);
CREATE INDEX idx_appointments_status_starts ON appointments(status, starts_at DESC);
CREATE INDEX idx_appointments_client_starts ON appointments(client_id, starts_at DESC);
CREATE INDEX idx_appointments_prof_starts ON appointments(professional_id, starts_at DESC);
CREATE INDEX idx_appointments_pending_expiry
    ON appointments(status, reserved_until)
    WHERE status = 'pending_payment';

CREATE TRIGGER trg_appointments_updated_at
    BEFORE UPDATE ON appointments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- 6. PAGAMENTOS E SPLIT (10% PLATAFORMA / 90% PROFISSIONAL)
-- ============================================================================
CREATE TYPE payment_method AS ENUM ('pix', 'credit_card');
CREATE TYPE payment_status AS ENUM ('pending', 'paid', 'failed', 'refund_pending', 'refunded', 'expired');

CREATE TABLE payments (
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
    provider VARCHAR(40),
    provider_payment_id VARCHAR(255),
    provider_fee_cents INTEGER,
    expires_at TIMESTAMPTZ,
    raw_status VARCHAR(80),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_split_sum CHECK (platform_fee_cents + professional_net_cents = amount_cents)
);

CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_gateway_tx ON payments(gateway_transaction_id);
CREATE INDEX idx_payments_client_created ON payments(client_id, created_at DESC);
CREATE INDEX idx_payments_prof_created ON payments(professional_id, created_at DESC);
CREATE INDEX idx_payments_status_expires ON payments(status, expires_at);
CREATE UNIQUE INDEX idx_payments_provider_payment
    ON payments(provider, provider_payment_id)
    WHERE provider_payment_id IS NOT NULL;

CREATE TABLE payment_webhook_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider VARCHAR(40) NOT NULL,
    provider_event_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(80) NOT NULL,
    payload JSONB NOT NULL,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, provider_event_id)
);

CREATE TRIGGER trg_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- 6.1 AUDITORIA DO PAINEL ADMINISTRATIVO (ADR 0008)
-- ============================================================================
CREATE TABLE admin_audit_log (
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

CREATE INDEX idx_admin_audit_target ON admin_audit_log(target_type, target_id, created_at DESC);
CREATE INDEX idx_admin_audit_admin ON admin_audit_log(admin_user_id, created_at DESC);

-- ============================================================================
-- 7. PUSH NOTIFICATIONS (PREPARADO PARA FASE POSTERIOR)
-- ============================================================================
CREATE TYPE device_platform AS ENUM ('android', 'ios', 'web');

CREATE TABLE device_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform device_platform NOT NULL,
    token TEXT NOT NULL UNIQUE,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_device_tokens_user ON device_tokens(user_id);

-- ============================================================================
-- SEED BASICO: MODALIDADES/ESPECIALIDADES INICIAIS
-- ============================================================================
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

-- Seeds dos catalogos do ADR 0011 (a migration 0011 normaliza o catalogo de
-- servicos: renomeia, mescla duplicatas e move objetivos para objectives)
INSERT INTO objectives (name) VALUES
    ('Emagrecimento'),
    ('Ganho de massa muscular'),
    ('Ganho de força'),
    ('Condicionamento físico'),
    ('Mobilidade e flexibilidade'),
    ('Iniciação à atividade física'),
    ('Retorno à atividade física'),
    ('Saúde e bem-estar'),
    ('Preparação para corrida'),
    ('Performance esportiva'),
    ('Preparação para prova ou teste físico')
ON CONFLICT (name) DO NOTHING;

INSERT INTO audiences (name, is_specific) VALUES
    ('Adultos em geral', FALSE),
    ('Iniciantes', FALSE),
    ('Pessoas com 60 anos ou mais', TRUE),
    ('Gestantes', TRUE),
    ('Mulheres no pós-parto', TRUE),
    ('Pessoas com deficiência', TRUE),
    ('Pessoas com diabetes', TRUE),
    ('Pessoas com hipertensão', TRUE),
    ('Pessoas com obesidade', TRUE),
    ('Pessoas em retorno após lesão, com liberação para atividade física', TRUE),
    ('Outro público específico', TRUE)
ON CONFLICT (name) DO NOTHING;

INSERT INTO service_locations (name) VALUES
    ('Residência do cliente'),
    ('Academia do condomínio'),
    ('Academia comercial (quando houver autorização)'),
    ('Parque ou espaço público'),
    ('Hotel'),
    ('Estúdio ou espaço próprio do profissional')
ON CONFLICT (name) DO NOTHING;
