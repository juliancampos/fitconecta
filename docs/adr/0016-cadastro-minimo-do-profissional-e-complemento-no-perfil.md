# ADR 0016: Cadastro Mínimo do Profissional e Complemento no Perfil

* **Status:** Aceito
* **Data:** 2026-11-11
* **Decisores:** Time de Arquitetura e Engenharia FitConecta+
* **Contexto Técnico:** Next.js (Web), Go (Core API), PostgreSQL (`professional_profiles`)
* **Supersede parcialmente:** [ADR 0015](0015-sessao-iniciada-no-cadastro.md) (apenas o destino do redirecionamento do profissional)

---

## 1. Contexto e Problema

O formulário de cadastro do profissional (`frontend/app/auth/register/page.tsx`) pede hoje o perfil completo de uma vez: ramo + registro do conselho, grid Cidade/UF, especialidades (chips), bairros (`NeighborhoodPicker`), valor da sessão e bio. É atrito demais no momento em que o usuário só quer criar a conta.

O backend **já aceita o cadastro mínimo**: `Register` (`backend/core-api/internal/auth/service.go`) valida apenas e-mail, senha, nome, role, ramo e registro do conselho — nada de cidade, especialidades, bairros, preço ou bio. No INSERT (`backend/core-api/internal/auth/repository.go`), `session_price_cents <= 0` recebe o default `10000` (R$ 100, comentário existente), cidade/UF vazias passam nas colunas `VARCHAR NOT NULL` (string vazia ≠ NULL) e especialidades/bairros são loops opcionais. Portanto **nenhuma mudança de backend é necessária no registro**.

Duas consequências precisam de tratamento:

1. **Vitrine:** a query `List` (`backend/core-api/internal/professionals/repository.go`, linhas 81–90) não tem gate de completude — o profissional recém-cadastrado apareceria com "cidade, UF" vazios e preço default.
2. **Perfil:** `UpdateMyProfile` (`backend/core-api/internal/professionals/service.go`, linhas 77–84) valida só `DisplayName` e preço > 0; sem exigir cidade/UF, o profissional pode permanecer invisível para sempre sem entender por quê. O `ProfileView` (`frontend/components/professional/ProfileView.tsx`) engole erros do PUT com mensagem genérica no `catch`.

O admin **não tem ação de aprovação** (o painel só cancela agendamento e revalida CREF; `approved_at` só é setado nos seeds mock), então o gate não pode depender de aprovação. O redirecionamento pós-cadastro do ADR 0015 (`frontend/app/auth/register/page.tsx`, linha 322) manda o profissional para `/app/professional`; com o cadastro reduzido, essa home deixaria o cadastro pela metade.

### Requisito

A tela de cadastro do profissional deve conter apenas os campos até o **Registro CREF/CRN** (ramo + registro do conselho). O restante (cidade/UF, especialidades, bairros, valor da sessão, bio) é preenchido no **perfil**. Após o cadastro, a primeira tela do profissional é o **seu perfil**, para continuar o cadastro. O fluxo do cliente não muda.

---

## 2. Decisão

