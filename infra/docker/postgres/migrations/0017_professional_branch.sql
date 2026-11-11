-- ============================================================================
-- Migracao 0017: ramo do profissional e cadastro de nutricionistas (ADR 0014)
-- branch: 'physical_education' (default, registros existentes) ou 'nutrition'.
-- crn_number/crn_verified_at: espelham o padrao do CREF (ADR 0013).
-- NULL em crn_verified_at = ainda nao verificado (em analise).
-- ============================================================================

ALTER TABLE professional_profiles
    ADD COLUMN IF NOT EXISTS branch VARCHAR(20) NOT NULL DEFAULT 'physical_education';

ALTER TABLE professional_profiles
    ADD COLUMN IF NOT EXISTS crn_number VARCHAR(30);

ALTER TABLE professional_profiles
    ADD COLUMN IF NOT EXISTS crn_verified_at TIMESTAMPTZ;

-- ADD CONSTRAINT nao tem IF NOT EXISTS: protege via bloco DO
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'professional_profiles_branch_check'
    ) THEN
        ALTER TABLE professional_profiles
            ADD CONSTRAINT professional_profiles_branch_check
            CHECK (branch IN ('physical_education', 'nutrition'));
    END IF;
END $$;

-- Especialidades de nutricao (sem elas o cadastro de nutricionista forcarias
-- modalidades EF). Senha dos usuarios seed: "senha12345" (padrao do seed 0004).
INSERT INTO specialties (name) VALUES
    ('Consulta nutricional'),
    ('Plano alimentar'),
    ('Nutrição esportiva'),
    ('Reeducação alimentar'),
    ('Nutrição clínica')
ON CONFLICT (name) DO NOTHING;

-- Nutricionistas de exemplo (CRNs ficticios): demonstram o filtro por ramo e o
-- icone no card. O primeiro com selo verificado, o segundo "em analise",
-- para exercitar os dois estados do card.
INSERT INTO users (email, password_hash, role, status) VALUES
    ('marina.ribeiro@fitconecta.dev',  '$2a$10$oBYTAKptcYe9joffx8/OGeOTmCiukHtBaUOIKkrLJOyK/3L1A8ImG', 'professional', 'active'),
    ('patricia.mendes@fitconecta.dev', '$2a$10$oBYTAKptcYe9joffx8/OGeOTmCiukHtBaUOIKkrLJOyK/3L1A8ImG', 'professional', 'active')
ON CONFLICT (email) DO NOTHING;

INSERT INTO professional_profiles
    (user_id, display_name, bio, city, state, session_price_cents, rating_avg, reviews_count, approved_at,
     branch, crn_number, crn_verified_at)
SELECT u.id, v.display_name, v.bio, v.city, v.state, v.session_price_cents, v.rating_avg, v.reviews_count, NOW(),
       'nutrition', v.crn_number, v.crn_verified_at
FROM (VALUES
    ('marina.ribeiro@fitconecta.dev',  'Marina Ribeiro',  'Nutricionista esportiva com foco em performance e composição corporal de atletas amadores.', 'Belo Horizonte', 'MG', 11000, 4.9::numeric, 96,  'CRN3 51284', NOW()),
    ('patricia.mendes@fitconecta.dev', 'Patrícia Mendes','Nutricionista clínica para reeducação alimentar e acompanhamento de dietas com restrições.', 'São Paulo', 'SP', 13000, 4.8::numeric, 74, 'CRN3 43107', NULL)
) AS v(email, display_name, bio, city, state, session_price_cents, rating_avg, reviews_count, crn_number, crn_verified_at)
JOIN users u ON u.email = v.email
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO professional_specialties (professional_id, specialty_id)
SELECT p.id, s.id
FROM (VALUES
    ('marina.ribeiro@fitconecta.dev',  'Nutrição esportiva'),
    ('marina.ribeiro@fitconecta.dev',  'Plano alimentar'),
    ('patricia.mendes@fitconecta.dev', 'Consulta nutricional'),
    ('patricia.mendes@fitconecta.dev', 'Reeducação alimentar')
) AS v(email, specialty_name)
JOIN users u ON u.email = v.email
JOIN professional_profiles p ON p.user_id = u.id
JOIN specialties s ON s.name = v.specialty_name
ON CONFLICT DO NOTHING;

INSERT INTO availability_rules (professional_id, weekday, start_time, end_time, slot_minutes)
SELECT p.id, w.weekday,
       make_time(t.hour, 0, 0),
       make_time(t.hour + 1, 0, 0),
       60
FROM (VALUES
    ('marina.ribeiro@fitconecta.dev',  8),
    ('marina.ribeiro@fitconecta.dev',  18),
    ('patricia.mendes@fitconecta.dev', 9),
    ('patricia.mendes@fitconecta.dev', 17)
) AS t(email, hour)
CROSS JOIN (VALUES (1), (2), (3), (4), (5), (6)) AS w(weekday)
JOIN users u ON u.email = t.email
JOIN professional_profiles p ON p.user_id = u.id
WHERE NOT EXISTS (
    SELECT 1 FROM availability_rules ar
    WHERE ar.professional_id = p.id
      AND ar.weekday = w.weekday
      AND ar.start_time = make_time(t.hour, 0, 0)
      AND ar.end_time = make_time(t.hour + 1, 0, 0)
);
