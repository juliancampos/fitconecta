# ADR 0003: Armazenamento de Mídias dos Profissionais (Avatar e Galeria)

* **Status:** Aceito
* **Data:** 2026-10-14
* **Decisores:** Time de Arquitetura e Engenharia FitConecta+
* **Contexto Técnico:** Go (Core API), Next.js (Web), PostgreSQL 16+, Object Storage S3-compatível

---

## 1. Contexto e Problema

O perfil público do profissional (ver ADR 0002) exibia um avatar provisório (a "letra em um círculo") e não permitia o envio de imagens reais. Para transmitir confiança e melhorar a conversão, o profissional precisa de:

1. **Foto de perfil (avatar):** uma imagem que o represente, exibida nos cards de busca, na página de perfil e na área do profissional.
2. **Galeria de fotos:** até **3 imagens** do profissional em ação (ex.: antes/depois, ambiente de treino), exibidas no perfil público.

Restrições e requisitos técnicos:

- **Baixo custo e previsibilidade:** o volume de imagens cresce com o número de profissionais; o custo de **egress** (saída de dados) é o principal vilão em provedores tradicionais.
- **Não sobrecarregar a Core API:** uploads de arquivos binários não devem trafegar pelo servidor Go.
- **Paridade local/produção:** o ambiente de desenvolvimento deve rodar sem custo e sem dependência de contas em nuvem, mas usar a **mesma interface** da produção.
- **Segurança:** apenas o próprio profissional autenticado pode enviar/remover suas mídias; uploads não podem ficar abertos.
- **Simplicidade operacional:** evitar introduzir um servidor de arquivos com estado próprio a ser gerenciado.

---

## 2. Decisão Arquitetural

Decidimos usar **Object Storage compatível com a API S3** e realizar o upload por **URL pré-assinada (`presigned PUT`)**, com os seguintes pontos:

### 2.1 Provedor em Produção: Cloudflare R2
- **Cloudflare R2** é compatível com a API S3 e **não cobra egress** (saída de dados), apenas armazenamento e operações — adequado ao crescimento de leitura de imagens.
- O bucket é servido por um **domínio público** (custom domain ou `r2.dev`) configurado no Cloudflare.
- A **política de CORS** do bucket é configurada no painel do R2 para permitir `PUT` a partir do domínio do frontend.