1. **Formulário de cadastro profissional reduzido até o Registro CREF/CRN** (ramo + registro, com blur-check existente). Remover: grid Cidade/UF, chips de especialidades, `NeighborhoodPicker`, input Valor da sessão, textarea Bio — e os estados/carregamentos órfãos (`specialtyIds`, `neighborhood`, `sessionPrice`, `bio` no FormState/validate/payload; `listSpecialties`/lista de bairros no useEffect; imports `NeighborhoodPicker`, tipos `Neighborhood`/`Specialty`). Adicionar uma linha de dica após o registro do conselho: os demais dados são completados no perfil.
2. **Redirect pós-cadastro do profissional para `/app/professional/perfil`** (continuação do cadastro no próprio perfil); cliente inalterado (`/app/client`). Supersede **parcialmente** o ADR 0015 (apenas o alvo do profissional, não o mecanismo de sessão).
3. **Sem mudança no backend do registro** — ele já aceita o cadastro mínimo; o preço default R$ 100 permanece para quem não preencheu.
4. **Gate de vitrine:** `List` (`professionals/repository.go`) passa a filtrar `AND p.city <> '' AND p.state <> ''` — profissionais incompletos não aparecem até completar o perfil. Mocks e profissionais existentes têm cidade/UF (nenhuma regressão).
5. **Gate de completude no perfil:** `UpdateMyProfile` (`professionals/service.go`) passa a exigir cidade e UF (erros "informe a cidade" / "informe a UF", mesmo estilo dos existentes de nome/preço); `ProfileView` usa `apiErrorMessage` no `catch` do PUT para exibir a mensagem real do backend.
6. **`openapi.yaml`:** `city`, `state`, `specialty_ids`, `neighborhood_ids`, `session_price_cents` e `bio` no `POST /auth/register` passam a ser documentados como opcionais/completáveis no perfil (apenas descriptions, sem mudança de schema).

Fora do escopo: wizard multi-etapa, gate por especialidades, aprovação de admin na vitrine (ver alternativas).

### 2.1 Alternativas consideradas

| Alternativa | Prós | Contras | Decisão |
|---|---|---|---|
| **A. Manter todos os campos no cadastro** | Cadastro completo de uma vez | É exatamente o atrito que o usuário pediu para remover | Rejeitada |
| **B. Wizard multi-etapa no cadastro** | Progressão guiada | Duplicaria a tela de perfil, que já edita todos esses campos | Rejeitada |
| **C. Sem gate na vitrine** | Zero mudança de código | Profissionais incompletos apareceriam com cidade vazia e preço default | Rejeitada |
| **D. Publicar na vitrine só após aprovação do admin (`approved_at`)** | Curadoria explícita | O painel admin não tem ação de aprovação; criaria pipeline novo fora do escopo | Rejeitada |
| **E. Gate por especialidades (`EXISTS professional_specialties`)** | Gate mais estrito | Mais complexo; cidade/UF já basta para o objetivo da vitrine | Adiada |

---

## 3. Contrato e Erros

| Situação | Status HTTP | Comportamento |
|---|---|---|
| `POST /auth/register` (professional) sem cidade/especialidades/bairros/preço/bio | 201 | Aceito como hoje: continua exigindo ramo + registro do conselho; demais campos opcionais (retrocompatível — quem os envia continua funcionando) |
| `PUT /professionals/me/profile` com cidade vazia | 400 | Novo: `informe a cidade` (junto aos existentes de nome/preço) |
| `PUT /professionals/me/profile` com UF vazia | 400 | Novo: `informe a UF` |
| `GET /professionals` (vitrine) | 200 | Não retorna perfis sem cidade/UF — o gate é server-side na query `List` |
| `GET /professionals/{id}` (link direto para perfil incompleto) | 200 | Continua funcionando; a vitrine é a porta de entrada |

Nenhuma mudança de schema: `city`/`state` continuam `VARCHAR NOT NULL` (string vazia), sem migration.

---

## 4. Implementação

