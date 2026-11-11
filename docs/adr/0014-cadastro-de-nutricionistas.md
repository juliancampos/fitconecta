# ADR 0014: Ramo do Profissional e Cadastro de Nutricionistas (CFN/CRN)

* **Status:** Aceito
* **Data:** 2026-11-10
* **Decisores:** Time de Arquitetura e Engenharia FitConecta+
* **Contexto Técnico:** Go (Core API), Next.js (Web), Consulta Nacional de Nutricionistas do CFN (CNN)

---

## 1. Contexto e Problema

A plataforma cadastra apenas profissionais de educação física: o cadastro exige o **CREF** (ADR 0013), o catálogo `specialties` só tem modalidades EF (Musculação, Treinamento Funcional, Pilates...) e a vitrine (`frontend/components/HomeView.tsx`) carrega todos os profissionais de uma vez (`listProfessionals()` sem filtros), filtrando no cliente.

O produto agora aceita **nutricionistas**, sem remover os profissionais de educação física. O CFN mantém a **Consulta Nacional de Nutricionistas (CNN)** em `https://cnn.cfn.org.br`, app AngularJS que consulta um endpoint JSON:

- `POST https://cnn.cfn.org.br/application/front-resource/get`
- Body: `{"comando":"get-nutricionista","options":{"nome":"","crn":<regional 1-11 ou null>,"registro":"<numero>","estado":"","geral":false,"tecnico":null}}`
- Resposta 200: `{"success":true,"data":[{"nome":"JULIA VIANA RESENDE","registro":"39104","crn":9,"data_cadastro":"10-11-2026","situacao":"ATIVO","tipo_registro":"NUTRICIONISTA DEFINITIVO"}, ...]}`

Sem CAPTCHA (o desafio de soma está comentado no HTML) e sem viewstate — POST JSON puro, bem mais robusto que o scraping do CREF6 (ADR 0013). Está atrás de Cloudflare, mas responde a HTTP simples (testado). Cobertura **nacional** (os 11 CRNs regionais), dados atualizados aproximadamente por mês. Dois detalhes afetam a validação:

- Linhas com `tipo_registro` = `AVULSO-PF` são produtores rurais, não nutricionistas — devem ser ignoradas; só linhas com `tipo_registro` começando com `NUTRICIONISTA` contam.
- O mesmo `registro` pode existir em mais de um regional (ex.: `39104` existe no CRN9 e no CRN3); a busca aceita o regional como prefixo do número digitado ("CRN9 39104", "9-39104") ou apenas o número ("39104").

### Requisitos

- Campo **ramo** no cadastro do profissional: Educação Física ou Nutrição.
- Nutrição → verificar o **CFN/CRN** (número de registro no conselho de nutrição).
- Educação Física → informar o **CREF** (comportamento atual, ADR 0013).
- Tela home (vitrine): filtro por tipo de profissional + ícone em cada card identificando o ramo.

---

## 2. Decisão

**O ramo do profissional é uma coluna do perfil, não um papel de usuário.** Todo o fluxo que hoje switcha em `role=professional` (agendamentos, pagamentos, admin, middleware) continua intacto; o ramo só decide qual registro do conselho o profissional informa e qual validação roda.

