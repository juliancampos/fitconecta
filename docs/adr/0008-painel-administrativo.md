# ADR 0008: Painel Administrativo com Acesso Separado, Gestão de Agendamentos, Pagamentos e Status de Usuários

* **Status:** Substituído parcialmente pelo [ADR 0009](0009-admin-no-mesmo-dominio.md)
* **Data:** 2026-10-20
* **Decisores:** Time de Arquitetura, Produto e Engenharia FitConecta+
* **Contexto Técnico:** Go (Core API), Next.js 16 (Web), PostgreSQL 16, Fly.io

---

> **Nota (2026-10-20):** a separação em um app/host próprio foi substituída pelo [ADR 0009](0009-admin-no-mesmo-dominio.md). O painel agora vive em `/admin` no mesmo frontend público, mantendo login, cookie e segredo administrativos exclusivos. As demais decisões deste ADR — endpoints, status, cancelamento, auditoria e matriz de autorização — continuam válidas.

---

## 1. Contexto e Problema

O FitConecta+ já possui os fluxos de cadastro, login, perfil profissional, disponibilidade, agendamento e pagamento simulado. Porém, a operação ainda não tem uma interface para acompanhar e intervir no marketplace. Hoje, qualquer correção (cancelar um agendamento, investigar um pagamento ou restringir um usuário) exigiria acesso direto ao banco de dados.

O painel administrativo precisa permitir:

1. **Agendamentos:** listar todos os agendamentos e filtrar por cliente, profissional e data.
2. **Pagamentos:** listar todos os pagamentos e filtrar por cliente, profissional e data.
3. **Cancelamento:** cancelar um agendamento em nome da plataforma.
4. **Bloqueio:** bloquear um cliente ou profissional, impedindo **novos agendamentos**.
5. **Inativação:** inativar um cliente ou profissional, impedindo o **login** na plataforma.

A dúvida central é onde o painel deve viver:

- Uma nova URL, com login separado.
- Um menu dentro da aplicação atual.

A avaliação preliminar do time é que uma **nova URL com login separado** é a opção mais adequada. Este ADR confirma essa direção e detalha as consequências técnicas.

### Restrições do projeto

- Time pequeno e operação enxuta; a solução deve ser simples de operar.
- O backend já expõe JWT em cookie `HttpOnly` (`fitconecta_jwt`) e o frontend já usa um proxy same-origin (`frontend/proxy.ts`).
- O papel `admin` já existe em `users.role`, mas não há login, rota, tela ou regra de autorização específica para administradores.
- O schema já tem `users.status` com os valores `pending`, `active` e `suspended`, porém o status **não é verificado no login** nem nas rotas autenticadas.
- Dados pessoais (nome, e-mail, telefone) e financeiros serão expostos a administradores; segurança, auditoria e LGPD são requisitos.

---

## 2. Decisão

Decidimos criar um **painel administrativo em uma aplicação separada**, publicado em uma **URL própria**, com **login próprio e sessão própria**:

- **Produção:** `https://admin.fitconecta.com.br`
- **Fly.io (antes do domínio customizado):** `https://admin-fit-conecta.fly.dev`

A aplicação administrativa será um novo app Next.js (`admin/`), separado do frontend público (`frontend/`), e será publicada como um terceiro app no Fly.io. O backend continua sendo um único Core API, que ganhará um módulo `internal/admin` e rotas sob `/api/v1/admin/*`.

O login administrativo é **separado do login público**:

- Endpoint próprio: `POST /api/v1/admin/auth/login`.
- Cookie próprio: `fitconecta_admin_jwt`, `HttpOnly`, `Secure`, `SameSite=Strict`, **host-only** (sem `Domain`).
- Segredo de assinatura próprio: `ADMIN_JWT_SECRET`, diferente do `JWT_SECRET` público.
- Audiência própria no JWT: `aud = "fitconecta-admin"`.
- TTL menor que o da sessão pública (padrão: 8 horas).
- Sem cadastro público de administradores.

Assim, um token ou cookie do site público não é aceito no painel, e vice-versa. O painel não compartilha origem, bundle JavaScript, cookie nem segredo de sessão com a aplicação pública.

### 2.1 Alternativas consideradas

