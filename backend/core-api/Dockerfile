# Dockerfile do FitConecta+ Core API (Go)
# Build multi-stage: compila um binario estatico e o executa numa imagem minima.

# --- Estagio de build ---
FROM golang:1.26-alpine AS build

WORKDIR /src

# Baixa as dependencias primeiro para aproveitar o cache de camadas do Docker.
COPY go.mod go.sum ./
RUN go mod download

# Copia o codigo-fonte e compila um binario estatico (sem CGO).
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath -ldflags="-s -w" \
    -o /out/core-api ./cmd/api

# --- Estagio de runtime ---
FROM alpine:3.21 AS runtime

# ca-certificates: TLS para o storage S3/R2; tzdata: fuso horario;
# wget: usado pelo HEALTHCHECK.
RUN apk add --no-cache ca-certificates tzdata wget \
    && adduser -D -u 10001 appuser

ENV TZ=UTC

WORKDIR /app
COPY --from=build /out/core-api /app/core-api
COPY --from=build /src/.env /app/.env

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=5 \
  CMD wget -q -O - http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["/app/core-api"]
