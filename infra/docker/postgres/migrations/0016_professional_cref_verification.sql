-- ============================================================================
-- Migracao 0016: verificacao do CREF do profissional (ADR 0013)
-- Data da ultima validacao positiva no portal do CREF6.
-- NULL = ainda nao verificado (em analise).
-- ============================================================================

ALTER TABLE professional_profiles
    ADD COLUMN IF NOT EXISTS cref_verified_at TIMESTAMPTZ;
