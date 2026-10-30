# Plataforma de Serviços Fitness — Ideia, Arquitetura e Planejamento

> Nome de trabalho: **FitConecta+**

---

## 1. Visão geral

Uma plataforma que conecta **profissionais de educação física autônomos** a **clientes** que buscam atendimento. Os profissionais contam com cadastro gratuito e publicam seus horários disponíveis. Os clientes encontram, filtram, pagam e agendam sessões com confirmação atrelada ao pagamento. A plataforma retém uma comissão de **10% sobre o valor de cada sessão**.

Produtos previstos:

- **Web Pública (Landing / SEO)**: páginas públicas voltadas para captação e indexação de profissionais nos buscadores, com redirecionamento para a área logada.
- **Web App (Logada)**: acesso pelo navegador via Next.js (React) para agendamentos e gestão.
- **App mobile**: Android e iOS (Flutter), com foco no uso recorrente (agendamentos, pagamentos, lembretes e notificações).
- **Backend**: API central (Core API) + BFFs (Backend for Frontend) em **Go**.
- **Banco de dados**: relacional (PostgreSQL).
- **Infraestrutura**: nuvem na **AWS** para produção, com ambiente de desenvolvimento **100% local** (Docker Compose).
- **Push notifications**: prevista para uma fase posterior.
- **Mascote**: um **tucano com uma faixa na testa**, simbolizando foco, energia e identidade brasileira, presente no onboarding, mensagens e estados vazios.

---

## 2. Problema e proposta de valor

| Para quem | Dor atual | O que a plataforma oferece |
|---|---|---|
| Profissional autônomo de EF | Agenda manual (WhatsApp/planilhas), inadimplência/calote, "vai e volta" de horários | Cadastro gratuito, vitrine profissional, agenda organizada e recebimento garantido (split automático com 10% para a plataforma) |
| Cliente | Dificuldade de achar profissional confiável, negociar horários e formas de pagamento | Busca por modalidade e disponibilidade real, agendamento direto e pagamento integrado e seguro |

Diferencial inicial: **disponibilidade real e pagamento integrado**. O horário só é confirmado na agenda quando o pagamento for concluído.

---

## 3. Personas

**Profissional (Personal / Professor de EF Autônomo)**
- Atuação estritamente autônoma (pessoa física).
- Quer preencher a agenda, ter segurança de recebimento prévio e ser encontrado.
- Cadastro gratuito, sem custos de adesão ou mensalidades iniciais.
- Usa majoritariamente o celular.

**Cliente**
- Pessoa que quer treinar com acompanhamento, geralmente com rotina corrida.
- Quer encontrar o profissional, agendar e pagar em poucos toques.
- Acessa via páginas públicas / web ou aplicativo mobile.

---

## 4. Escopo

### 4.1 MVP (primeira versão utilizável)

**Profissional**
- Cadastro gratuito e login (exclusivo para autônomos).
- Perfil público: nome, foto, modalidades, bio, cidade/região, valor da sessão.
- Cadastro de disponibilidade semanal (regras recorrentes) e exceções (folgas, feriados).
- Visualização da agenda e acompanhamento dos agendamentos confirmados.
- Extrato de saldo e recebimentos das sessões realizadas (líquido de 90%).

**Cliente**
- Cadastro e login.
- Busca de profissionais por modalidade, cidade/região e dia/horário (através das páginas públicas com redirecionamento para login).
- Visualização do perfil e dos horários livres.
- **Agendamento e pagamento integrado**:
  - Pagamento realizado via plataforma (Pix no MVP; cartão de crédito em fase futura).
  - O horário fica em pré-reserva e **só é confirmado na agenda após a confirmação do pagamento**.
- **Cancelamento**:
  - Inicialmente **sem cobrança de taxa de cancelamento**, respeitando regra de antecedência mínima estipulada.
- Listagem de agendamentos futuros e passados.

**Plataforma**
- Processamento e split automático de pagamento: **10% retido pela plataforma**, 90% destinado ao profissional.
- E-mail transacional (confirmação de cadastro, link/status de pagamento, confirmação de agendamento).
- Painel administrativo em `/admin` (login próprio, acompanhar agendamentos/transações, cancelar e moderar contas).

