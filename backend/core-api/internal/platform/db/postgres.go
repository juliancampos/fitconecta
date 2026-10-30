package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool cria e configura um pool de conexoes pgxpool para o PostgreSQL.
func NewPool(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("erro ao analisar string de conexao postgres: %w", err)
	}

	// Pool enxuto, adequado a bancos serverless (ex.: Neon), que limitam o
	// numero de conexoes simultaneas e cobram por tempo de conexao ativa.
	config.MaxConns = 10
	config.MinConns = 1
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("erro ao instanciar pool pgx: %w", err)
	}

	// Ping com timeout para verificar disponibilidade imediata
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("falha ao conectar no postgres: %w", err)
	}

	return pool, nil
}
