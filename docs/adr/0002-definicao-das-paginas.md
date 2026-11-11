# ADR 0002: Definição das Páginas Frontend (Next.js)

* **Status:** Aceito
* **Data:** 2026-10-13
* **Decisores:** Time de Produto e Engenharia Frontend
* **Contexto:** O frontend foi implementado com **Next.js (React) + Tailwind CSS**. Precisamos mapear as páginas da aplicação à modelagem de dados já definida no backend.

## 1. Visão Geral das Áreas

| Área | Descrição | Principais Rotas (Next.js) | Principais Entidades do DB | Observações |
|------|-----------|---------------------------|----------------------------|-------------|
| **Landing / SEO (Público)** | Página institucional, captura de leads e redirecionamento para login/registro. | `/` (Home), `/sobre`, `/contato` | — | Usa SEO meta tags e OpenGraph. |
| **Auth** | Fluxo de autenticação (login, signup, reset de senha). | `/auth/login`, `/auth/signup`, `/auth/forgot-password` | `users` (credenciais) | Integração com JWT (backend `/auth`). |
| **Dashboard Cliente** | Área logada do cliente para buscar profissionais, visualizar agenda e agendar sessões. | `/app/client`, `/app/client/profissionais`, `/app/client/profissional/[id]`, `/app/client/agendamentos` | `client_profiles`, `professional_profiles`, `availability_rules`, `appointments`, `payments` | Consome APIs: `/api/v1/availability`, `/api/v1/appointments`, `/api/v1/payments`. |
| **Dashboard Profissional** | Área logada do profissional para gerenciar disponibilidade, agenda e finanças. | `/app/professional`, `/app/professional/disponibilidade`, `/app/professional/agendamentos`, `/app/professional/financeiro` | `professional_profiles`, `availability_rules`, `availability_exceptions`, `appointments`, `payments` | Integração direta com APIs de disponibilidade e pagamentos. |
| **Disponibilidade** | Visualização e edição de regras semanais e exceções. | `/app/professional/disponibilidade` (sub‑rota) | `availability_rules`, `availability_exceptions` | UI de calendário com slots gerados a partir das regras. |
| **Agendamento** | Tela de criação/visualização de um agendamento. | `/app/client/agendamento/[professionalId]` | `appointments` (status `pending_payment` → `confirmed`) | Fluxo inclui chamada a `/api/v1/payments` para pagamento. |
| **Pagamento** | Checkout para concluir pagamento da sessão. | `/app/client/pagamento/[appointmentId]` | `payments` | Integração com gateway externo (Pix, credit‑card). |
| **Perfil** | Edição de dados de usuário (cliente ou profissional). | `/app/client/perfil`, `/app/professional/perfil` | `client_profiles`, `professional_profiles` | Permite upload de foto, atualização de bio, especialidades, etc. |
| **Notificações** *(fase posterior)* | Exibição de push/web notifications. | `/app/notifications` | `device_tokens` | Preparado para futuro uso de Service Workers. |

## 2. Decisões de Design

1. **Rotas no App Router** – Utilizamos o **App Router** de Next.js (pasta `app/`) para aproveitar a segmentação de layout e carregamento incremental.
2. **Proteção de Rotas** – Todas as rotas dentro `/app/` são protegidas por um **middleware** que valida o JWT e redireciona para `/auth/login` caso ausente/expirado.
3. **Data Fetching** – Preferimos **Server‑Components** para páginas que não requerem interatividade imediata (ex.: listagem de profissionais) e **Client‑Components** com SWR/React‑Query para dados mutáveis (ex.: agenda em tempo real).
4. **Styling** – Tailwind CSS com tema dark/light baseado em `prefers-color-scheme`. Variáveis de cor definidas em `globals.css`.
5. **SEO** – Cada página pública preenche `<title>`, `<meta name="description">` e `og:*` via `next/head`.

## 3. Mapeamento de API (OpenAPI)

| Página | Endpoint(s) da API utilizados | Método(s) |
|--------|------------------------------|-----------|
| Landing | — | — |
| Auth (login) | `POST /api/v1/auth/login` | POST |
| Auth (signup) | `POST /api/v1/auth/signup` | POST |
| Busca de Profissionais | `GET /api/v1/professionals` (com filtros) | GET |
| Disponibilidade | `GET /api/v1/availability/:professionalId`, `POST /api/v1/availability/rules`, `POST /api/v1/availability/exceptions` | GET/POST |
| Agendamento | `POST /api/v1/appointments`, `GET /api/v1/appointments/:id` | POST/GET |
| Pagamento | `POST /api/v1/payments`, `GET /api/v1/payments/:id` | POST/GET |
| Perfil Cliente | `GET/PUT /api/v1/clients/:id` | GET/PUT |
| Perfil Profissional | `GET/PUT /api/v1/professionals/:id` | GET/PUT |

> **Nota (2026-11-01):** o [ADR 0012](0012-dados-pessoais-do-cliente.md) substitui `GET/PUT /api/v1/clients/:id` por `GET/PUT /api/v1/clients/me/profile`, derivado do token (sem id na URL, para eliminar IDOR). A página `/app/client/perfil` continua sendo a mesma.

## 4. Próximos Passos

- Implementar **middleware de autenticação** (já presente em `frontend/middleware.ts`).
- Criar **componentes de UI** reutilizáveis (Card, Table, Calendar) sob `frontend/components/`.
- Definir **testes end‑to‑end** com Cypress ou Playwright para fluxos críticos (login → agendamento → pagamento).

---
*Este ADR serve como guia de referência para o time de frontend ao construir as páginas e garantir alinhamento com o modelo de dados já definido.*
