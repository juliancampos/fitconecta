import { defineConfig } from "@neon/config/v1";

/**
 * Configuracao declarativa do backend Neon (infra como codigo).
 *
 *   npm install @neon/config
 *   neon link --project-id square-queen-83183876 --branch production
 *   neon deploy
 *
 * `neon deploy` provisiona o que esta declarado aqui e grava no `.env` as
 * variaveis injetadas do Object Storage:
 * AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_ENDPOINT_URL_S3, AWS_REGION.
 * O Postgres nao precisa ser declarado: toda branch nasce com ele e o
 * DATABASE_URL ja e injetado.
 *
 * A core-api (Go) le essas variaveis em
 * backend/core-api/internal/platform/config (applyNeonStorage).
 */
export default defineConfig({
  // Buckets de midia dos profissionais (avatar + galeria).
  // public_read: os <img src={avatar_url}> do frontend leem sem credencial.
  // O nivel de acesso NAO muda pela API S3 (PutBucketAcl/PutBucketPolicy
  // respondem 501); so via Console ou aqui, apos `neon deploy`.
  buckets: {
    "fitconecta-images": { access: "public_read" },
  },
});
