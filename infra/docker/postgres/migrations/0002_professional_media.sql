-- ============================================================================
-- Migracao 0002: Midias do profissional (avatar + galeria de fotos)
-- ADR 0003 - Armazenamento de Midias dos Profissionais
-- Idempotente: pode ser executada com seguranca mais de uma vez.
-- ============================================================================

ALTER TABLE professional_profiles
    ADD COLUMN IF NOT EXISTS avatar_url TEXT;

CREATE TABLE IF NOT EXISTS professional_photos (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    professional_id UUID NOT NULL REFERENCES professional_profiles(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    position SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_professional_photos_prof
    ON professional_photos(professional_id, position);
