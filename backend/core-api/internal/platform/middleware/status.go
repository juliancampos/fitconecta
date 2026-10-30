package middleware

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgUserStatusReader struct {
	pool *pgxpool.Pool
}

// NewUserStatusReader consulta users.status a cada requisicao autenticada.
func NewUserStatusReader(pool *pgxpool.Pool) UserStatusReader {
	return &pgUserStatusReader{pool: pool}
}

func (r *pgUserStatusReader) GetStatus(ctx context.Context, userID uuid.UUID) (string, error) {
	var status string
	err := r.pool.QueryRow(ctx, `SELECT status::text FROM users WHERE id = $1`, userID).Scan(&status)
	if err != nil {
		return "", err
	}
	return status, nil
}