| Alternativa | Prós | Contras | Decisão |
|---|---|---|---|
| **A. Menu `/admin` dentro do frontend público** | Menor custo; reutiliza layout e componentes; um único deploy | Administração no mesmo bundle e na mesma origem do site público; cookie público com acesso administrativo; superfície de ataque maior; link descobrível; difícil restringir por IP/rede | Rejeitada |
| **B. Mesma base Next.js, segundo deploy em subdomínio** | Reutiliza componentes; URL separada; cookie separado | O bundle público ainda carrega código administrativo; roteamento por host adiciona complexidade; deploys e permissões ficam acoplados | Viável, mas não escolhida |
| **C. Nova aplicação `admin/` em URL própria + novo app Fly** | Isolamento de origem, cookie, segredo e bundle; deploy independente; menor superfície pública; evolução de segurança sem afetar o site | Mais um app, Dockerfile, fly.toml e secrets; alguma duplicação inicial de componentes | **Escolhida** |
| **D. Backoffice em serviço/banco separado** | Isolamento máximo; escalabilidade e compliance | Custo e complexidade desproporcionais ao MVP; duplicação de dados e sincronização | Postergada |

> A URL separada **não é um mecanismo de segurança por obscuridade**. O controle de acesso continua sendo autenticação, autorização, rate limit, auditoria e, futuramente, MFA. A separação serve para reduzir o raio de exposição de cookies, bundles e erros de configuração.

### 2.2 Por que login separado

| Aspecto | Login público | Login administrativo |
|---|---|---|
| Público-alvo | Clientes e profissionais | Equipe interna |
| Cadastro | Aberto (`/auth/register`) | Fechado; criado por CLI/seed |
| Endpoint | `/api/v1/auth/login` | `/api/v1/admin/auth/login` |
| Cookie | `fitconecta_jwt` | `fitconecta_admin_jwt` |
| Segredo JWT | `JWT_SECRET` | `ADMIN_JWT_SECRET` |
| Audiência | `fitconecta-web` | `fitconecta-admin` |
| TTL padrão | 24 h | 8 h |
| Status exigido | `active` ou `blocked` | `active` |
| MFA | Não previsto no MVP | Recomendado; obrigatório antes de produção com dados reais |

O endpoint público de login passa a recusar explicitamente `role = admin`. O endpoint administrativo recusa qualquer usuário que não seja `admin` com a mesma mensagem genérica de credenciais inválidas, evitando enumeração de contas.

---

## 3. Modelo de Status do Usuário

O requisito distingue duas restrições diferentes:

- **Bloqueado:** pode autenticar e consultar seus dados, mas **não pode realizar novos agendamentos**.
- **Inativo:** **não pode autenticar**; sessões já emitidas deixam de valer.

O enum atual `user_status` possui `pending`, `active` e `suspended`. Propomos:

- Renomear `suspended` para `inactive` (preserva o significado de "acesso suspenso" e evita dois nomes para a mesma ideia).
- Adicionar `blocked`.
- Manter `pending` para o futuro fluxo de aprovação de profissionais; hoje o cadastro já cria `active`.
- Adicionar `status_reason` e `status_changed_at` em `users` para exibir o motivo ao usuário restrito e facilitar o suporte.

### 3.1 Matriz de comportamento

| Ação | `active` | `blocked` | `inactive` |
|---|---|---|---|
| Login público | Sim | Sim | **Não** |
| Acessar `/app/*` com sessão existente | Sim | Sim | **Não** (sessão invalidada) |
| Ver/cancelar os próprios agendamentos | Sim | Sim | Não |
| Criar novo agendamento como cliente | Sim | **Não** | Não |
| Ser escolhido em novo agendamento como profissional | Sim | **Não** | Não |
| Aparecer na listagem pública de profissionais | Sim | **Não** | Não |
| Confirmar pagamento de agendamento já existente | Sim | Sim | Não |
| Login administrativo | Sim, se `role = admin` | Não | Não |
| Receber ações administrativas | Sim | Sim | Sim |

Decisões importantes:

- **Bloqueio não cancela nem apaga agendamentos existentes.** A plataforma pode cancelá-los manualmente pelo painel, com motivo e tratamento financeiro.
- **Inativação invalida a sessão em todas as requisições autenticadas**, não apenas no próximo login. Para isso, o middleware consulta o status atual do usuário no banco (com cache curto opcional).
- **Bloqueio não retroage sobre pagamentos pendentes já existentes**; o cliente pode concluir ou cancelar o que já havia reservado. Novos agendamentos são recusados.
- **Administradores não podem bloquear/inativar a si mesmos nem outros administradores** por este endpoint. Gestão de administradores é operação separada (CLI/runbook).

### 3.2 Alternativas de modelagem

| Alternativa | Prós | Contras | Decisão |
|---|---|---|---|
| **A. Estender `users.status`** (`active`, `blocked`, `inactive`, `pending`) | Simples; uma fonte de verdade; consultas e filtros triviais; já existe enum | Histórico completo exige tabela de auditoria | **Escolhida** |
| **B. Colunas `blocked_at` / `inactivated_at`** | Preserva datas e permite consultas temporais | Duplica estado; combinações inválidas; mais difícil de consultar | Rejeitada |
| **C. Tabela `user_restrictions`** | Várias restrições simultâneas, expiração, escopo | Complexidade desproporcional ao MVP | Postergada |