1. **Coluna `branch`.** `professional_profiles.branch VARCHAR(20) NOT NULL DEFAULT 'physical_education'` com CHECK `IN ('physical_education','nutrition')`. Registros existentes ficam como Educação Física (correto — hoje a base só tem EFs). A API expõe `branch` no perfil.
2. **Colunas CRN no padrão do CREF.** `crn_number VARCHAR(30)` (nullable) e `crn_verified_at TIMESTAMPTZ`, espelhando exatamente o padrão do ADR 0013: selo público no card, NULL = "em análise".
3. **Ramo no cadastro.** `POST /auth/register` aceita `branch` quando `role=professional`. Ausente ou vazio → assume `physical_education` (retrocompatibilidade para clientes existentes que só enviam `cref_number`). Valor fora do conjunto → `400`. `branch=nutrition` → `crn_number` obrigatório (`400` "informe o numero do CRN") e `cref_number` ignorado (gravado NULL); `branch=physical_education` → `cref_number` obrigatório (`400` atual). O ramo **não é editável** depois do cadastro — não entra no `UpdateProfileRequest`.
4. **Pacote `internal/crn`.** `Client.Validate(ctx, number)` faz um único POST JSON à CNN (timeout 15 s), compara `registro` por dígitos exatos sem zeros à esquerda e aceita prefixo regional opcional digitado junto ("CRN9 39104" ou "9-39104"). Ignora linhas cujo `tipo_registro` não começa com `NUTRICIONISTA` (ex.: `AVULSO-PF`) e considera válido apenas `situacao` = `ATIVO`. Erros espelham o cref: `ErrNotFound`, `ErrInactive`, `ErrUnavailable`. A validação é **informativa** — nunca bloqueia o cadastro (mesma filosofia do ADR 0013).
5. **Endpoint público de consulta.** `GET /api/v1/professionals/crn/validate?number=...` responde **sempre 200** quando o número é informado, com `{valid, reason, message, number?, name?}` (`reason`: `not_found`, `inactive`, `unavailable`), espelhando `/cref/validate`.
6. **Verificação em segundo plano** no cadastro e na edição: a troca de número limpa `crn_verified_at` e revalida; `SetCRNVerified` espelha `SetCREFVerified` — grava apenas se o número não mudou no caminho.
7. **Edição de perfil.** `PUT /professionals/me/profile` ganha `crn_number`; o profissional edita o registro do próprio ramo (CREF para EF, CRN para nutricionista).
8. **Filtro por ramo na home é client-side** (igual ao filtro de especialidade hoje — a página já carrega a lista completa). `FilterBar` ganha select "Tipo de profissional" (Todos / Educação Física / Nutrição). Sem query param novo no backend.
9. **Card da vitrine.** Pílula com ícone + rótulo do ramo ao lado do nome (halteres para Educação Física, maçã para Nutrição) e selo "CRN verificado" / "CRN em análise" para nutricionistas, espelhando o selo do CREF.
10. **Migração 0017**, idempotente (`infra/docker/postgres/migrations/0017_professional_branch.sql`): colunas acima + constraint CHECK (via bloco `DO`, pois `ADD CONSTRAINT` não tem `IF NOT EXISTS`), especialidades de nutrição e dois nutricionistas de exemplo. Aplicada via `scripts/apply-sql.sh` sobre `init.sql` + migrations, como a 0016.
11. **Contrato documentado.** `openapi.yaml` atualizado com `branch`/`crn_number` no cadastro e no perfil e com a rota `/professionals/crn/validate`.

### 2.1 Alternativas consideradas

| Alternativa | Prós | Contras | Decisão |
|---|---|---|---|
| **A. Novo papel de usuário "nutritionist"** | Ramo explícito na autenticação | Duplica todos os fluxos que switcham em `role=professional` (agendamentos, pagamentos, admin, middleware) | Rejeitada |
| **B. Campo genérico `council_number` substituindo `cref_number`** | Uma coluna para qualquer conselho futuro | Quebra dados e API existentes; duas colunas nullable mantêm compat | Rejeitada |
| **C. Scraping dos portais regionais dos CRNs (ex.: crn9.org.br)** | Fonte regional oficial | A CNN do CFN é nacional, JSON puro, sem viewstate; um portal por regional multiplicaria manutenção | Rejeitada |
| **D. Bloquear cadastro quando o CRN não validar (fail-closed)** | Impede CRN inválido | Queda da CNN impede novos nutricionistas; mesma análise do ADR 0013 (portal fora ≠ sem nutricionistas) | Rejeitada |
| **E. Ramo como tabela de catálogo** | Extensível a novos ramos | Dois valores fixos; CHECK basta; tabela só criaria outra fonte de verdade para divergir | Rejeitada |
| **F. Validar CRN só na aprovação manual do admin** | Robusto e auditável | Não é automática; fila de aprovação e atraso no cadastro | Rejeitada |
| **G. Filtro por ramo no backend (query param)** | Escala para vitrine paginada | A home já carrega a lista completa e filtra no cliente (igual especialidade); YAGNI | Adiada |

