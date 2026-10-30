-- ============================================================================
-- Migracao 0005: Bairros de atuacao dos profissionais (ADR 0004)
-- Idempotente: pode ser executada com seguranca mais de uma vez.
-- ============================================================================

-- Catalogo de bairros (cresce organicamente, como specialties)
CREATE TABLE IF NOT EXISTS neighborhoods (
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

CREATE INDEX IF NOT EXISTS idx_neighborhoods_city ON neighborhoods(state, city);
CREATE INDEX IF NOT EXISTS idx_neighborhoods_slug ON neighborhoods(slug);

DROP TRIGGER IF EXISTS trg_neighborhoods_updated_at ON neighborhoods;
CREATE TRIGGER trg_neighborhoods_updated_at
    BEFORE UPDATE ON neighborhoods
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Associacao N:N: profissional x bairros atendidos
CREATE TABLE IF NOT EXISTS professional_neighborhoods (
    professional_id UUID NOT NULL REFERENCES professional_profiles(id) ON DELETE CASCADE,
    neighborhood_id INTEGER NOT NULL REFERENCES neighborhoods(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (professional_id, neighborhood_id)
);

CREATE INDEX IF NOT EXISTS idx_prof_neighborhoods_neighborhood
    ON professional_neighborhoods(neighborhood_id);