---

## 4. Arquitetura

```text
┌──────────────────────────────┐        ┌──────────────────────────────┐
│  Site público                │        │  Painel administrativo       │
│  frontend-fit-conecta        │        │  admin-fit-conecta           │
│  fitconecta.com.br           │        │  admin.fitconecta.com.br     │
│  cookie: fitconecta_jwt      │        │  cookie: fitconecta_admin_jwt│
│  segredo: JWT_SECRET         │        │  segredo: ADMIN_JWT_SECRET   │
└──────────────┬───────────────┘        └──────────────┬───────────────┘
               │  /api/* (proxy same-origin)           │  /api/v1/admin/* (proxy)
               └──────────────────┬────────────────────┘
                                  ▼
                    ┌──────────────────────────┐
                    │  Core API (Go/chi)       │
                    │  /api/v1/*      público  │
                    │  /api/v1/admin/* admin   │
                    └────────────┬─────────────┘
                                 ▼
                    ┌──────────────────────────┐
                    │  Neon PostgreSQL 16      │
                    │  users, appointments,    │
                    │  payments, audit log     │
                    └──────────────────────────┘
```

### 4.1 Componentes

- **`admin/` (novo app Next.js):** login, dashboard, agendamentos, pagamentos e usuários. Usa o mesmo padrão de proxy same-origin do site público, mas com um matcher restrito a `/api/v1/admin/*`.
- **`internal/admin` (novo pacote Go):** handlers, serviço e repositório das consultas e ações administrativas. Não duplica regras de agendamento; chama serviços/repositórios existentes quando possível.
- **Core API:** monta `api.Mount("/admin", adminHandler.Routes(...))` dentro de `/api/v1`.
- **Banco:** mesmas tabelas, com os ajustes de enum, colunas, índices e auditoria descritos neste ADR.
- **Fly.io:** ~~novo app `admin-fit-conecta`~~ (substituído pelo ADR 0009: rotas `/admin` no frontend público).

### 4.2 Isolamento de sessão e rotas

> **Substituído pelo [ADR 0009](0009-admin-no-mesmo-dominio.md):** o frontend público encaminha `/api/v1/admin/*` e protege `/admin/*`; a separação passa a ser por rota, cookie e segredo, não por hostname.

- ~~O app público **não** deve encaminhar `/api/v1/admin/*`. O `frontend/proxy.ts` deve responder `404` para esse prefixo, reduzindo a superfície mesmo que alguém obtenha um token administrativo.~~
- O app administrativo deve encaminhar somente `/api/v1/admin/*`; não precisa do restante da API pública.
- O cookie administrativo é host-only. Como `admin.fitconecta.com.br` é outro host, o navegador nunca o envia ao site público.
- O token administrativo usa `ADMIN_JWT_SECRET` e `aud=fitconecta-admin`. O middleware público rejeita tokens com essa audiência; o middleware administrativo rejeita tokens públicos.
- `ALLOWED_ORIGINS` do backend deve incluir a URL administrativa apenas se houver chamada direta à API. Com proxy same-origin, o navegador não faz CORS; ainda assim, manter a lista explícita e sem `*`.
- O site público não deve ter link para o painel. O app administrativo deve enviar `X-Robots-Tag: noindex, nofollow` e `X-Frame-Options: DENY` (ou CSP `frame-ancestors 'none'`).

### 4.3 Autenticação administrativa

Fluxo:

1. Administrador acessa `https://admin.fitconecta.com.br/login`.
2. Envia e-mail e senha para `POST /api/v1/admin/auth/login` via proxy same-origin.
3. Backend valida credenciais, exige `role = admin` e `status = active`, e emite JWT assinado com `ADMIN_JWT_SECRET`, `aud=fitconecta-admin`.
4. Resposta grava o cookie `fitconecta_admin_jwt` (`HttpOnly`, `Secure`, `SameSite=Strict`, host-only, TTL 8 h).
5. O proxy do app administrativo protege todas as rotas, exceto `/login`, verificando o cookie.
6. O backend valida o JWT e consulta o status atual do administrador; se não estiver `active`, retorna `401`.
7. `POST /api/v1/admin/auth/logout` expira o cookie.

Criação de administradores:

- Nenhum endpoint público cria `admin`. `auth.Register` já aceita apenas `client` e `professional`.
- Criar um comando `cmd/admin` (ou script operacional) para `create`, `reset-password` e `disable`, usando o mesmo algoritmo de hash do `auth.Service`.
- Nunca usar senha padrão em código, migration ou repositório. O primeiro administrador é criado por comando manual com senha forte.

