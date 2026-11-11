# ADR 0013: Validação Automática do CREF no Cadastro do Profissional

* **Status:** Aceito
* **Data:** 2026-11-03
* **Decisores:** Time de Arquitetura e Engenharia FitConecta+
* **Contexto Técnico:** Go (Core API), Next.js (Web), portal público do CREF6

---

## 1. Contexto e Problema

O cadastro do profissional pede o **registro CREF**, mas nunca confere se ele existe. O campo é texto livre (`professional_profiles.cref_number VARCHAR(30)`) e a tela apenas exige preenchimento.

O CREF6 mantém uma **consulta pública** em `https://portal.cref6.org.br/spw/ConsultaCadastral/TelaConsultaPublicaCompleta.aspx`. Não há API documentada: a página é ASP.NET WebForms com DevExpress, um `UpdatePanel` e postback assíncrono, sem CAPTCHA.

### Requisitos

- Consultar o CREF **automaticamente**, sem etapa manual.
- **Não bloquear** o cadastro quando a validação falhar: número não encontrado, registro inativo ou portal indisponível.
- Sinalizar o problema no formulário com um **ícone de exclamação** próximo ao campo.
- Não mudar o cadastro do cliente.

---

## 2. Decisão

**A validação do CREF é informativa.** O cadastro e a edição do perfil nunca são recusados por causa da consulta ao portal; o resultado é exposto ao frontend, que mostra um aviso ao lado do campo.

1. **Pacote `internal/cref`.** `Client.Validate(ctx, number)` faz duas requisições: `GET` da página (guarda `__VIEWSTATE`, `__VIEWSTATEGENERATOR`, `__EVENTVALIDATION` e o cookie de sessão) e `POST` do formulário com o número, como o navegador faria. O HTML parcial devolvido é lido e a linha do grid é extraída.
2. **Número exato, não prefixo.** A busca do portal é por prefixo: `0123` devolve 20 registros. A comparação é feita sem máscara e sem zeros à esquerda, aceitando apenas a linha idêntica.
3. **Só `ATIVO` é considerado válido.** `INATIVO`, `SUSPENSO` etc. retornam `valid: false, reason: inactive`.
4. **CREF continua obrigatório para profissional.** String vazia é `400`; o cliente não é afetado.
5. **Cadastro e edição não bloqueiam.** `POST /auth/register` e `PUT /professionals/me/profile` apenas gravam o número informado. A consulta externa não participa da decisão de criar/atualizar.
6. **Endpoint público de consulta.** `GET /api/v1/professionals/cref/validate?number=...` responde **sempre 200** quando o número é informado, com `{valid, reason, message}` (`reason`: `not_found`, `inactive`, `unavailable`). Quando válido, devolve também `number` e `name` do portal.
7. **Indicadores no frontend.** O cadastro consulta o endpoint no `blur` do campo CREF e mostra um de dois estados: **verificado** (ícone de check verde + número canônico e nome do portal) quando `valid: true`, ou **em análise** (ícone de relógio âmbar + mensagem) quando `valid: false`. Respostas fora de ordem são descartadas e o indicador some ao editar o campo. O botão "Criar conta" nunca é desabilitado por esse resultado.
8. **Status persistido.** `professional_profiles.cref_verified_at` guarda a data da última validação positiva; `NULL` significa "em análise". A consulta roda **em segundo plano**, depois da resposta do cadastro/edição, e grava apenas se o número ainda for o mesmo que foi consultado.
9. **Selo no card público.** A vitrine de profissionais mostra o estado: check verde "CREF verificado" quando há `cref_verified_at`; relógio âmbar "CREF em análise" quando não há. Perfis criados antes desta entrega ficam "em análise" até uma nova consulta (cadastro novo ou edição do perfil).
10. **Uma chamada compartilhada.** `cref.Validator` é injetado em `auth.Service` e `professionals.Service`; a rota de consulta fica em `internal/professionals`.

### 2.1 Alternativas consideradas

| Alternativa | Prós | Contras | Decisão |
|---|---|---|---|
| **A. Consulta HTTP direta, sem bloquear** | Sem dependência nova, ~1 s, usa a fonte oficial, cadastro resiliente | Depende do HTML do portal; quebra se o layout mudar | **Escolhida** |
| **B. Bloquear o cadastro (fail-closed)** | Impede CREF inválido | Queda do portal impede novos profissionais; números de outros regionais ficam de fora | Rejeitada |
| **C. Aprovação manual pelo admin** | Robusto e auditável | Não é automático; fila de aprovação e atraso no cadastro | Rejeitada |
| **D. Navegador headless (Playwright) no backend** | Tolera JavaScript e mudanças de layout | Centenas de MB de imagem, processo extra, mais lento e mais frágil na infra | Rejeitada |
| **E. Validação assíncrona com `cref_verified_at` + job** | Não bloqueia e registra o estado | Exige job/fila, migration e política de reprocessamento | Adiada |
| **F. Consulta no frontend** | Menos carga no backend | CORS e cookie de terceiro bloqueiam; concentra a regra no cliente | Rejeitada |
| **G. Não validar** | Zero risco operacional | Não avisa o profissional nem o time sobre um CREF inválido | Rejeitada |

---

## 3. Contrato e Erros

`POST /auth/register` (role `professional`):