### 2.2 Desenvolvimento Local: armazenamento S3-compatível auto-hospedado
- Localmente usamos um servidor S3-compatível no `docker-compose`, exposto em `localhost:9000` (API) e `localhost:9001` (console).
- **Nota importante (2026):** a MinIO removeu as imagens públicas `minio/minio` e `minio/mc` dos registries públicos (Docker Hub e, em seguida, Quay). Como contornamos isso:
  - Usamos o **[PGSTY Silo](https://silo.pgsty.com)** (`pgsty/silo`), um fork AGPL da MinIO Community Edition, **drop-in** (mesmos comandos, portas, variáveis `MINIO_*` e API S3).
  - Alternativas avaliadas: `chrislusf/seaweedfs` (SeaweedFS, maduro) e `localstack/localstack`. Optamos pelo Silo por ser o menor atrito (substituição direta do MinIO).
- O backend provisiona o bucket automaticamente (`EnsureBucket`) e aplica leitura pública **best-effort** (no R2 o acesso público é definido no painel).

### 2.3 Fluxo de Upload (upload direto, sem passar pela API)
1. O frontend (autenticado por cookie) chama `POST /api/v1/professionals/me/media/presign` com `{ kind: "avatar" | "photo", content_type }`.
2. A Core API valida o tipo de imagem, monta a chave do objeto e devolve `upload_url` (pré-assinada), `public_url`, `key` e `expires_in`.
3. O frontend faz `PUT upload_url` **diretamente no storage**, sem cookies.
4. O frontend registra a mídia na API: `PUT /me/avatar` (avatar) ou `POST /me/photos` (galeria), enviando a `public_url`.

### 2.4 Convenção de Chaves de Objeto
```
professionals/{professional_id}/avatar/{uuid}.{ext}
professionals/{professional_id}/photos/{uuid}.{ext}
```
Extensões aceitas: `.jpg`, `.png`, `.webp`, `.gif`.

### 2.5 Abstração no Backend
- Pacote `internal/platform/storage`: encapsula o cliente (`minio-go`) e expõe `PresignPut`, `PublicURL`, `KeyFromURL`, `DeleteObject`, `EnsureBucket`, `ExpiresInSeconds`.
- O handler depende de uma interface estreita (`MediaStore`), permitindo trocar R2/Silo/S3/AWS sem alterar as regras de negócio.

---

## 3. Alterações no Banco de Dados

```sql
-- Coluna de avatar no perfil do profissional
ALTER TABLE professional_profiles
    ADD COLUMN IF NOT EXISTS avatar_url TEXT;

-- Galeria de fotos (ate 3 por profissional, controlado na camada de aplicacao)
CREATE TABLE IF NOT EXISTS professional_photos (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    professional_id UUID NOT NULL REFERENCES professional_profiles(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    position SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_professional_photos_prof
    ON professional_photos(professional_id, position);
```

Migração versionada: `infra/docker/postgres/migrations/0002_professional_media.sql` (idempotente). O `init.sql` também foi atualizado para novas bases.

O limite de **3 fotos** é aplicado na camada de serviço (`professionals.Service.AddMyPhoto` → `MaxPhotos = 3`), retornando **400** ao exceder.

---

## 4. Endpoints da API (área do profissional)

| Método | Rota | Descrição |
|---|---|---|
| `POST` | `/api/v1/professionals/me/media/presign` | Gera URL pré-assinada (`kind`: `avatar`\|`photo`) |
| `PUT` | `/api/v1/professionals/me/avatar` | Define a foto de perfil (`avatar_url`) |
| `GET` | `/api/v1/professionals/me/photos` | Lista a galeria |
| `POST` | `/api/v1/professionals/me/photos` | Adiciona uma foto à galeria |
| `DELETE` | `/api/v1/professionals/me/photos/{id}` | Remove uma foto |

Todas exigem papel `professional` (middleware `RequireRole("professional")`).

---

## 5. Configuração (variáveis de ambiente)

| Variável | Padrão (dev) | Produção (R2) |
|---|---|---|
| `STORAGE_ENDPOINT` | `localhost:9000` | `<account>.r2.cloudflarestorage.com` |
| `STORAGE_REGION` | `us-east-1` | `auto` |
| `STORAGE_ACCESS_KEY_ID` | `fitconecta_minio` | token de acesso R2 |
| `STORAGE_SECRET_ACCESS_KEY` | `fitconecta_minio_secret` | secret do R2 |
| `STORAGE_BUCKET` | `fitconecta-media` | nome do bucket R2 |
| `STORAGE_PUBLIC_BASE_URL` | `http://localhost:9000/fitconecta-media` | `https://media.seudominio.com` |
| `STORAGE_USE_SSL` | `false` | `true` |
| `STORAGE_PATH_STYLE` | `true` | `false` |
| `STORAGE_PRESIGN_MINUTES` | `15` | `15` |

No frontend, as imagens remotas são permitidas via `next.config.ts` → `images.remotePatterns` (host local `localhost:9000`/`127.0.0.1:9000` e um coringa `https://**` para o domínio público do R2). Como o storage local roda em IP privado, o Next.js 16 exige `images.dangerouslyAllowLocalIP: true`; em produção (R2 via HTTPS) esse flag não é necessário, mas permanece inócuo porque os `remotePatterns` restringem os hosts permitidos.

---

## 6. Consequências

### Positivas
- **Custo de egress zero** em produção (R2) — leitura de imagens não gera cobrança de saída.
- **Core API leve:** o upload binário vai do navegador direto ao storage; a API apenas assina e registra metadados.
- **Paridade local/produção:** mesma interface S3 nos dois ambientes; apenas variáveis de ambiente mudam.
- **Segurança:** URLs pré-assinadas expiram (`STORAGE_PRESIGN_MINUTES`); apenas o dono gerencia suas mídias; tipos de imagem validados no servidor.
- **Sem servidor de arquivos com estado:** menos um componente para operar e escalar.

### Pontos de Atenção & Mitigações
- **Distribuição do MinIO em 2026:** as imagens oficiais saíram dos registries públicos; usamos o fork `pgsty/silo` no compose. Se ele também for descontinuado, a abstração `MediaStore` permite migrar rapidamente para SeaweedFS/S3/AWS.
- **CORS:** o bucket (R2) precisa de CORS liberando `PUT` do domínio do frontend; localmente, o Silo usa `MINIO_API_CORS_ALLOW_ORIGIN`.
- **Orfandade de objetos:** ao trocar avatar/remover foto, o objeto antigo pode permanecer no bucket. Mitigação futura: usar `KeyFromURL` + `DeleteObject` na remoção e/ou um job de limpeza.
- **Validação de conteúdo:** a assinatura garante o `content_type` declarado no momento do presign; validação de conteúdo real (magic bytes) pode ser adicionada depois.
- **Acesso público de leitura:** imagens de perfil são públicas por natureza; o bucket é público somente para leitura (`GetObject`).