Proteções adicionais:

- Rate limit no login administrativo: por IP e por e-mail (ex.: 5 tentativas por 15 minutos, com backoff).
- Mensagem genérica para credenciais inválidas, usuário inexistente, não-admin e inativo.
- Registrar tentativas de login (sucesso e falha) em log estruturado; não registrar senha nem token.
- MFA TOTP/WebAuthn: recomendado para o MVP e **obrigatório antes de operar com dados reais de produção**.
- Opcionalmente, restringir o app administrativo por Cloudflare Access, VPN ou allowlist de IP antes do login.

---

## 5. API Administrativa

Todas as rotas abaixo ficam sob `/api/v1/admin` e, exceto o login, exigem `AdminAuthMiddleware` (`aud=fitconecta-admin`, `role=admin`, status `active`).

### 5.1 Autenticação

| Método | Rota | Descrição |
|---|---|---|
| `POST` | `/auth/login` | Login exclusivo de administrador; grava `fitconecta_admin_jwt` |
| `POST` | `/auth/logout` | Expira o cookie administrativo |
| `GET` | `/auth/me` | Dados do administrador autenticado |

### 5.2 Agendamentos

| Método | Rota | Descrição |
|---|---|---|
| `GET` | `/appointments` | Lista paginada com filtros |
| `GET` | `/appointments/{id}` | Detalhe do agendamento, cliente, profissional e pagamento |
| `POST` | `/appointments/{id}/cancel` | Cancela em nome da plataforma |

Filtros de `GET /appointments`:

| Parâmetro | Tipo | Semântica |
|---|---|---|
| `client_id` | UUID | `users.id` do cliente |
| `professional_id` | UUID | `users.id` do profissional |
| `q` | string | Busca textual em nome/e-mail do cliente e nome/e-mail do profissional |
| `status` | enum | `pending_payment`, `confirmed`, `cancelled_by_client`, `cancelled_by_professional`, `cancelled_by_admin`, `completed`, `no_show` |
| `date_from` | data/RFC3339 | Início inclusivo do intervalo |
| `date_to` | data/RFC3339 | Fim exclusivo do intervalo |
| `date_field` | enum | `starts_at` (padrão), `created_at`, `updated_at` |
| `page` | int | Página 1-based; padrão 1 |
| `page_size` | int | Padrão 20; máximo 100 |
| `sort` | enum | `starts_at` (padrão desc), `created_at`, `status` |

Resposta (dentro do envelope `data` já usado por `httputil.Success`):

```json
{
  "items": [
    {
      "id": "uuid",
      "starts_at": "2026-10-20T14:00:00-03:00",
      "ends_at": "2026-10-20T15:00:00-03:00",
      "status": "confirmed",
      "client": { "user_id": "uuid", "name": "Maria", "email": "maria@exemplo.com" },
      "professional": { "user_id": "uuid", "name": "João", "email": "joao@exemplo.com" },
      "specialty": "Musculação",
      "payment": { "amount_cents": 10000, "status": "paid", "method": "pix" }
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 42
}
```

Regras:

- `client_id` e `professional_id` são **IDs de usuário** (`users.id`), consistentes com a tabela `payments` e com as ações de bloqueio/inativação. Os IDs de perfil (`client_profiles.id` e `professional_profiles.id`) são detalhes internos e não devem ser exigidos do operador.
- `date_from`/`date_to` usam intervalo **fechado no início e aberto no fim** (`>= from` e `< to`). Datas sem hora são interpretadas em `America/Sao_Paulo`; o frontend envia o intervalo já convertido quando necessário.
- `q` usa `ILIKE` no MVP. Índices trigram podem ser adicionados quando o volume justificar.
- Ordenação e paginação usam allowlist; nunca concatenar `sort` ou `page_size` diretamente no SQL.

### 5.3 Pagamentos

| Método | Rota | Descrição |
|---|---|---|
| `GET` | `/payments` | Lista paginada com filtros |
| `GET` | `/payments/summary` | Totais e contagens com os mesmos filtros |

Filtros de `GET /payments`:

| Parâmetro | Tipo | Semântica |
|---|---|---|
| `client_id` | UUID | `payments.client_id` (`users.id`) |
| `professional_id` | UUID | `payments.professional_id` (`users.id`) |
| `q` | string | Busca em nome/e-mail do cliente e do profissional |
| `status` | enum | `pending`, `paid`, `failed`, `refunded`, `refund_pending` |
| `method` | enum | `pix`, `credit_card` |
| `date_from` / `date_to` | data/RFC3339 | Intervalo semiaberto |
| `date_field` | enum | `created_at` (padrão), `paid_at` |
| `page` / `page_size` | int | Mesmos limites de agendamentos |
| `sort` | enum | `created_at` (padrão desc), `paid_at`, `amount_cents` |