### 4.2 Fora do MVP (fases seguintes)

- Push notifications (FCM para Android/iOS e Web Push).
- Avaliações e reputação pública.
- Chat em tempo real entre profissional e cliente.
- Planos de treino, fichas e acompanhamento de evolução física.
- Integração com Google Calendar.
- Múltiplos profissionais / academias (foco inicial é estritamente profissionais autônomos).

---

## 5. Arquitetura

### 5.1 Visão de alto nível

```
┌────────────────────────┐        ┌──────────────────┐        ┌──────────────────┐
│ Páginas Públicas (SEO) │        │  Web (Logada)    │        │  App Mobile      │
│ (Landing / Descoberta) │        │  Flutter Web     │        │  Flutter         │
│ Redireciona p/ login   │        │                  │        │  (Android / iOS) │
└───────────┬────────────┘        └────────┬─────────┘        └────────┬─────────┘
            │                              │                           │
            └──────────────────────────────┼───────────────────────────┘
                                           │
                                  ┌────────▼─────────┐        ┌────────▼─────────┐
                                  │   BFF Web (Go)   │        │   BFF App (Go)   │
                                  │ (agregação web)  │        │ (payloads app)   │
                                  └────────┬─────────┘        └────────┬─────────┘
                                           │                           │
                                           └─────────────┬─────────────┘
                                                         │ (REST interno / gRPC)
                                                ┌────────▼─────────┐
                                                │  Core API (Go)   │
                                                │ (regras, auth,   │
                                                │  pagamentos/10%) │
                                                └────────┬─────────┘
                                                         │
                                        ┌────────────────┼────────────────┐
                                        │                │                │
                               ┌────────▼─────────┐      │      ┌─────────▼────────┐
                               │   PostgreSQL     │      │      │ Gateway / PSP    │
                               │                  │      │      │ (Pix/10%)       │
                               └──────────────────┘      │      └──────────────────┘
                                                ┌────────▼─────────┐
                                                │ Worker / Fila    │
                                                │ (e-mail, jobs,   │
                                                │  webhooks pgto)  │
                                                └──────────────────┘
```

### 5.2 Por que BFF por cliente

- O **app** precisa de payloads enxutos, pensados para rede móvel e cache local.
- A **web** necessita de agregações específicas para apresentação de perfil e fluxo de checkout web.
- Mudanças no fluxo web não quebram o app mobile.
- Permite políticas de segurança e rate limiting específicas por canal.
- Regras de negócio permanecem centralizadas no Core API em Go.

### 5.3 Estratégia Web e SEO

Para contornar as limitações de SEO do Flutter Web:
- **Páginas públicas dedicadas para SEO**: páginas estáticas/SSR para vitrine pública dos profissionais, indexação por buscadores e atração de novos clientes.
- **Redirecionamento para a área logada**: as ações de agendamento e pagamento nessas páginas públicas direcionam o usuário diretamente para a área logada da plataforma (Flutter Web ou deep link para o app mobile).

---

## 6. Tecnologia do backend: Go

O backend será desenvolvido integralmente em **Go** tanto para o Core API quanto para os BFFs.

**Motivos da escolha:**
- Excelente performance com baixo consumo de memória e inicialização rápida.
- Tipagem forte e estática com compilador rápido.
- Superfície de dependências enxuta (stdlib robusta), mitigando riscos de supply chain e vulnerabilidades transitivas.
- Binários estáticos que facilitam containers Docker leves e seguros.
- Concorrência nativa (goroutines/channels) ideal para concorrência de agendamentos e workers.
- Ferramental oficial de segurança (`govulncheck`).

**Stack Go adotada:**
- Roteamento: `chi` ou stdlib `net/http` (Go 1.22+).
- Acesso ao banco: `pgx` + `sqlc` (SQL explícito com tipos gerados, prevenindo SQL injection e dispensando ORM pesado).
- Migrações: `goose` ou `golang-migrate`.
- Validação: validação explícita ou `go-playground/validator`.
- Logs: `log/slog` (estruturado em JSON).
- Testes: `testing` + `testcontainers-go` (PostgreSQL real nos testes locais).
- Contrato de API: OpenAPI (gerando clientes e DTOs para o Flutter).

