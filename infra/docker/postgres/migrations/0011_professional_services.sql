-- ============================================================================
-- Migracao 0011: Servicos, objetivos, publicos e locais (ADR 0011)
-- Fase 1-2: apenas aditiva. Precos e agendamento ficam para a 0012 (fase 3).
-- Idempotente: pode ser executada com seguranca mais de uma vez.
-- ============================================================================

-- 1) Catalogos novos
CREATE TABLE IF NOT EXISTS objectives (
    id SERIAL PRIMARY KEY,
    name VARCHAR(80) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS audiences (
    id SERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL UNIQUE,
    is_specific BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS service_locations (
    id SERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL UNIQUE
);

-- 2) Vinculos N:N com o profissional
CREATE TABLE IF NOT EXISTS professional_objectives (
    professional_id UUID NOT NULL REFERENCES professional_profiles(id) ON DELETE CASCADE,
    objective_id INTEGER NOT NULL REFERENCES objectives(id) ON DELETE RESTRICT,
    PRIMARY KEY (professional_id, objective_id)
);

CREATE TABLE IF NOT EXISTS professional_audiences (
    professional_id UUID NOT NULL REFERENCES professional_profiles(id) ON DELETE CASCADE,
    audience_id INTEGER NOT NULL REFERENCES audiences(id) ON DELETE RESTRICT,
    PRIMARY KEY (professional_id, audience_id)
);

CREATE TABLE IF NOT EXISTS professional_locations (
    professional_id UUID NOT NULL REFERENCES professional_profiles(id) ON DELETE CASCADE,
    location_id INTEGER NOT NULL REFERENCES service_locations(id) ON DELETE RESTRICT,
    PRIMARY KEY (professional_id, location_id)
);

-- 3) Textos livres do publico (ADR 0011 §2.3)
ALTER TABLE professional_profiles
    ADD COLUMN IF NOT EXISTS audience_other TEXT,
    ADD COLUMN IF NOT EXISTS audience_experience TEXT;

-- 4) Seeds dos catalogos
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

-- 5) Normalizacao do catalogo de servicos
-- 5.1 Servicos faltantes
INSERT INTO specialties (name) VALUES
    ('Preparação física esportiva'),
    ('Preparação para TAF e testes físicos de concursos'),
    ('Avaliação física')
ON CONFLICT (name) DO NOTHING;

-- 5.2 Renomes (mantem os vinculos existentes, pois muda apenas o nome)
UPDATE specialties SET name = 'Musculação e treino de força' WHERE name = 'Musculação';
UPDATE specialties SET name = 'Treinamento de corrida' WHERE name = 'Corrida';
UPDATE specialties SET name = 'Pilates solo' WHERE name = 'Pilates';
UPDATE specialties SET name = 'Alongamento e mobilidade' WHERE name = 'Alongamento / Mobilidade';

-- 5.3 Duplicatas: 'Funcional' -> 'Treinamento Funcional'
INSERT INTO professional_specialties (professional_id, specialty_id)
SELECT ps.professional_id, target.id
FROM professional_specialties ps
JOIN specialties src ON src.id = ps.specialty_id AND src.name = 'Funcional'
CROSS JOIN specialties target
WHERE target.name = 'Treinamento Funcional'
ON CONFLICT DO NOTHING;

DELETE FROM professional_specialties ps
USING specialties s
WHERE ps.specialty_id = s.id AND s.name = 'Funcional';

DELETE FROM specialties WHERE name = 'Funcional';

-- 5.4 Duplicatas: 'Mobilidade' -> 'Alongamento e mobilidade'
INSERT INTO professional_specialties (professional_id, specialty_id)
SELECT ps.professional_id, target.id
FROM professional_specialties ps
JOIN specialties src ON src.id = ps.specialty_id AND src.name = 'Mobilidade'
CROSS JOIN specialties target
WHERE target.name = 'Alongamento e mobilidade'
ON CONFLICT DO NOTHING;

DELETE FROM professional_specialties ps
USING specialties s
WHERE ps.specialty_id = s.id AND s.name = 'Mobilidade';

DELETE FROM specialties WHERE name = 'Mobilidade';

-- 5.5 Objetivos que estavam no catalogo de servicos: os vinculos viram objetivos
INSERT INTO professional_objectives (professional_id, objective_id)
SELECT ps.professional_id, o.id
FROM professional_specialties ps
JOIN specialties s ON s.id = ps.specialty_id AND s.name = 'Hipertrofia'
CROSS JOIN objectives o
WHERE o.name = 'Ganho de massa muscular'
ON CONFLICT DO NOTHING;

DELETE FROM professional_specialties ps
USING specialties s
WHERE ps.specialty_id = s.id AND s.name = 'Hipertrofia';

DELETE FROM specialties WHERE name = 'Hipertrofia';

INSERT INTO professional_objectives (professional_id, objective_id)
SELECT ps.professional_id, o.id
FROM professional_specialties ps
JOIN specialties s ON s.id = ps.specialty_id AND s.name = 'Emagrecimento'
CROSS JOIN objectives o
WHERE o.name = 'Emagrecimento'
ON CONFLICT DO NOTHING;

DELETE FROM professional_specialties ps
USING specialties s
WHERE ps.specialty_id = s.id AND s.name = 'Emagrecimento';

DELETE FROM specialties WHERE name = 'Emagrecimento';

INSERT INTO professional_objectives (professional_id, objective_id)
SELECT ps.professional_id, o.id
FROM professional_specialties ps
JOIN specialties s ON s.id = ps.specialty_id AND s.name = 'Condicionamento Físico'
CROSS JOIN objectives o
WHERE o.name = 'Condicionamento físico'
ON CONFLICT DO NOTHING;

DELETE FROM professional_specialties ps
USING specialties s
WHERE ps.specialty_id = s.id AND s.name = 'Condicionamento Físico';

DELETE FROM specialties WHERE name = 'Condicionamento Físico';
