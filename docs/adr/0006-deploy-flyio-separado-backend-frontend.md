# ADR 0006 - Deploy Separado do Backend e Frontend no Fly.io

## Status
Proposto

## Data
2026-10-16

## Contexto

O projeto FitConecta+ é composto por dois serviços independentes: `backend/core-api` (Go) e `frontend` (Next.js 16). Atualmente a aplicação roda localmente via `docker compose` (infra/local) com banco PostgreSQL externo (Neon) e armazenamento de objetos (Neon Object Storage, com MinIO mantido como alternativa).

A decisão é realizar o deploy dos serviços separadamente no Fly.io, mantendo:
- Backend: API HTTP pública (REST) responsável por autenticação, regras de negócio, disponibilidade, agendamentos e media presign.
- Frontend: aplicação Next.js (standalone) servida pelo próprio Next.js (output: "standalone").
- Banco de dados: Neon PostgreSQL (managed, já utilizado).
- Armazenamento: Neon Object Storage (S3-compatível, path-style + SigV4).

## Decisão

1. **Deploy separado (dois apps no Fly.io)**
   - `fitconecta-core-api` (app backend) — expõe a API em `https://<core-api>.fly.dev` (porta 8080)
   - `fitconecta-web` (app frontend) — expõe o site em `https://<web>.fly.dev` (porta 3000 via `next start`)
   Ambos com domínios próprios. O frontend faz chamadas para `NEXT_PUBLIC_API_URL` apontando para o backend.

2. **Arquitetura de rede/ambiente**
   - CORS: configurar `ALLOWED_ORIGINS` no backend para incluir `https://<web>.fly.dev` (e eventuais domínios customizados).
   - Cookies: em produção usar `COOKIE_SECURE=true` (HTTPS). SameSite=Strict já adotado.
   - Storage: utilizar Neon Object Storage (provider `neon`) via variáveis `AWS_*`/`STORAGE_*`. Bucket `fitconecta-images` com `access: public_read`. O CORS do bucket deve permitir origem `https://<web>.fly.dev`.
   - Banco: `DATABASE_URL` do Neon com `sslmode=require` e `default_query_exec_mode=simple_protocol` (PgBouncer).
   - Migrações/schema: aplicar `infra/docker/postgres/init.sql` e migrations 0002–0008 no Neon (já aplicados). Não executar DDL no deploy do backend — DDL controlado manualmente.

3. **Configuração de runtime**
   - **Backend (Go)**: multi-stage Dockerfile (`backend/core-api/Dockerfile`). Rodar com `ENV=production`, `PORT=8080`. Healthcheck via `/health`. Não expor storage local.
   - **Frontend (Next.js)**: multi-stage Dockerfile (`frontend/Dockerfile`) com `output: "standalone"`. Build recebe `NEXT_PUBLIC_API_URL` (injectado no build). Rodar `node server.js` na porta 3000.
   - **fly.toml** por app (ou via `fly launch --dockerfile ... --app ...`), com `internal_port`, `force_https=true`, healthchecks adequados.
   - Secrets: `JWT_SECRET`, `DATABASE_URL`, credenciais de storage (`AWS_*`/`STORAGE_*`), `ALLOWED_ORIGINS` gerenciados via `fly secrets set`.

4. **Fluxo de deploy (CLI Fly)**
   - Instalar `flyctl` e autenticar (`fly auth login`).
   - Criar apps: `fly apps create fitconecta-core-api` e `fly apps create fitconecta-web`.
   - Backend: `cd backend/core-api && fly launch --dockerfile Dockerfile --app fitconecta-core-api --no-deploy` (ou configurar `fly.toml`), definir secrets/vars, `fly deploy`.
   - Frontend: `cd frontend && fly launch --dockerfile Dockerfile --app fitconecta-web --no-deploy`, definir `NEXT_PUBLIC_API_URL=https://fitconecta-core-api.fly.dev`, `fly deploy`.
   - Atualizações: `fly deploy` por diretório (ou CI futura).

5. **Observabilidade/segurança**
   - Logs via `fly logs`. Healthchecks em ambos apps.
   - HTTPS forçado. CORS restritivo (sem `*`).
   - Secrets nunca commitados. `.env`/`.env.*` gitignored.
   - Redirecionamentos HTTP→HTTPS.

## Consequências

- **Vantagens**
  - Deploys independentes (frontend/backend), rollback isolado, escalonamento independente.
  - URLs estáveis por app. Integração limpa entre serviços.
  - Reaproveita Neon (DB+Storage) já configurado; evita rodar Postgres/MinIO no Fly.
  - Frontend Next.js standalone eficiente no Fly (node server).

- **Desvantagens**
  - Necessário gerenciar dois apps + secrets + CORS + `NEXT_PUBLIC_API_URL`.
  - Cold starts possíveis no Fly (Free/Standard); pode ajustar `min_machines_running`.
  - CORS do bucket Neon precisa incluir origem frontend (já suportado via `PutBucketCors`/console).
  - `NEXT_PUBLIC_API_URL` é embutido no build do frontend (mudança exige rebuild/deploy).

- **Mitigações**
  - Usar `fly.toml` versionado por app.
  - Documentar secrets necessários em `infra/flyio/` (não commitar valores).
  - Validar CORS/health após primeiro deploy.
  - Considerar custom domain posteriormente.

## Plano de Ação (prático)

1. `fly auth whoami` / `fly auth login` (se necessário)
2. Criar apps: `fly apps create fitconecta-core-api` e `fly apps create fitconecta-web`
3. Backend: configurar `fly.toml` em `backend/core-api/` (porta 8080, internal_port 8080, `[[services]]`, health `/health`)
4. Definir secrets backend: `fly secrets set --app fitconecta-core-api ...` (JWT_SECRET, DATABASE_URL, STORAGE_PROVIDER=neon, AWS_*, STORAGE_BUCKET, ALLOWED_ORIGINS, COOKIE_SECURE=true, ENV=production)
5. `cd backend/core-api && fly deploy`
6. Frontend: `fly.toml` em `frontend/` com `NEXT_PUBLIC_API_URL=https://fitconecta-core-api.fly.dev`
7. `cd frontend && fly deploy`
8. Validar `/health`, CORS, login, presign upload, agendamento ponta-a-ponta.

## Referências
- Fly.io docs (apps, toml, secrets, docker)
- Fly.io Next.js standalone
- Neon (Postgres + Object Storage)
- ADRs existentes (0001–0005)