---

## 7. Modelo de dados (rascunho)

Banco: **PostgreSQL**.

### 7.1 Entidades principais

```
users
  id (uuid, pk)
  email (unique, citext)
  password_hash
  role (client | professional | admin)
  status (pending | active | blocked | inactive)
  created_at, updated_at

professional_profiles           -- exclusivo para autônomos (pessoa física)
  id (uuid, pk)
  user_id (fk users, unique)
  display_name
  bio
  city, state
  session_price_cents
  approved_at

specialties
  id, name (ex.: musculação, funcional, pilates, corrida)

professional_specialties
  professional_id, specialty_id (pk composta)

client_profiles
  id (uuid, pk)
  user_id (fk users, unique)
  name, phone, city, state
  -- dados de saúde: avaliados posteriormente com jurídico

availability_rules            -- disponibilidade semanal recorrente
  id (uuid, pk)
  professional_id (fk)
  weekday (0-6)
  start_time, end_time         -- no fuso do profissional
  slot_minutes (ex.: 60)
  valid_from, valid_until (nullable)

availability_exceptions       -- folgas, feriados, bloqueios
  id, professional_id
  starts_at, ends_at (timestamptz)
  reason

appointments
  id (uuid, pk)
  professional_id (fk)
  client_id (fk)
  starts_at, ends_at (timestamptz)
  status (pending_payment | confirmed | cancelled_by_client | cancelled_by_professional | completed | no_show)
  cancelled_at, cancel_reason
  created_at, updated_at

payments
  id (uuid, pk)
  appointment_id (fk appointments, unique)
  client_id (fk users)
  professional_id (fk users)
  amount_cents                  -- valor bruto total
  platform_fee_cents            -- 10% retido pela FitConecta+
  professional_net_cents        -- 90% repassado ao autônomo
  payment_method (pix | credit_card)
  status (pending | paid | failed | refunded)
  gateway_transaction_id (nullable)
  paid_at (nullable)
  created_at, updated_at

device_tokens                 -- preparado para push futuro
  id, user_id (fk)
  platform (android | ios | web)
  token (unique)
  last_seen_at
```

### 7.2 Garantia contra dupla reserva (ponto crítico)

Evita condições de corrida (race conditions) entre dois clientes agendando simultaneamente:

```sql
CREATE EXTENSION IF NOT EXISTS btree_gist;

ALTER TABLE appointments
  ADD COLUMN period tstzrange GENERATED ALWAYS AS (tstzrange(starts_at, ends_at)) STORED;

ALTER TABLE appointments
  ADD CONSTRAINT no_overlap_professional
  EXCLUDE USING gist (
    professional_id WITH =,
    period WITH &&
  ) WHERE (status IN ('pending_payment', 'confirmed'));
```

O próprio banco bloqueia sobreposições tanto durante a pré-reserva de pagamento (`pending_payment`) quanto após a confirmação (`confirmed`).

### 7.3 Fuso horário

- Todos os registros temporais utilizam `timestamptz` em UTC.
- Fuso do profissional salvo no perfil (ex.: `America/Sao_Paulo`) para expansão dos slots semanais.

---

## 8. Autenticação, autorização e fluxo financeiro

- **Senhas**: Argon2id.
- **Sessão**:
  - App: JWT de curta duração + refresh token rotacionado armazenado em Keychain/Keystore.
  - Web: cookie `HttpOnly`, `Secure`, `SameSite=Strict` com token anti-CSRF.
- **Fluxo de agendamento & pagamento**:
  1. O cliente escolhe um horário disponível.
  2. O backend cria o agendamento em status `pending_payment` com trava temporária (ex.: 10 minutos para conclusão do checkout).
  3. O cliente realiza o pagamento via Pix na plataforma (cartão em fase futura).
  4. O webhook do gateway confirma o pagamento; o agendamento muda para `confirmed`.
  5. A plataforma retém 10% e disponibiliza 90% para repasse ao profissional.
  6. Caso o pagamento expire sem sucesso, a pré-reserva é liberada automaticamente.
