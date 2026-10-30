-- ============================================================================
-- Migracao 0004: Dados de exemplo (seed) - profissionais da vitrine
-- Conteudo migrado do antigo mock do frontend (frontend/lib/mock.ts).
-- Idempotente: pode ser executada com seguranca mais de uma vez.
-- Senha de todos os usuarios seed: "senha12345"
-- ============================================================================

-- 1) Especialidades usadas pelo seed (complementa as ja existentes em init.sql)
INSERT INTO specialties (name) VALUES
    ('Hipertrofia'),
    ('Funcional'),
    ('Crossfit'),
    ('Natação'),
    ('Emagrecimento'),
    ('Mobilidade')
ON CONFLICT (name) DO NOTHING;

-- 2) Usuarios profissionais de exemplo
INSERT INTO users (email, password_hash, role, status) VALUES
    ('bruno.almeida@fitconecta.dev',    '$2a$10$oBYTAKptcYe9joffx8/OGeOTmCiukHtBaUOIKkrLJOyK/3L1A8ImG', 'professional', 'active'),
    ('camila.rocha@fitconecta.dev',     '$2a$10$oBYTAKptcYe9joffx8/OGeOTmCiukHtBaUOIKkrLJOyK/3L1A8ImG', 'professional', 'active'),
    ('diego.martins@fitconecta.dev',    '$2a$10$oBYTAKptcYe9joffx8/OGeOTmCiukHtBaUOIKkrLJOyK/3L1A8ImG', 'professional', 'active'),
    ('fernanda.lima@fitconecta.dev',    '$2a$10$oBYTAKptcYe9joffx8/OGeOTmCiukHtBaUOIKkrLJOyK/3L1A8ImG', 'professional', 'active'),
    ('gabriel.souza@fitconecta.dev',    '$2a$10$oBYTAKptcYe9joffx8/OGeOTmCiukHtBaUOIKkrLJOyK/3L1A8ImG', 'professional', 'active'),
    ('helena.castro@fitconecta.dev',    '$2a$10$oBYTAKptcYe9joffx8/OGeOTmCiukHtBaUOIKkrLJOyK/3L1A8ImG', 'professional', 'active'),
    ('igor.nascimento@fitconecta.dev',  '$2a$10$oBYTAKptcYe9joffx8/OGeOTmCiukHtBaUOIKkrLJOyK/3L1A8ImG', 'professional', 'active'),
    ('juliana.prado@fitconecta.dev',    '$2a$10$oBYTAKptcYe9joffx8/OGeOTmCiukHtBaUOIKkrLJOyK/3L1A8ImG', 'professional', 'active')
ON CONFLICT (email) DO NOTHING;

-- 3) Perfis profissionais (nome, bio, cidade, UF, preco, nota, avaliacoes)
INSERT INTO professional_profiles
    (user_id, display_name, bio, city, state, session_price_cents, rating_avg, reviews_count, approved_at)
SELECT u.id, v.display_name, v.bio, v.city, v.state, v.session_price_cents, v.rating_avg, v.reviews_count, NOW()
FROM (VALUES
    ('bruno.almeida@fitconecta.dev',   'Bruno Almeida',   'Personal com 8 anos de experiência em hipertrofia e ganho de força para iniciantes e avançados.', 'Belo Horizonte', 'MG', 12000, 4.9::numeric, 128),
    ('camila.rocha@fitconecta.dev',    'Camila Rocha',    'Treinos funcionais e circuitos para emagrecimento, com acompanhamento de evolução e nutrição básica.', 'São Paulo', 'SP', 15000, 5.0::numeric, 214),
    ('diego.martins@fitconecta.dev',   'Diego Martins',   'Assessoria de corrida de rua e mobilidade. Do 5km à meia maratona, com planilha individual.', 'Curitiba', 'PR', 10000, 4.8::numeric, 96),
    ('fernanda.lima@fitconecta.dev',   'Fernanda Lima',   'Instrutora de Pilates com foco em postura, core e reabilitação de dores lombares.', 'Belo Horizonte', 'MG', 14000, 4.9::numeric, 173),
    ('gabriel.souza@fitconecta.dev',   'Gabriel Souza',   'Coach de Crossfit e condicionamento físico. Treinos intensos e escaláveis para todos os níveis.', 'Rio de Janeiro', 'RJ', 16000, 4.7::numeric, 88),
    ('helena.castro@fitconecta.dev',   'Helena Castro',   'Professora de yoga e alongamento. Aulas para reduzir o estresse e melhorar a flexibilidade.', 'Florianópolis', 'SC', 11000, 5.0::numeric, 142),
    ('igor.nascimento@fitconecta.dev', 'Igor Nascimento', 'Técnico de natação para adultos, do nado recreativo ao aprimoramento de técnica.', 'São Paulo', 'SP', 13000, 4.8::numeric, 65),
    ('juliana.prado@fitconecta.dev',   'Juliana Prado',   'Personal para mulheres que buscam hipertrofia e definição com treino periodizado.', 'Curitiba', 'PR', 13500, 4.9::numeric, 201)
) AS v(email, display_name, bio, city, state, session_price_cents, rating_avg, reviews_count)
JOIN users u ON u.email = v.email
ON CONFLICT (user_id) DO NOTHING;