Para "pagamentos realizados", a tela usa `status=paid` e `date_field=paid_at`. O resumo retorna:

```json
{
  "count": 120,
  "amount_cents": 1200000,
  "platform_fee_cents": 120000,
  "professional_net_cents": 1080000,
  "by_status": { "paid": 110, "pending": 6, "refunded": 4 }
}
```

### 5.4 Usuários (clientes e profissionais)

| Método | Rota | Descrição |
|---|---|---|
| `GET` | `/users` | Lista clientes/profissionais com filtros |
| `GET` | `/users/{id}` | Detalhe do usuário e perfil |
| `PATCH` | `/users/{id}/status` | Altera status para `active`, `blocked` ou `inactive` |

Filtros de `GET /users`: `role` (`client` ou `professional`), `status`, `q` (nome/e-mail), `page`, `page_size`. A listagem não retorna hash de senha nem dados sensíveis além do necessário.

`PATCH /users/{id}/status`:

```json
{
  "status": "blocked",
  "reason": "Descumprimento dos termos de uso"
}
```

Regras:

- Apenas `client` e `professional` podem ser alvo.
- O administrador não pode alterar o próprio status nem o de outro administrador.
- `blocked` e `inactive` exigem `reason` com pelo menos 5 caracteres; `active` limpa o motivo.
- A operação é transacional e grava auditoria.
- A resposta inclui o usuário atualizado e, quando aplicável, a contagem de agendamentos futuros que permanecem ativos (para orientar o operador).

### 5.5 Cancelamento administrativo

`POST /appointments/{id}/cancel`:

```json
{
  "reason": "Cancelamento solicitado pela operação"
}
```

Fluxo transacional:

1. `SELECT ... FOR UPDATE` do agendamento.
2. Permitir apenas `pending_payment` ou `confirmed`; caso contrário, `409`.
3. Atualizar `status = 'cancelled_by_admin'`, `cancelled_at = NOW()` e `cancel_reason`.
4. Se o pagamento estiver `paid`, atualizar para `refund_pending` (reembolso processado conforme ADR 0007; no piloto, fila manual).
5. Gravar `admin_audit_log`.
6. Notificar cliente e profissional (e-mail/device token) em fase posterior.

> O método atual `appointments.Cancel` mapeia qualquer papel diferente de `professional` para `cancelled_by_client`. O cancelamento administrativo **não deve reutilizar esse método** sem ajuste, sob pena de registrar o cancelamento com o status errado.

### 5.6 Códigos HTTP

| Situação | Código |
|---|---|
| Não autenticado / token inválido / inativo | `401` |
| Autenticado sem `role=admin` ou audiência errada | `403` |
| Recurso não encontrado | `404` |
| Conflito de estado (agendamento já cancelado/concluído) | `409` |
| Payload inválido (status, motivo, filtros) | `422` |
| Rate limit de login | `429` |

---

## 6. Persistência e Migrações

### 6.1 Enums e colunas

```sql
-- Migration separada; não usar os novos valores na mesma transação.
ALTER TYPE user_status RENAME VALUE 'suspended' TO 'inactive';
ALTER TYPE user_status ADD VALUE 'blocked';

ALTER TYPE appointment_status ADD VALUE 'cancelled_by_admin';
ALTER TYPE payment_status ADD VALUE 'refund_pending';

ALTER TABLE users ADD COLUMN IF NOT EXISTS status_reason TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS status_changed_at TIMESTAMPTZ;
```

Observações:

- `ALTER TYPE ... ADD VALUE` deve rodar em migration própria ou fora de transação que utilize o valor imediatamente. `RENAME VALUE` é transacional.
- O valor `pending` permanece para o futuro fluxo de aprovação de profissionais. Hoje o cadastro cria `active`.
- `status_reason` e `status_changed_at` são o estado atual; o histórico completo fica em `admin_audit_log`.

### 6.2 Auditoria

```sql
CREATE TABLE IF NOT EXISTS admin_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    admin_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action VARCHAR(40) NOT NULL,
    target_type VARCHAR(20) NOT NULL,
    target_id UUID,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    ip INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_admin_action CHECK (action IN (
        'admin_login',
        'appointment_cancel',
        'user_block',
        'user_unblock',
        'user_inactivate',
        'user_reactivate'
    )),
    CONSTRAINT chk_admin_target_type CHECK (target_type IN ('appointment', 'user', 'payment'))
);

CREATE INDEX IF NOT EXISTS idx_admin_audit_created
    ON admin_audit_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_admin_audit_admin
    ON admin_audit_log(admin_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_admin_audit_target
    ON admin_audit_log(target_type, target_id, created_at DESC);
```

