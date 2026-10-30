# Guia rápido: Iniciando o serviço FitConecta+ Core API

## Pré‑requisitos
- Go 1.26+ instalado
- Banco PostgreSQL em execução (pode ser via Docker)
- Variáveis de ambiente configuradas (`DATABASE_URL`, `JWT_SECRET`, `ADMIN_JWT_SECRET`, `PORT`)

## Passos
1. **Instalar dependências**
```bash
cd /Users/julianoliveiracampos/projetos/fitconecta/backend/core-api
go mod tidy
```
2. **Iniciar o banco (exemplo Docker)**
```bash
docker run -d \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=fitconecta \
  -p 5432:5432 \
  postgres:16
```
```bash
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/fitconecta?sslmode=disable
```
3. **Aplicar o schema (banco novo)**
```bash
psql "$DATABASE_URL" -f infra/docker/postgres/init.sql
for file in infra/docker/postgres/migrations/*.sql; do
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$file"
done
```
O `init.sql` cria o schema base; as migrations incrementais são idempotentes. Em um banco já existente, aplique apenas as migrations pendentes. As mais recentes são `0009_admin_panel.sql` (painel admin) e `0010_pix_payment.sql` (pagamento Pix obrigatório).

4. **Executar a API**
```bash
go run ./cmd/api
```
O servidor será iniciado na porta configurada (default `8080`).

5. **Checar saúde**
```bash
curl http://localhost:${PORT:-8080}/health
```
Resposta esperada:
```json
{"status":"ok","app":"FitConecta+ Core API",...}
```

## Opcional – modo production
```bash
go build -o core-api ./cmd/api
./core-api
```

## Variáveis de ambiente relevantes

| Variável | Padrão | Descrição |
|---|---|---|
| `PORT` | `8080` | Porta da Core API |
| `DATABASE_URL` | `postgres://fitconecta_user:fitconecta_password@localhost:5433/fitconecta?sslmode=disable` | Conexão com o Postgres local (docker-compose) |
| `JWT_SECRET` | valor de dev | Chave de assinatura do JWT público |
| `ADMIN_JWT_SECRET` | valor de dev | Chave de assinatura exclusiva do painel admin. **Use um valor diferente de `JWT_SECRET` em produção** |
| `JWT_EXPIRATION_HOURS` | `24` | Validade do token público |
| `ADMIN_JWT_EXPIRATION_HOURS` | `8` | Validade do token administrativo |
| `ALLOWED_ORIGINS` | `http://localhost:3000,http://127.0.0.1:3000` | Origens CORS (separadas por vírgula). **Não usar `*`** com cookies/credenciais |
| `COOKIE_SECURE` | `false` | Defina `true` em produção (HTTPS) para os cookies `fitconecta_jwt` e `fitconecta_admin_jwt` |
| `PIX_PROVIDER` | `mock` | Provedor Pix. `mock` apenas para desenvolvimento/testes; use o PSP real em produção |
| `PIX_WEBHOOK_SECRET` | valor de dev | Segredo usado para validar a assinatura do webhook Pix |
| `PIX_EXPIRATION_MINUTES` | `10` | Tempo de pré-reserva para o cliente pagar |
| `PIX_GRACE_MINUTES` | `2` | Carência antes de expirar a pré-reserva (pagamento tardio vira reembolso pendente) |
| `STORAGE_ENDPOINT` | `localhost:9000` | Endpoint S3-compatível (MinIO/Silo local ou `<account>.r2.cloudflarestorage.com` em produção) |
| `STORAGE_REGION` | `us-east-1` | Região do storage (`auto` no Cloudflare R2) |
| `STORAGE_ACCESS_KEY_ID` | `fitconecta_minio` | Access key do storage |
| `STORAGE_SECRET_ACCESS_KEY` | `fitconecta_minio_secret` | Secret key do storage |
| `STORAGE_BUCKET` | `fitconecta-media` | Nome do bucket de mídias |
| `STORAGE_PUBLIC_BASE_URL` | `http://localhost:9000/fitconecta-media` | Base pública de leitura das imagens |
| `STORAGE_USE_SSL` | `false` | `true` em produção (HTTPS) |
| `STORAGE_PATH_STYLE` | `true` | `false` no Cloudflare R2 (virtual-host) |
| `STORAGE_PRESIGN_MINUTES` | `15` | Validade das URLs pré-assinadas de upload |

> **Mídias (avatar e galeria):** o upload é feito por **URL pré-assinada** direto ao storage, sem passar pela API. Veja o ADR 0003 para o desenho completo.

### Pagamento Pix local (provedor mock)

Com `PIX_PROVIDER=mock`, o QR Code e o copia e cola são válidos apenas para teste e nenhum dinheiro é liquidado. A confirmação é simulada enviando um **webhook assinado**, como o PSP faria; não existe botão no frontend para o cliente confirmar o pagamento.

1. Faça login (guarde o cookie de sessão) e crie um agendamento pela tela ou pela API:

```bash
curl -c cookies.txt -X POST http://localhost:3000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"SEU_EMAIL","password":"SUA_SENHA"}'
```

