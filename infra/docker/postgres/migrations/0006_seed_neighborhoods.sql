-- ============================================================================
-- Migracao 0006: Seed de bairros e associacao aos profissionais de exemplo
-- (ADR 0004). Idempotente: pode ser executada mais de uma vez.
-- ============================================================================

-- 1) Catalogo de bairros das cidades do piloto
INSERT INTO neighborhoods (state, city, name, slug) VALUES
    -- Belo Horizonte / MG
    ('MG', 'Belo Horizonte', 'Savassi',        'savassi'),
    ('MG', 'Belo Horizonte', 'Lourdes',        'lourdes'),
    ('MG', 'Belo Horizonte', 'Pampulha',       'pampulha'),
    ('MG', 'Belo Horizonte', 'Funcionários',   'funcionarios'),
    ('MG', 'Belo Horizonte', 'Centro',         'centro'),
    ('MG', 'Belo Horizonte', 'Buritis',        'buritis'),
    -- São Paulo / SP
    ('SP', 'São Paulo', 'Pinheiros',    'pinheiros'),
    ('SP', 'São Paulo', 'Vila Mariana', 'vila-mariana'),
    ('SP', 'São Paulo', 'Itaim Bibi',   'itaim-bibi'),
    ('SP', 'São Paulo', 'Moema',        'moema'),
    ('SP', 'São Paulo', 'Perdizes',     'perdizes'),
    ('SP', 'São Paulo', 'Jardins',      'jardins'),
    -- Curitiba / PR
    ('PR', 'Curitiba', 'Batel',            'batel'),
    ('PR', 'Curitiba', 'Água Verde',       'agua-verde'),
    ('PR', 'Curitiba', 'Centro Cívico',    'centro-civico'),
    ('PR', 'Curitiba', 'Bigorrilho',       'bigorrilho'),
    ('PR', 'Curitiba', 'Jardim Botânico',  'jardim-botanico'),
    -- Rio de Janeiro / RJ
    ('RJ', 'Rio de Janeiro', 'Copacabana',      'copacabana'),
    ('RJ', 'Rio de Janeiro', 'Ipanema',         'ipanema'),
    ('RJ', 'Rio de Janeiro', 'Botafogo',        'botafogo'),
    ('RJ', 'Rio de Janeiro', 'Tijuca',          'tijuca'),
    ('RJ', 'Rio de Janeiro', 'Barra da Tijuca', 'barra-da-tijuca'),
    -- Florianópolis / SC
    ('SC', 'Florianópolis', 'Centro',                'centro'),
    ('SC', 'Florianópolis', 'Lagoa da Conceição',    'lagoa-da-conceicao'),
    ('SC', 'Florianópolis', 'Trindade',              'trindade'),
    ('SC', 'Florianópolis', 'Campeche',              'campeche'),
    ('SC', 'Florianópolis', 'Jurerê',                'jurere')
ON CONFLICT (state, city, name) DO NOTHING;

-- 2) Bairros de atuacao dos profissionais de exemplo
INSERT INTO professional_neighborhoods (professional_id, neighborhood_id)
SELECT p.id, n.id
FROM (VALUES
    ('bruno.almeida@fitconecta.dev',   'MG', 'Belo Horizonte', 'Savassi'),
    ('bruno.almeida@fitconecta.dev',   'MG', 'Belo Horizonte', 'Lourdes'),
    ('bruno.almeida@fitconecta.dev',   'MG', 'Belo Horizonte', 'Funcionários'),
    ('fernanda.lima@fitconecta.dev',   'MG', 'Belo Horizonte', 'Pampulha'),
    ('fernanda.lima@fitconecta.dev',   'MG', 'Belo Horizonte', 'Buritis'),
    ('fernanda.lima@fitconecta.dev',   'MG', 'Belo Horizonte', 'Centro'),
    ('camila.rocha@fitconecta.dev',    'SP', 'São Paulo',      'Pinheiros'),
    ('camila.rocha@fitconecta.dev',    'SP', 'São Paulo',      'Vila Mariana'),
    ('camila.rocha@fitconecta.dev',    'SP', 'São Paulo',      'Perdizes'),
    ('igor.nascimento@fitconecta.dev', 'SP', 'São Paulo',      'Itaim Bibi'),
    ('igor.nascimento@fitconecta.dev', 'SP', 'São Paulo',      'Moema'),
    ('igor.nascimento@fitconecta.dev', 'SP', 'São Paulo',      'Jardins'),
    ('diego.martins@fitconecta.dev',   'PR', 'Curitiba',       'Batel'),
    ('diego.martins@fitconecta.dev',   'PR', 'Curitiba',       'Água Verde'),
    ('diego.martins@fitconecta.dev',   'PR', 'Curitiba',       'Bigorrilho'),
    ('juliana.prado@fitconecta.dev',   'PR', 'Curitiba',       'Centro Cívico'),
    ('juliana.prado@fitconecta.dev',   'PR', 'Curitiba',       'Jardim Botânico'),
    ('juliana.prado@fitconecta.dev',   'PR', 'Curitiba',       'Batel'),
    ('gabriel.souza@fitconecta.dev',   'RJ', 'Rio de Janeiro', 'Copacabana'),
    ('gabriel.souza@fitconecta.dev',   'RJ', 'Rio de Janeiro', 'Ipanema'),
    ('gabriel.souza@fitconecta.dev',   'RJ', 'Rio de Janeiro', 'Botafogo'),
    ('helena.castro@fitconecta.dev',   'SC', 'Florianópolis',  'Lagoa da Conceição'),
    ('helena.castro@fitconecta.dev',   'SC', 'Florianópolis',  'Campeche'),
    ('helena.castro@fitconecta.dev',   'SC', 'Florianópolis',  'Trindade')
) AS v(email, state, city, neighborhood_name)
JOIN users u ON u.email = v.email
JOIN professional_profiles p ON p.user_id = u.id
JOIN neighborhoods n ON n.state = v.state AND n.city = v.city AND n.name = v.neighborhood_name
ON CONFLICT DO NOTHING;