- **Cancelamentos**:
  - Inicialmente **sem incidência de taxa ou multa**, respeitando a antecedência mínima estabelecida.
- **Autorização (IDOR/BOLA)**: validação de posse estrita dos recursos no Core API em Go.

---

## 9. Segurança e aspectos jurídicos

### 9.1 Boas práticas contínuas
- Baseado em OWASP Top 10 e OWASP API Security.
- Validação estrita de entradas nos BFFs e Core API.
- Consultas 100% parametrizadas via `sqlc`.
- Cabeçalhos HTTP de segurança (CSP, HSTS, X-Content-Type-Options).
- Gestão de segredos via variáveis de ambiente; nunca comitar `.env`.
- Análise estática no CI com `govulncheck`, `gitleaks` e SAST.

### 9.2 Aspectos jurídicos, LGPD e CREF
- A validação jurídica detalhada (política de privacidade, consentimento para dados pessoais e de saúde pela LGPD, verificação formal do registro CREF e termos de serviço) **será consultada posteriormente**, antes do lançamento oficial em produção.

---

## 10. Push notifications (fase futura)

- Arquitetura de eventos desacoplada: Core API emite eventos (ex.: `payment.confirmed`, `appointment.cancelled`) para uma fila.
- Worker consome a fila e dispara notificações via Firebase Cloud Messaging (FCM para Android/iOS e Web Push).
- Tabela `device_tokens` já prevista na modelagem do banco.

---

## 11. Mascote

### 11.1 Conceito definido
- Personagem: um **tucano com uma faixa na testa**.
- **Atributos visuais e personalidade**:
  - Representa brasilidade, vivacidade e energia acolhedora.
  - A faixa na testa traz o aspecto atlético, remetendo a determinação, treino e foco, de forma leve e carismática (longe de estereótipos agressivos).
  - Bico colorido e expressivo que facilita transmitir reações nas telas da plataforma.

### 11.2 Aplicações no produto
- **Onboarding**: apresenta os primeiros passos e orienta o cadastro de clientes e profissionais.
- **Confirmação de pagamento/agendamento**: comemora a sessão agendada ("Treino confirmado!").
- **Estados vazios**: incentiva a ação ("Ainda sem agendamentos essa semana, bora treinar?").
- **Telas de erro / 404**: pose pensativa ou descontraída.

### 11.3 Implementação
- Ilustrações vetoriais (SVG) e animações leves utilizando **Rive** ou Lottie no Flutter.

---

## 12. Estrutura do repositório (monorepo)

```
fitconecta/
├── backend/
│   ├── core-api/            # Go: regras de negócio, auth, pagamentos (10%), DB
│   │   ├── cmd/api/
│   │   ├── internal/
│   │   │   ├── auth/
│   │   │   ├── professionals/
│   │   │   ├── clients/
│   │   │   ├── availability/
│   │   │   ├── appointments/
│   │   │   ├── payments/    # split de 10%, integração gateway
│   │   │   └── platform/    # db, logs, config
│   │   ├── migrations/
│   │   ├── queries/         # SQL para sqlc
│   │   └── openapi.yaml
│   ├── bff-web/             # Go: agregação para web
│   └── bff-app/             # Go: agregação para mobile
├── web-public/              # Páginas públicas / SEO que redirecionam para a área logada
├── web/                     # Flutter Web (área logada)
├── mobile/                  # Flutter mobile (Android/iOS) com Riverpod
├── packages/                # Dart compartilhado
│   ├── design_system/       # tema, componentes, tokens
│   ├── api_client/          # cliente gerado do OpenAPI
│   └── mascot/              # ilustrações e animações do tucano com faixa
├── infra/
│   ├── local/               # Docker Compose para desenvolvimento 100% local
│   └── aws/                 # Configurações de infraestrutura para produção na AWS
├── docs/                    # ADRs e documentações
└── .github/workflows/       # CI/CD
```

---

## 13. Stack resumida

