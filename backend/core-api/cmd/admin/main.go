package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"fitconecta/core-api/internal/platform/config"
	"fitconecta/core-api/internal/platform/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	action := flag.String("action", "create", "create | reset-password | disable | enable")
	email := flag.String("email", "", "e-mail do administrador")
	password := flag.String("password", "", "senha (ou use ADMIN_PASSWORD)")
	reason := flag.String("reason", "alteracao via CLI", "motivo para disable")
	flag.Parse()

	_ = godotenv.Load()
	_ = godotenv.Load(".env.local")

	*email = strings.TrimSpace(strings.ToLower(*email))
	if *email == "" {
		log.Fatal("informe -email")
	}
	if *password == "" {
		*password = os.Getenv("ADMIN_PASSWORD")
	}
	if *action == "create" || *action == "reset-password" {
		if len(*password) < 8 {
			log.Fatal("a senha deve ter pelo menos 8 caracteres (use -password ou ADMIN_PASSWORD)")
		}
	}

	cfg := config.Load()
	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("erro ao conectar no banco: %v", err)
	}
	defer pool.Close()

	switch *action {
	case "create":
		hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("erro ao gerar hash: %v", err)
		}
		var id uuid.UUID
		var role string
		err = pool.QueryRow(ctx, `SELECT id, role::text FROM users WHERE email = $1`, *email).Scan(&id, &role)
		if err == nil {
			if role != "admin" {
				log.Fatalf("o e-mail %s ja pertence a um usuario %s", *email, role)
			}
			log.Fatalf("administrador ja existe (id %s); use -action reset-password", id)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			log.Fatalf("erro ao verificar e-mail: %v", err)
		}
		err = pool.QueryRow(ctx, `
			INSERT INTO users (email, password_hash, role, status)
			VALUES ($1, $2, 'admin', 'active')
			RETURNING id`, *email, string(hash)).Scan(&id)
		if err != nil {
			log.Fatalf("erro ao criar administrador: %v", err)
		}
		fmt.Printf("administrador criado: %s (%s)\n", *email, id)

	case "reset-password":
		hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("erro ao gerar hash: %v", err)
		}
		tag, err := pool.Exec(ctx, `
			UPDATE users SET password_hash = $1, updated_at = NOW()
			WHERE email = $2 AND role = 'admin'`, string(hash), *email)
		if err != nil {
			log.Fatalf("erro ao redefinir senha: %v", err)
		}
		if tag.RowsAffected() == 0 {
			log.Fatalf("administrador nao encontrado: %s", *email)
		}
		fmt.Printf("senha redefinida: %s\n", *email)

	case "disable":
		tag, err := pool.Exec(ctx, `
			UPDATE users
			SET status = 'inactive', status_reason = $1, status_changed_at = NOW(), updated_at = NOW()
			WHERE email = $2 AND role = 'admin'`, *reason, *email)
		if err != nil {
			log.Fatalf("erro ao desativar administrador: %v", err)
		}
		if tag.RowsAffected() == 0 {
			log.Fatalf("administrador nao encontrado: %s", *email)
		}
		fmt.Printf("administrador desativado: %s\n", *email)

	case "enable":
		tag, err := pool.Exec(ctx, `
			UPDATE users
			SET status = 'active', status_reason = NULL, status_changed_at = NOW(), updated_at = NOW()
			WHERE email = $1 AND role = 'admin'`, *email)
		if err != nil {
			log.Fatalf("erro ao ativar administrador: %v", err)
		}
		if tag.RowsAffected() == 0 {
			log.Fatalf("administrador nao encontrado: %s", *email)
		}
		fmt.Printf("administrador ativado: %s\n", *email)

	default:
		log.Fatalf("acao desconhecida: %s", *action)
	}
}