A tabela é **append-only**: não há endpoint de edição ou exclusão. Retenção e expurgo seguem a política de dados do projeto.

### 6.3 Índices para consultas administrativas

```sql
CREATE INDEX IF NOT EXISTS idx_users_role_status
    ON users(role, status);

CREATE INDEX IF NOT EXISTS idx_appointments_starts_at
    ON appointments(starts_at DESC);
CREATE INDEX IF NOT EXISTS idx_appointments_status_starts
    ON appointments(status, starts_at DESC);

CREATE INDEX IF NOT EXISTS idx_payments_created_at
    ON payments(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payments_paid_at
    ON payments(paid_at DESC);
CREATE INDEX IF NOT EXISTS idx_payments_client_created
    ON payments(client_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payments_professional_created
    ON payments(professional_id, created_at DESC);
```

Os índices existentes `idx_appointments_client`, `idx_appointments_prof` e `idx_appointments_status` continuam válidos. Índices trigram para busca textual podem ser adicionados depois, se necessário.

---

## 7. Regras de Autorização e Impactos no Código Existente

### 7.1 Middleware

O `AuthMiddleware` atual (`internal/platform/middleware/auth.go`) valida o JWT e cria `AuthUser{ID, Email, Role}`, mas não consulta o status. A proposta:

- Adicionar `Status` a `AuthUser`.
- Injetar um `UserStatusReader` no `AuthMiddleware` público para consultar `users.status` (com cache curto de 30–60 s por instância, se necessário).
- Se `inactive`: `401` imediatamente.
- Se `blocked`: permitir a requisição; a restrição de agendamento é aplicada no serviço.
- Criar `AdminAuthMiddleware`, com `ADMIN_JWT_SECRET`, leitura de `fitconecta_admin_jwt`, validação de `aud=fitconecta-admin`, `role=admin` e status `active`.
- O middleware público deve rejeitar tokens administrativos; o middleware administrativo deve rejeitar tokens públicos. Segredos diferentes já garantem isso criptograficamente.
- Revisar `RequireRole`: hoje qualquer `admin` passa em qualquer `RequireRole`. Rotas administrativas devem usar um `RequireAdmin` explícito, e rotas públicas não devem tratar `admin` como cliente/profissional por acidente.

### 7.2 Autenticação pública

`auth.Service.Login` atualmente não verifica `users.status`. Deve passar a:

- Permitir login apenas para `active` e `blocked`.
- Recusar `inactive` e `pending` com `401` genérico.
- Recusar `admin` no endpoint público (o administrador usa o endpoint próprio).

`auth.Service.Register` já restringe o papel a `client` ou `professional`; manter essa regra e testar explicitamente que `admin` é rejeitado.

### 7.3 Agendamentos

`appointments.Service.Create` já exige `clientUser.Role == "client"`. Deve também:

- Exigir `clientUser.Status == "active"`; se `blocked`, retornar `403` com mensagem clara ("Sua conta está bloqueada para novos agendamentos").
- Buscar o status do profissional e exigir `active`; se `blocked` ou `inactive`, retornar `409` ("Profissional indisponível para novos agendamentos").
- A listagem pública de profissionais e a geração de slots devem excluir profissionais `blocked`/`inactive`.
- O cancelamento e a confirmação de pagamento de agendamentos existentes continuam permitidos para usuários `blocked`, pois não criam nova reserva.

### 7.4 Administração

- `internal/admin` deve reutilizar `appointments.Repository` para leitura e cancelamento, mas com um método explícito de cancelamento administrativo.
- Ações de status devem ser transacionais e sempre gravar auditoria com `admin_user_id`, `action`, `target_type`, `target_id`, `metadata`, `ip` e `user_agent`.
- Nenhuma rota administrativa deve aceitar `role` ou status vindos do cliente sem validação de allowlist.

---

## 8. Interface Administrativa

Telas mínimas:

1. **Login** (`/login`): e-mail, senha, mensagem genérica de erro.
2. **Visão geral** (opcional no MVP): agendamentos do dia, pagamentos recentes, contagens por status e alertas de reembolso pendente.
3. **Agendamentos** (`/agendamentos`):
   - Filtros por cliente, profissional, intervalo de datas, status e busca textual.
   - Tabela com data/hora, cliente, profissional, especialidade, status e valor.
   - Detalhe em drawer/página com histórico e pagamento.
   - Ação "Cancelar" com motivo obrigatório e confirmação; se houver pagamento pago, avisar que será marcado para reembolso.