1. **`frontend/app/auth/register/page.tsx`** — remover grid Cidade/UF, chips de especialidades, `NeighborhoodPicker`, input Valor da sessão, textarea Bio e todos os estados/carregamentos órfãos (FormState, `validate`, payload, `useEffect` de especialidades/bairros, imports/tipos órfãos); manter ramo + Registro CREF/CRN com blur-check; adicionar dica de complemento no perfil; trocar o redirect do profissional para `/app/professional/perfil` (cliente segue `/app/client`).
2. **`backend/core-api/internal/professionals/repository.go`** — gate `AND p.city <> '' AND p.state <> ''` na query `List` (linhas 81–90).
3. **`backend/core-api/internal/professionals/service.go`** — em `UpdateMyProfile` (linhas 77–84), exigir cidade e UF (`TrimSpace` + erros "informe a cidade"/"informe a UF", mesmo estilo das validações existentes).
4. **`backend/core-api/internal/professionals/service_test.go`** — helper `newUpdateReq` (linha 86) passa a setar `City: "Belo Horizonte"`, `State: "MG"`; novo teste do erro com cidade vazia.
5. **`frontend/components/professional/ProfileView.tsx`** — usar `apiErrorMessage` no `catch` do PUT do perfil (exibir a mensagem real do backend).
6. **`backend/core-api/openapi.yaml`** — descriptions marcando `city`, `state`, `specialty_ids`, `neighborhood_ids`, `session_price_cents`, `bio` do `POST /auth/register` como opcionais/completáveis no perfil.

**Validação:** `go build ./... && go vet ./... && go test ./...` em `backend/core-api/`; `npm run lint && npm run build` em `frontend/` (sem infra de teste no frontend).

---

## 5. Privacidade e Segurança

1. **Gate server-side na vitrine:** perfis incompletos ficam invisíveis no `GET /professionals` — menos exposição de dados parciais; não é apenas esconder no frontend.
2. **Validação de completude no servidor:** cidade/UF exigidas no `PUT /me/profile`, independente do cliente — o mesmo padrão das validações existentes de nome/preço.
3. **Papel continua vindo da resposta da API** no redirect pós-cadastro (fonte de verdade do servidor, conforme ADR 0015); nada muda no transporte da sessão (cookie HttpOnly).
4. **`apiErrorMessage` no `ProfileView`:** exibe a mensagem do backend no padrão do projeto, sem vazar detalhes internos.
5. Nenhum dado pessoal novo é coletado; campos são apenas movidos do cadastro para o perfil.

---

## 6. Riscos e Dívidas

- **Profissional incompleto invisível na vitrine até preencher cidade/UF** — intencional; o erro do perfil ("informe a cidade") guia o preenchimento.
- **Preço default R$ 100 para quem não preencheu** — comportamento pré-existente; ajustável no perfil.
- **Link direto para perfil incompleto (`GET /professionals/{id}`) ainda funciona** — aceitável; a vitrine é a porta de entrada.
- **Consumidores da API que enviam cidade/especialidades no registro continuam funcionando** — campos ficam apenas documentados como opcionais, não ignorados.
- **Sem teste automatizado no frontend:** coberto por lint + build; o fluxo de redirect é o mesmo do ADR 0015.
- Alternativa E (gate por especialidades) fica registrada como dívida admissível caso o produto peça vitrine mais estrita.

---

## 7. Referências

- `frontend/app/auth/register/page.tsx` — formulário atual (campos a remover, redirect na linha 322).
- `frontend/components/professional/ProfileView.tsx` — `catch` genérico do PUT do perfil.
- `backend/core-api/internal/auth/service.go` / `backend/core-api/internal/auth/repository.go` — `Register` já aceita cadastro mínimo (default de preço, city/state vazios).
- `backend/core-api/internal/professionals/service.go` — `UpdateMyProfile` (validações atuais nas linhas 77–84).
- `backend/core-api/internal/professionals/repository.go` — query `List` sem gate (linhas 81–90).
- `backend/core-api/internal/professionals/service_test.go` — helper `newUpdateReq` sem City/State.
- `backend/core-api/openapi.yaml` — contrato de `POST /auth/register` e `PUT /professionals/me/profile`.
- [ADR 0013](0013-validacao-cref-no-cadastro.md) — blur-check do CREF mantido no cadastro reduzido.
- [ADR 0014](0014-cadastro-de-nutricionistas.md) — ramo CREF/CRN, último campo do cadastro.
- [ADR 0015](0015-sessao-iniciada-no-cadastro.md) — sessão iniciada no cadastro (mecanismo preservado; alvo do profissional alterado).
