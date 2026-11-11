-- ============================================================================
-- Migracao 0014: nivel de atividade e objetivo do cliente (ADR 0012, revisao)
-- Substitui body_type ("tipo fisico") por activity_level ("faco atividades")
-- e adiciona o objetivo em texto livre. Aditiva e idempotente.
-- ============================================================================

-- 1) Enum (CREATE TYPE nao tem IF NOT EXISTS)
DO $$ BEGIN
    CREATE TYPE client_activity_level AS ENUM ('sedentary', 'casual', 'frequent');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- 2) Colunas novas (nullable: nada e obrigatorio)
ALTER TABLE client_profiles
    ADD COLUMN IF NOT EXISTS activity_level client_activity_level,
    ADD COLUMN IF NOT EXISTS objective TEXT;

ALTER TABLE client_profiles DROP CONSTRAINT IF EXISTS chk_client_objective_length;
ALTER TABLE client_profiles ADD CONSTRAINT chk_client_objective_length
    CHECK (objective IS NULL OR char_length(objective) <= 500);

-- 3) body_type nunca chegou a ser preenchido; sai do schema em vez de virar coluna morta
ALTER TABLE client_profiles DROP COLUMN IF EXISTS body_type;
DROP TYPE IF EXISTS client_body_type;