| Camada | Tecnologia | Detalhes |
|---|---|---|
| Mobile & Web Logada | **Flutter (Dart)** | Código compartilhado entre Android, iOS e Web logada |
| Gerenciamento de Estado | **Riverpod** | Reativo, testável e sem boilerplate excessivo |
| Roteamento Flutter | `go_router` | Roteamento declarativo |
| Páginas Públicas / SEO | **Web Pública (Landing/SEO)** | Páginas de captação que redirecionam para a área logada |
| Backend | **Go** | Core API + BFF Web + BFF App |
| Banco de Dados | **PostgreSQL** | Relacional com extensão `btree_gist` para exclusão de concorrência |
| Acesso ao Banco | `pgx` + `sqlc` | SQL puro compilado com type-safety em Go |
| Migrações | `goose` | Migrações SQL versionadas |
| Pagamentos & Split | Gateway / PSP integrado | Pix (MVP) com retenção de **10% para a plataforma** e 90% para o autônomo; cartão em fase futura |
| Desenvolvimento Local | **Docker Compose (100% local)** | Ambiente completo sem dependência externa durante o desenvolvimento |
| Nuvem / Produção | **AWS** | Deploy de produção e serviços gerenciados |
| E-mail Transacional | SES / Postmark | Disparo de confirmações e links |

---

## 14. Estratégia de testes

- **Unitários**: cálculo de disponibilidade, geração de slots, retenção de taxa de 10% e regras de cancelamento.
- **Integração**: repositórios testados contra PostgreSQL real via `testcontainers-go`.
- **Concorrência**: validação rigorosa contra dupla reserva simultânea (race conditions).
- **Pagamentos**: testes dos webhooks e transição de status de `pending_payment` para `confirmed`.
- **E2E**: fluxos principais com Flutter e Riverpod em mock e ambiente integrado.

---

## 15. Ambientes e Desenvolvimento Local

- **Desenvolvimento 100% Local**:
  - Todo o ecossistema (PostgreSQL, Core API em Go, BFFs e simulador de webhooks) roda localmente via Docker Compose.
  - O desenvolvedor não precisa de conexão ativa com a AWS para codificar, debugar e rodar a suíte de testes.
- **Nuvem (AWS)**:
  - Destinada aos ambientes de homologação (staging) e produção (prod).
  - Deploy automatizado via GitHub Actions.

---

## 16. Observabilidade

- Logs estruturados em JSON (`log/slog`) com `request_id` propagado.
- Métricas: latência de rotas, taxa de sucesso de pagamentos, agendamentos confirmados vs. expirados.
- Tracing com OpenTelemetry e monitoramento de uptime.

---

## 17. Roadmap

### Fase 0: Descoberta e alinhamento inicial (2 a 4 semanas)
- Entrevistas com profissionais autônomos e clientes.
- Protótipo no Figma dos fluxos com o mascote (tucano de faixa na testa).
- Escolha da região geográfica piloto.
- Seleção do gateway de pagamento para o split de 10%.

### Fase 1: Fundações e Setup Local (3 a 4 semanas)
- Monorepo e ambiente de desenvolvimento 100% local com Docker Compose.
- CI/CD básico e linters.
- Base do Core API em Go com autenticação e schema do Postgres.
- Setup inicial do Flutter com **Riverpod** e Design System.
- Primeiro ADR documentando Go, Riverpod, AWS e split de pagamentos.

### Fase 2: Core de agendamento e pagamentos (4 a 6 semanas)
- Cadastro gratuito de profissionais autônomos.
- Regras de disponibilidade semanal e exceções.
- Integração de pagamento com split (10% plataforma / 90% profissional).
- Regra anti-dupla reserva (`pending_payment` -> `confirmed`).
- Regra de cancelamento sem taxa inicial.

### Fase 3: Web pública e Web MVP (3 a 4 semanas)
- Páginas públicas para SEO/vitrine dos profissionais com redirecionamento.
- Aplicação web logada em Flutter Web com Riverpod.
- BFF Web em Go.

### Fase 4: App Mobile MVP (4 a 6 semanas)
- Aplicativo Flutter Android e iOS com Riverpod.
- BFF App em Go.
- Mascote tucano integrado ao onboarding e às confirmações de agendamento.
- Testes fechados (TestFlight e Google Play Console).

