-- ============================================================================
-- Migracao 0013: dados pessoais e fisicos do cliente (ADR 0012)
-- Aditiva e idempotente. Nenhum backfill possivel: birth_date nunca foi salva.
-- Nenhum campo e obrigatorio e nada aqui altera o cadastro inicial.
-- ============================================================================

-- 1) Enums (CREATE TYPE nao tem IF NOT EXISTS)
DO $$ BEGIN
    CREATE TYPE client_sex AS ENUM ('male', 'female', 'other');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE client_body_type AS ENUM ('lean', 'average', 'muscular', 'robust');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- 2) Colunas (todas nullable: nada e obrigatorio para agendar)
ALTER TABLE client_profiles
    ADD COLUMN IF NOT EXISTS birth_date DATE,
    ADD COLUMN IF NOT EXISTS sex client_sex,
    ADD COLUMN IF NOT EXISTS height_cm SMALLINT,
    ADD COLUMN IF NOT EXISTS weight_kg NUMERIC(5,1),
    ADD COLUMN IF NOT EXISTS body_type client_body_type;

-- 3) Faixas plausiveis (DROP + ADD para manter idempotencia)
ALTER TABLE client_profiles DROP CONSTRAINT IF EXISTS chk_client_birth_date;
ALTER TABLE client_profiles ADD CONSTRAINT chk_client_birth_date
    CHECK (birth_date IS NULL OR birth_date >= DATE '1900-01-01');

ALTER TABLE client_profiles DROP CONSTRAINT IF EXISTS chk_client_height;
ALTER TABLE client_profiles ADD CONSTRAINT chk_client_height
    CHECK (height_cm IS NULL OR height_cm BETWEEN 100 AND 250);

ALTER TABLE client_profiles DROP CONSTRAINT IF EXISTS chk_client_weight;
ALTER TABLE client_profiles ADD CONSTRAINT chk_client_weight
    CHECK (weight_kg IS NULL OR weight_kg BETWEEN 30 AND 300);