| Situação | Status | Resposta |
|---|---|---|
| `cref_number` ausente | 400 | `informe o numero do CREF` |
| CREF inválido/inativo/portal fora | 201 | cadastro conclui normalmente |
| Demais dados inválidos | 400 | mensagem específica, sem o prefixo interno |

`PUT /professionals/me/profile`: grava o `cref_number` sem consultar o portal.

`GET /professionals/cref/validate?number=...`:

| Situação | `valid` | `reason` | `message` |
|---|---|---|---|
| Encontrado e `ATIVO` | `true` | — | `CREF ativo no portal do CREF6.` |
| Não encontrado | `false` | `not_found` | mensagem do portal |
| Situação diferente de `ATIVO` | `false` | `inactive` | mensagem do portal |
| Portal fora do ar / timeout de 15 s | `false` | `unavailable` | mensagem de indisponibilidade |
| `number` vazio | 400 | — | `informe o numero do CREF` |

---

## 4. Implementação

- `internal/cref/client.go`: `Validate`, `openPortal`, `search`, `findProfile`, `parseInputs`.
- `internal/cref/client_test.go`: portal falso em `httptest` cobrindo número exato entre resultados de prefixo, zeros à esquerda, máscara `012345-G/MG`, inexistente, inativo, GET/POST fora do ar e número vazio sem requisição.
- `internal/auth/service.go`: exige CREF não vazio e agenda a validação em segundo plano após criar a conta.
- `internal/professionals/service.go`: `ValidateCREF` delega ao cliente; `UpdateMyProfile` preserva o status quando o número não muda e revalida em segundo plano quando muda.
- `internal/professionals/handler.go`: rota pública `GET /cref/validate` e aviso `{valid, reason, message}`.
- `infra/docker/postgres/migrations/0016_professional_cref_verification.sql`: coluna `cref_verified_at`.
- `cmd/api/main.go`: `cref.NewClient()` injetado no serviço de profissionais.
- `frontend/app/auth/register/page.tsx`: consulta no `blur`, check verde de verificado e relógio âmbar de "em análise".
- `frontend/components/ProfessionalCard.tsx`: selo "CREF verificado" / "CREF em análise" no card público.
- `openapi.yaml`: descrição do cadastro/edição e contrato da rota de consulta.

**Cobertura:** apenas o **CREF6 (MG)**. Registros de outros regionais não aparecem no portal consultado e recebem `valid: false, reason: not_found` — o cadastro segue normalmente, com o aviso na tela. Quando o produto atender outro estado, entra um cliente por regional (as URLs e o layout variam).

---

## 5. Privacidade e Segurança

1. **Dado público.** A consulta envia apenas o número do registro a um portal público do conselho; nenhum dado do usuário (nome, e-mail, CPF) é enviado. Do resultado, o banco guarda só a data da validação; nome e categoria do portal não são persistidos.
2. **Uma coluna nova.** `cref_verified_at` em `professional_profiles`; nenhuma tabela adicional. O número continua em `cref_number`.
3. **Falha aberta e visível.** Portal fora do ar não impede o cadastro; o usuário vê o aviso e o time pode revisar o CREF depois. O risco aceito é um perfil com CREF não confirmado.
4. **Status público.** `cref_verified_at` é exposto na vitrine de propósito: é o selo que o cliente vê. Nenhum dado pessoal é incluído — só a existência da validação.
5. **Rate limit.** Cada tentativa custa duas requisições ao portal. Não há cache nem fila; a consulta só acontece no `blur` do campo. Gatilho para cache por número: primeiro pico de consultas ou bloqueio por IP.

---

## 6. Riscos e Dívidas

- **HTML do portal.** Qualquer mudança de layout quebra o parser. Os testes usam HTML fixo; um smoke test manual contra o portal real deve acompanhar o deploy. Com a falha aberta, uma quebra aparece como `unavailable` (aviso), não como erro de cadastro.
- **Somente MG.** Profissionais de outros estados recebem aviso de "não encontrado". Decisão de escopo, não bug.
- **Sem conferência de titularidade.** A validação confirma número e situação, não que o profissional seja o dono do registro. O portal não expõe CPF na busca por número.
- **Aviso só no cadastro.** A tela de edição de perfil (`/profissional/perfil`) ainda não consulta o endpoint. Gatilho para incluir: quando houver tráfego de edição de CREF.
- **Falha silenciosa em segundo plano.** Se a consulta falhar, o perfil permanece "em análise" e o erro não é logado nesta entrega. Gatilho para log/retry: quando houver suporte a revisão manual no admin.
- **Sem histórico.** `cref_verified_at` guarda só a última validação positiva, não as tentativas. Auditoria completa exigiria tabela própria.

---

## 7. Referências

- `backend/core-api/internal/cref/client.go` — cliente do portal.
- `backend/core-api/internal/professionals/handler.go` — rota pública de consulta.
- `backend/core-api/internal/auth/service.go` — exigência de CREF no cadastro.
- `backend/core-api/openapi.yaml` — contrato de `POST /auth/register`, `PUT /professionals/me/profile` e `GET /professionals/cref/validate`.
- `frontend/app/auth/register/page.tsx` — indicadores de CREF verificado e em análise.
- `frontend/components/ProfessionalCard.tsx` — selo público no card do profissional.