### Fase 5: Consultoria jurídica e Piloto (4 a 8 semanas)
- **Consulta jurídica**: alinhamento de LGPD, termos e validação do CREF.
- Pentest e checagens finais de segurança.
- Lançamento do piloto na região selecionada.

### Fase 6: Push notifications e expansão
- Ativação de push notifications (FCM / Web Push).
- Avaliações e reputação.
- Expansão de regiões.

---

## 18. Métricas de sucesso (MVP)

- Volume bruto transacionado (GMV) e receita líquida gerada (10% de taxa da plataforma).
- Profissionais autônomos ativos com agenda aberta.
- Taxa de conversão: visitante da página pública → pagamento/agendamento concluído.
- Taxa de cancelamento e de no-show.
- Retenção de clientes em 30 dias.

---

## 19. Riscos e mitigação

| Risco | Impacto | Mitigação |
|---|---|---|
| Abandono no momento do pagamento | Alto | Checkout simples (Pix instantâneo), reserva temporária de 10 minutos |
| Desistência e cancelamento de última hora | Médio | Regra de antecedência mínima clara; cancelamento sem taxa no início para atrito zero |
| Falhas de concorrência / agendamento duplo | Alto | Constraint de exclusão (`tstzrange` GiST) no PostgreSQL |
| Indexação fraca com Flutter Web | Médio | Páginas públicas dedicadas para SEO que redirecionam para a área logada |
| Vazamento de dados / LGPD | Crítico | Segurança por design, criptografia e consultoria jurídica pré-lançamento |
| Fraude de prestadores sem CREF | Alto | Processo de aprovação manual e consulta jurídica das diretrizes |

---

## 20. Decisões do projeto

### 20.1 Decisões consolidadas
1. **Nome inicial**: **FitConecta+**.
2. **Público prestador**: Exclusivo para profissionais **autônomos** (pessoa física); cadastro **100% gratuito**.
3. **Monetização**: Pagamento da sessão via plataforma com retenção de **10% de comissão** pela FitConecta+ e 90% repassado ao profissional.
4. **Confirmação e cancelamento**: O horário só é **confirmado após a realização do pagamento**. Inicialmente **não haverá cobrança de taxa de cancelamento** (respeitando antecedência mínima).
5. **Linguagem do Backend**: **Go** para o Core API e os BFFs.
6. **Estratégia Web e SEO**: **Páginas públicas** focadas em SEO/captação que redirecionam o usuário para a aplicação web logada (Flutter Web).
7. **Gerenciamento de Estado**: **Riverpod** no Flutter.
8. **Mascote**: Um **tucano com uma faixa na testa**, simbolizando energia, dedicação e identidade brasileira.
9. **Nuvem e Ambiente**: Nuvem de produção na **AWS**, porém com **desenvolvimento 100% local** (via Docker Compose).
10. **Jurídico**: Consultoria jurídica (LGPD, dados de saúde e CREF) será realizada **posteriormente**, antes da abertura pública.

### 20.2 Decisões em aberto
1. **Região geográfica inicial**: Definição da cidade ou bairro específico para início do piloto.
2. **Gateway de pagamento**: Escolha do provedor com suporte a split nativo (ex.: Asaas, Pagar.me, Mercado Pago, Stripe).
3. **Tempo de antecedência mínima para cancelamento**: Definição do prazo mínimo (ex.: 2h, 4h ou 24h antes da sessão).
4. **Registro de marca e domínio**: Verificação no INPI da marca *FitConecta+* e compra dos domínios oficiais.

---

## 21. Próximos passos imediatos

1. Definir a região inicial do piloto e o gateway de split de pagamento.
2. Estruturar o repositório monorepo com Docker Compose para desenvolvimento 100% local.
3. Criar os primeiros ADRs (Go, Riverpod, AWS + Dev local, split de pagamentos de 10%).
4. Criar o protótipo no Figma (fluxos do cliente, profissional e artes do mascote tucano com faixa).
5. Implementar o schema PostgreSQL (entidades `users`, `appointments`, `payments` com split e constraint anti-sobreposição).