---

## 3. Contrato e Erros

`POST /auth/register` (role `professional`):

| Situação | Status | Resposta |
|---|---|---|
| `branch` ausente ou vazio | 201 | assumido `physical_education` (comportamento atual) |
| `branch` com valor fora do conjunto | 400 | `ramo deve ser 'physical_education' ou 'nutrition'` |
| `branch=nutrition` sem `crn_number` | 400 | `informe o numero do CRN` |
| `branch=physical_education` sem `cref_number` | 400 | `informe o numero do CREF` (erro atual) |
| `branch=nutrition` com `cref_number` preenchido | 201 | `cref_number` ignorado, gravado NULL |
| CRN inválido / inativo / CNN fora | 201 | cadastro conclui normalmente |
| Demais dados inválidos | 400 | mensagem específica, sem o prefixo interno |

`PUT /professionals/me/profile`: grava o número do próprio ramo (`cref_number` ou `crn_number`) sem consultar a CNN; a troca de número limpa o selo e revalida em segundo plano. `branch` não é aceito nesta rota (ramo imutável após o cadastro).

`GET /professionals/crn/validate?number=...`:

| Situação | `valid` | `reason` | `message` |
|---|---|---|---|
| Encontrado e `ATIVO` | `true` | — | `CRN ativo na CNN do CFN.` (devolve `number` canônico, ex.: `CRN9 39104`, e `name`) |
| Não encontrado | `false` | `not_found` | mensagem da CNN |
| Situação diferente de `ATIVO` | `false` | `inactive` | mensagem da CNN |
| CNN fora do ar / timeout de 15 s | `false` | `unavailable` | mensagem de indisponibilidade |
| `number` vazio | 400 | — | `informe o numero do CRN` |

---

## 4. Implementação

- `infra/docker/postgres/migrations/0017_professional_branch.sql`: colunas, CHECK, especialidades de nutrição e 2 nutricionistas mock (DDL completo abaixo).
- `backend/core-api/internal/crn/client.go`: `Validate`, parse do JSON da CNN, comparação de `registro`, filtro de `tipo_registro` e `situacao`.
- `backend/core-api/internal/crn/client_test.go`: CNN falsa em `httptest` cobrindo número exato entre resultados, prefixo regional ("CRN9 39104" e "9-39104"), zeros à esquerda, inexistente, inativo, linha `AVULSO-PF` ignorada, portal fora e número vazio sem requisição.
- `backend/core-api/internal/auth/model.go`: `Branch`, `CRNNumber` em `RegisterRequest`.
- `backend/core-api/internal/auth/service.go`: validação por ramo (default EF, `nutrition` exige CRN, `physical_education` exige CREF) e verificação CRN em segundo plano após criar a conta.
- `backend/core-api/internal/auth/repository.go`: insert com `branch`/`crn_number` (`cref_number` NULL para nutrição).
- `backend/core-api/internal/auth/service_test.go`: nutrição sem CRN → erro; nutrição com CRN → ok; EF sem CREF → erro; branch ausente → default EF.
- `backend/core-api/internal/professionals/model.go`: `Branch`, `CRNNumber`, `CRNVerifiedAt` no perfil; `CRNNumber` no `UpdateProfileRequest`.
- `backend/core-api/internal/professionals/service.go`: `ValidateCRN` (delega ao cliente) e `UpdateMyProfile` por ramo — preserva o status quando o número não muda e revalida em segundo plano quando muda.
- `backend/core-api/internal/professionals/handler.go`: rota pública `GET /crn/validate` e aviso `{valid, reason, message}`.
- `backend/core-api/internal/professionals/repository.go`: colunas novas nas queries, update com os dois números e seus selos, `SetCRNVerified`.
- `backend/core-api/internal/professionals/service_test.go`.
- `backend/core-api/cmd/api/main.go`: `crn.NewClient()` injetado junto ao cref.
- `backend/core-api/openapi.yaml`: `branch`/`crn_number` no cadastro e no perfil; contrato de `GET /professionals/crn/validate`.
- `frontend/lib/types.ts`: `branch`, `crn_number`, `crn_verified_at`.
- `frontend/app/auth/register/page.tsx`: seletor de ramo + campo CRN (aparece quando ramo = Nutrição) com consulta no `blur` e indicadores verificado/em análise.
- `frontend/components/FilterBar.tsx` + `frontend/components/HomeView.tsx`: select "Tipo de profissional" (Todos / Educação Física / Nutrição) com filtro client-side.
- `frontend/components/ProfessionalCard.tsx`: pílula de ramo (ícone + rótulo) ao lado do nome e selo "CRN verificado/em análise".
- `frontend/components/professional/ProfileView.tsx`: campo de registro conforme o ramo (CREF para EF, CRN para nutricionista).