4. **Pagamentos** (`/pagamentos`):
   - Filtros por cliente, profissional, data (padrão `paid_at`), status e método.
   - Totais do filtro (bruto, taxa da plataforma, líquido do profissional).
   - Detalhe com `gateway_transaction_id`, status e vínculo com o agendamento.
5. **Usuários** (`/usuarios`):
   - Filtros por papel, status e busca.
   - Ações "Bloquear", "Inativar" e "Reativar" com motivo.
   - Aviso da quantidade de agendamentos futuros antes de confirmar.

Convenções de UX:

- Bloqueio e inativação têm textos distintos e explícitos sobre o efeito (agendar vs. entrar).
- Ações destrutivas exigem confirmação e motivo; o motivo é exibido ao usuário restrito quando aplicável.
- Nenhuma ação administrativa depende de edição direta no banco.
- O painel não deve exibir hash de senha, token, payload completo do gateway ou dados de cartão.

---

## 9. Segurança, Privacidade e Operação

### 9.1 Segurança

- Cookie administrativo `HttpOnly`, `Secure`, `SameSite=Strict`, host-only, TTL curto.
- Segredo JWT separado e audiência separada.
- Rate limit no login e nas ações sensíveis.
- Verificação de `Origin`/`Referer` nas mutações administrativas (mesmo com `SameSite=Strict`), rejeitando origens diferentes da URL administrativa.
- CSP e `frame-ancestors 'none'`; `noindex`.
- Logs estruturados com `request_id`, `admin_user_id`, ação e alvo; nunca senha/token.
- Alarmes para picos de falha de login, cancelamentos em massa e mudanças de status em massa.
- MFA antes de produção com dados reais.
- Revisão periódica de contas administrativas e rotação de `ADMIN_JWT_SECRET`.

### 9.2 LGPD e retenção

- O painel acessa dados pessoais e financeiros; o acesso é restrito à equipe interna e auditado.
- Coletar apenas o necessário nas listagens; detalhes sob demanda.
- Definir prazo de retenção para `admin_audit_log` e logs de aplicação.
- Documentar a base legal e o processo de atendimento a titulares; não usar dados administrativos para finalidades incompatíveis.

### 9.3 Operação

- Healthcheck do app administrativo e do backend.
- Runbook para: criar administrador, resetar senha, desativar administrador, investigar reembolso e reverter bloqueio.
- Seed do primeiro administrador via CLI, nunca via migration com senha fixa.
- Deploy independente do app administrativo; mudanças de API exigem compatibilidade com o site público.

---

## 10. Consequências

### Positivas

- Separação clara entre sessão pública e administrativa; menor risco de vazamento de cookie entre origens.
- Código e bundle administrativos fora do site público.
- Operação do marketplace sem acesso direto ao banco.
- Auditoria de todas as ações sensíveis.
- Base para MFA, RBAC e backoffice mais robusto no futuro.
- Deploy e rollback independentes do painel.

### Negativas

- Mais um app, Dockerfile, fly.toml, domínio, secrets e monitoramento.
- Duplicação inicial de componentes de UI e configuração de API.
- Consulta de status por requisição no middleware (mitigável com cache curto).
- Necessidade de coordenar migrations de enum e desdobramento do backend antes do painel.
- Mais superfície operacional e responsabilidade sobre dados pessoais.

### Mitigações

- Extrair um pacote compartilhado de UI/API somente quando a duplicação se mostrar custosa.
- Cache curto de status por instância e invalidação no caminho de escrita.
- Migration forward-only, com checklist e teste em banco de staging.
- Testes automatizados da matriz de autorização e dos filtros.
- Runbook e revisão de acesso periódica.

---

## 11. Plano de Implementação

### Fase 0 — Decisão e preparação

- Aprovar este ADR e a nomenclatura de status (`blocked` = bloqueado, `inactive` = inativo).
- Definir domínio administrativo e política de MFA.
- Criar o comando/runbook de criação do primeiro administrador.

### Fase 1 — Banco e regras de status

- Migration de enums (`inactive`, `blocked`, `cancelled_by_admin`, `refund_pending`), colunas de status e auditoria.
- Índices administrativos.
- Atualizar `auth.Service.Login`, `AuthMiddleware` e `appointments.Service.Create`.
- Filtrar profissionais `blocked`/`inactive` na listagem e nos slots.
- Testes de migração e da matriz de comportamento.

### Fase 2 — Backend administrativo

- `internal/admin` com autenticação, middlewares e endpoints de leitura.
- Filtros, paginação, resumo de pagamentos e detalhe.
- Cancelamento administrativo transacional + auditoria.
- Bloqueio/inativação/reativação + auditoria.
- Testes de integração HTTP e autorização.

### Fase 3 — Painel administrativo (MVP)

