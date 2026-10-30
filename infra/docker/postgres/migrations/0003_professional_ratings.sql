-- ============================================================================
-- Migracao 0003: Reputacao do profissional (nota media e total de avaliacoes)
-- Idempotente: pode ser executada com seguranca mais de uma vez.
-- ============================================================================

ALTER TABLE professional_profiles
    ADD COLUMN IF NOT EXISTS rating_avg NUMERIC(2,1) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS reviews_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE professional_profiles
    DROP CONSTRAINT IF EXISTS chk_rating_range;

ALTER TABLE professional_profiles
    ADD CONSTRAINT chk_rating_range CHECK (rating_avg >= 0 AND rating_avg <= 5);