### 4.1 Migração 0017 — DDL completo

```sql
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
```

**Rollout:** migração apenas aditiva, sem `DROP`/`UPDATE` em dados existentes — os perfis atuais recebem `branch='physical_education'` pelo DEFAULT, que é o valor correto (hoje só existem EFs). Clientes que já enviam `cref_number` sem `branch` continuam funcionando (default EF). Nenhum fluxo do CREF é alterado.

---

## 5. Privacidade e Segurança

1. **Dado público.** A consulta envia apenas o número de registro à CNN (portal público do CFN); nenhum dado pessoal do usuário é enviado. Do resultado, o banco persiste apenas a data da validação.
2. **Três colunas novas.** `branch`, `crn_number` e `crn_verified_at` em `professional_profiles`; nenhuma tabela adicional.
3. **Falha aberta e visível.** CNN fora do ar não impede o cadastro; o usuário vê o aviso e o time pode revisar o CRN depois. Risco aceito, o mesmo do ADR 0013: perfil com CRN não confirmado.
4. **Status público.** `crn_verified_at` é exposto publicamente de propósito (selo do card), sem dado pessoal — só a existência da validação.
5. **Rate limit.** Uma chamada por `blur` do campo; sem cache (mesmo gatilho do ADR 0013).

---

## 6. Riscos e Dívidas

- **Cloudflare na frente da CNN** pode passar a bloquear chamadas server-side; mitigável com cache/user-agent — hoje funciona com HTTP simples.
- **CNN atualizada ~mensalmente:** nutricionista recém-registrado no CRN recebe "não encontrado" — aviso, não bloqueio.
- **Número sem regional pode existir em mais de um CRN** (ex.: 39104 no CRN9 e no CRN3): a validação aceita qualquer linha ATIVO de nutricionista — informativa, sem conferência de titularidade (mesma limitação do CREF).
- **Painel admin ainda mostra só CREF:** nutricionistas aparecem sem registro. Gatilho: quando o admin precisar auditar CRN.
- **Ramo imutável após o cadastro:** erro de ramo exige novo cadastro — simples e raro.
- **Especialidades de nutrição são seed simples:** serviços/preços por ramo (ADR 0011) ficam para quando o produto separar catálogos por ramo.

---

## 7. Referências

- `backend/core-api/internal/crn/client.go` — cliente da CNN (novo).
- `backend/core-api/internal/cref/client.go` — padrão de cliente, selos e erros a espelhar (ADR 0013).
- `backend/core-api/internal/auth/service.go` — exigência de registro por ramo no cadastro.
- `backend/core-api/internal/professionals/handler.go` — rota pública de consulta.
- `backend/core-api/openapi.yaml` — contrato de `POST /auth/register`, `PUT /professionals/me/profile` e `GET /professionals/crn/validate`.
- `frontend/app/auth/register/page.tsx` — seletor de ramo e indicadores de CRN.
- `frontend/components/ProfessionalCard.tsx` — ícone de ramo e selo público no card.
- [ADR 0013](0013-validacao-cref-no-cadastro.md) — validação do CREF, filosofia informativa e padrão de selos.
