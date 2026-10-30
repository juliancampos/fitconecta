#!/usr/bin/env bash
# Aplica um arquivo .sql no banco informado.
# Uso:
#   ./scripts/apply-sql.sh "postgres://usuario:senha@host:5432/banco?sslmode=require" infra/docker/postgres/migrations/0011_professional_services.sql
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "Uso: $0 \"<string-de-conexao>\" <arquivo.sql>" >&2
  exit 1
fi

DATABASE_URL="$1"
SQL_FILE="$2"

if [[ ! -f "$SQL_FILE" ]]; then
  echo "Arquivo nao encontrado: $SQL_FILE" >&2
  exit 1
fi

echo "Aplicando $SQL_FILE ..."
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$SQL_FILE"
echo "Concluido."
