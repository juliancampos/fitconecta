FROM postgres:16-alpine

# Definir fuso horario UTC por padrao
ENV TZ=UTC

# Copiar script SQL de inicializacao para o diretorio padrao do Postgres
# Qualquer script .sql neste diretorio e executado na criacao do banco de dados
COPY init.sql /docker-entrypoint-initdb.d/01-init.sql

# Porta padrao do PostgreSQL
EXPOSE 5432

# Healthcheck para monitorar o status do banco
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=5 \
  CMD pg_isready -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-fitconecta}" || exit 1