-- 4) Especialidades de cada profissional
INSERT INTO professional_specialties (professional_id, specialty_id)
SELECT p.id, s.id
FROM (VALUES
    ('bruno.almeida@fitconecta.dev',   'Musculação'),
    ('bruno.almeida@fitconecta.dev',   'Hipertrofia'),
    ('camila.rocha@fitconecta.dev',    'Funcional'),
    ('camila.rocha@fitconecta.dev',    'Emagrecimento'),
    ('diego.martins@fitconecta.dev',   'Corrida'),
    ('diego.martins@fitconecta.dev',   'Mobilidade'),
    ('fernanda.lima@fitconecta.dev',   'Pilates'),
    ('fernanda.lima@fitconecta.dev',   'Mobilidade'),
    ('gabriel.souza@fitconecta.dev',   'Crossfit'),
    ('gabriel.souza@fitconecta.dev',   'Funcional'),
    ('helena.castro@fitconecta.dev',   'Yoga'),
    ('helena.castro@fitconecta.dev',   'Mobilidade'),
    ('igor.nascimento@fitconecta.dev', 'Natação'),
    ('igor.nascimento@fitconecta.dev', 'Funcional'),
    ('juliana.prado@fitconecta.dev',   'Hipertrofia'),
    ('juliana.prado@fitconecta.dev',   'Musculação'),
    ('juliana.prado@fitconecta.dev',   'Emagrecimento')
) AS v(email, specialty_name)
JOIN users u ON u.email = v.email
JOIN professional_profiles p ON p.user_id = u.id
JOIN specialties s ON s.name = v.specialty_name
ON CONFLICT DO NOTHING;

-- 5) Disponibilidade semanal (gera slots). Cada horario vira uma regra de 1h
--    de segunda a sabado, reproduzindo os horarios do mock.
INSERT INTO availability_rules (professional_id, weekday, start_time, end_time, slot_minutes)
SELECT p.id, w.weekday,
       make_time(t.hour, 0, 0),
       make_time(t.hour + 1, 0, 0),
       60
FROM (VALUES
    ('bruno.almeida@fitconecta.dev',   7),
    ('bruno.almeida@fitconecta.dev',   18),
    ('camila.rocha@fitconecta.dev',    6),
    ('camila.rocha@fitconecta.dev',    19),
    ('diego.martins@fitconecta.dev',   6),
    ('diego.martins@fitconecta.dev',   20),
    ('fernanda.lima@fitconecta.dev',   9),
    ('fernanda.lima@fitconecta.dev',   17),
    ('gabriel.souza@fitconecta.dev',   7),
    ('gabriel.souza@fitconecta.dev',   19),
    ('helena.castro@fitconecta.dev',   8),
    ('helena.castro@fitconecta.dev',   18),
    ('igor.nascimento@fitconecta.dev', 6),
    ('igor.nascimento@fitconecta.dev', 18),
    ('juliana.prado@fitconecta.dev',   7),
    ('juliana.prado@fitconecta.dev',   20)
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