2. Gere/reutilize a cobrança Pix e copie o `provider_payment_id` e o `amount_cents` do retorno:

```bash
curl -b cookies.txt -X POST http://localhost:3000/api/v1/appointments/APPOINTMENT_ID/payment \
  -H "Content-Type: application/json" -d '{}'
```

3. Monte o evento e assine o corpo com o `PIX_WEBHOOK_SECRET` (o padrão local é `fitconecta_dev_pix_webhook_secret`):

```bash
BODY='{"event_id":"evt_local_1","type":"payment.updated","provider_payment_id":"mock_xxx","status":"paid","amount_cents":15000}'
SIG=$(printf %s "$BODY" | openssl dgst -sha256 -hmac "${PIX_WEBHOOK_SECRET:-fitconecta_dev_pix_webhook_secret}" -hex | sed 's/^.* //')

curl -X POST http://localhost:3000/api/v1/payments/webhook/mock \
  -H "Content-Type: application/json" -H "X-Mock-Signature: $SIG" -d "$BODY"
```

4. Confira o resultado:

```bash
curl -b cookies.txt http://localhost:3000/api/v1/appointments/APPOINTMENT_ID
```

O agendamento deve mudar de `pending_payment` para `confirmed` e o pagamento para `paid`. Use um `event_id` novo para cada nova simulação; repetir o mesmo `event_id` é idempotente e não altera o resultado. Também é possível chamar o webhook direto na API em `http://localhost:8080` em vez do proxy do frontend.

O mock mantém as cobranças em memória: reiniciar a `core-api` apaga esse estado, então gere uma nova cobrança antes de simular o webhook. A pré-reserva expira em 10 minutos (com 2 minutos de carência); um webhook de pagamento recebido após a expiração marca o pagamento como `refund_pending`.

### Armazenamento de mídia local (S3-compatível)

Suba o serviço de objetos (o bucket é criado automaticamente pela API na inicialização):
```bash
cd infra/local
docker compose up -d minio
```
- API S3: `http://localhost:9000`
- Console web: `http://localhost:9001` (usuário/senha padrão `fitconecta_minio` / `fitconecta_minio_secret`)

> **Nota (2026):** usamos o [PGSTY Silo](https://silo.pgsty.com) (`pgsty/silo`), um fork S3-compatível da MinIO Community Edition, porque a MinIO removeu as imagens públicas `minio/minio` dos registries. Em produção, use **Cloudflare R2** — basta ajustar as variáveis `STORAGE_*`.

> **Sessão web:** `POST /api/v1/auth/login` e `/auth/register` devolvem o JWT no corpo **e** gravam um cookie `HttpOnly` (`fitconecta_jwt`, `SameSite=Strict`). `POST /api/v1/auth/logout` expira esse cookie.

### Frontend

Crie `frontend/.env.local`:
```
BACKEND_PROXY_URL=http://localhost:8080
```
O `proxy.ts` tem três funções: (1) protege as rotas `/app/*` verificando a presença do cookie `fitconecta_jwt`; (2) protege `/admin/*` verificando o cookie `fitconecta_admin_jwt` e libera `/admin/login`; (3) reescreve `/api/*` (incluindo `/api/v1/admin/*`) para `BACKEND_PROXY_URL`, mantendo o navegador no mesmo host (same-origin) — por isso o frontend consome a API por caminhos relativos e os cookies de sessão funcionam. Em produção no Fly.io, defina `BACKEND_PROXY_URL` como secret de runtime (sem barra final).

### Painel administrativo

O painel fica no **mesmo frontend**, sob `/admin` (não há app ou serviço separado):

- Login: `http://localhost:3000/admin/login`
- Agendamentos: `http://localhost:3000/admin/agendamentos`
- Pagamentos: `http://localhost:3000/admin/pagamentos`
- Usuários: `http://localhost:3000/admin/usuarios`

Inicie apenas a API e o frontend (`npm run dev` em `frontend/`). A tela `/admin/login` é exclusiva do painel e usa cookie próprio `fitconecta_admin_jwt`, segredo `ADMIN_JWT_SECRET` e audience `fitconecta-admin`. O proxy do frontend protege `/admin/*` e encaminha `/api/v1/admin/*` para a Core API.

Crie o primeiro administrador (não há cadastro público):
```bash
cd backend/core-api
DATABASE_URL="postgres://..." go run ./cmd/admin \
  -action create -email admin@fitconecta.com.br -password "senha-forte"
```
Outras ações: `-action reset-password`, `-action disable` e `-action enable`. A senha também pode vir de `ADMIN_PASSWORD`.

Endpoints principais: `POST /api/v1/admin/auth/login`, `GET /api/v1/admin/appointments`, `POST /api/v1/admin/appointments/{id}/cancel`, `GET /api/v1/admin/payments`, `GET /api/v1/admin/payments/summary`, `GET /api/v1/admin/users` e `PATCH /api/v1/admin/users/{id}/status`.

---
**Dicas**
- Alterar a porta: `export PORT=9090` antes de iniciar.
- Use ferramentas como `air` para reload automático durante desenvolvimento.
- Consulte `openapi.yaml` para detalhes dos endpoints.