> Substituída pelo [ADR 0009](0009-admin-no-mesmo-dominio.md): o painel foi incorporado ao frontend público em `/admin`, sem app/Fly separado.

- ~~Novo app Next.js `admin/`, Dockerfile e fly.toml.~~
- Login, guarda de rotas e proxy restrito a `/api/v1/admin/*`.
- Telas de agendamentos, pagamentos e usuários.
- Deploy em `admin-fit-conecta.fly.dev` e validação ponta a ponta.

### Fase 4 — Endurecimento

- MFA, rate limit distribuído, CSP, alertas e retenção.
- Revisão de acessos e rotação de segredos.
- Domínio customizado `admin.fitconecta.com.br`.

---

### Status da implementação (2026-10-20)

- **Fases 0 a 3:** código concluído — migration 0009, matriz de status no backend, pacote `internal/admin`, CLI `cmd/admin` e painel em `/admin` no frontend (ADR 0009).
- **Validação local:** login admin, filtros, cancelamento com `refund_pending`, bloqueio/inativação/reativação e auditoria testados contra PostgreSQL 16; `go test ./...` e builds do Next.js passaram.
- **Pendente de deploy:** aplicar a migration 0009 em produção, configurar `ADMIN_JWT_SECRET`/`BACKEND_PROXY_URL`, publicar o frontend com `/admin` e criar o primeiro administrador.
- **Fase 4:** MFA, rate limit distribuído, CSP, alertas, retenção e domínio customizado continuam pendentes.

---

## 12. Riscos e Mitigação

| Risco | Impacto | Mitigação |
|---|---|---|
| Vazamento de sessão administrativa | Alto | Cookie host-only, `HttpOnly`, `Secure`, `SameSite=Strict`, TTL curto, segredo separado, MFA, rotação |
| Admin bloqueia/inativa usuário errado | Alto | Confirmação com nome/e-mail, motivo obrigatório, auditoria, possibilidade de reativar |
| Cancelamento administrativo com pagamento pago | Alto | Transação, status `refund_pending`, integração com ADR 0007, alerta operacional |
| Migration de enum falhar em produção | Médio | Migration separada, teste em staging, backup, janela controlada |
| Middleware consultando banco em toda requisição | Médio | Cache curto por instância; invalidar no caminho de escrita; monitorar latência |
| Painel exposto sem MFA | Alto | Bloqueador para produção com dados reais; rate limit e allowlist como mitigação temporária |
| Duplicação de UI/API entre apps | Baixo | Extrair pacote compartilhado quando necessário |
| Token público forjado com `role=admin` | Alto | Segredo separado + audiência; middleware administrativo não aceita token público |
| Profissional bloqueado continuar aparecendo | Médio | Filtrar status na listagem pública e nos slots; testar |

---

## 13. Métricas de Sucesso

- Tempo para localizar um agendamento/pagamento por cliente, profissional ou data.
- Percentual de operações administrativas feitas pelo painel (meta: 100% para cancelamento e status).
- Número de incidentes por ação administrativa incorreta.
- Latência p95 das listagens administrativas.
- Cobertura de auditoria (100% das ações sensíveis).
- Taxa de falha de login administrativo e alertas de anomalia.

---

## 14. Referências

### Documentos internos

- [ADR 0001 — Modelagem do banco](0001-estrutura-do-banco-de-dados.md)
- [ADR 0002 — Definição das páginas](0002-definicao-das-paginas.md)
- [ADR 0005 — Fluxo de agendamento](0005-fluxo-de-agendamento-de-sessao.md)
- [ADR 0006 — Deploy separado no Fly.io](0006-deploy-flyio-separado-backend-frontend.md)
- [ADR 0007 — Integração com plataforma de pagamentos](0007-integracao-plataforma-de-pagamentos.md)
- `backend/core-api/internal/platform/middleware/auth.go`
- `backend/core-api/internal/auth/service.go`
- `backend/core-api/internal/appointments/service.go`
- `frontend/proxy.ts`
- `backend/core-api/migrations/00001_initial_schema.sql`

### Externas

- [OWASP ASVS](https://owasp.org/www-project-application-security-verification-standard/)
- [OWASP Cheat Sheet: Authentication](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
- [OWASP Cheat Sheet: Authorization](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html)
- [OWASP Cheat Sheet: CSRF Prevention](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html)
- [Next.js — Proxy](https://nextjs.org/docs/app/api-reference/file-conventions/proxy)
- [Fly.io — Secrets](https://fly.io/docs/reference/secrets/)
- [LGPD — Lei 13.709/2018](https://www.planalto.gov.br/ccivil_03/_ato2015-2018/2018/lei/l13709.htm)
