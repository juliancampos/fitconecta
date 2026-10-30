package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"fitconecta/core-api/internal/admin"
	"fitconecta/core-api/internal/appointments"
	"fitconecta/core-api/internal/auth"
	"fitconecta/core-api/internal/availability"
	"fitconecta/core-api/internal/payments"
	"fitconecta/core-api/internal/platform/config"
	"fitconecta/core-api/internal/platform/db"
	"fitconecta/core-api/internal/platform/httputil"
	appMiddleware "fitconecta/core-api/internal/platform/middleware"
	"fitconecta/core-api/internal/platform/storage"
	"fitconecta/core-api/internal/professionals"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"

	"github.com/joho/godotenv"
)

func main() {
	// Logger estruturado com slog
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Carregar .env se existir (desenvolvimento)
	_ = godotenv.Load()
	_ = godotenv.Load(".env.local")

	// Carregar configuracoes
	cfg := config.Load()
	logger.Info("Iniciando FitConecta+ Core API",
		"env", cfg.Environment,
		"port", cfg.Port,
		"platform_fee_percent", cfg.PlatformFeePercent,
	)

	// Conectar ao PostgreSQL
	ctx := context.Background()
	dbPool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("Falha critica ao conectar com banco de dados", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()
	logger.Info("Conexao com PostgreSQL estabelecida com sucesso")

	// Middlewares de autenticacao JWT. O painel usa segredo, audience e
	// cookie proprios, pois um token do app publico nunca deve valer no admin.
	statusReader := appMiddleware.NewUserStatusReader(dbPool)
	jwtAuth := appMiddleware.AuthMiddlewareWithStatus(cfg.JWTSecret, statusReader)
	adminAuth := appMiddleware.AdminAuthMiddleware(cfg.AdminJWTSecret, statusReader)

	// Iniciar repositorios e servicos
	authRepo := auth.NewRepository(dbPool)
	authService := auth.NewService(authRepo, cfg.JWTSecret, cfg.JWTExpirationHours)
	authHandler := auth.NewHandler(authService, cfg.CookieSecure)

	profRepo := professionals.NewRepository(dbPool)
	profService := professionals.NewService(profRepo)

	// Origens permitidas (separadas por virgula). Nao usar "*" para que o
	// cookie de sessao (AllowCredentials) funcione: o navegador exige a
	// origem exata refletida.
	allowedOrigins := strings.Split(cfg.AllowedOrigins, ",")
	for i := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
	}

	// Armazenamento de midia (MinIO local / Cloudflare R2 em producao)
	mediaClient, err := storage.New(storage.Options{
		Endpoint:        cfg.StorageEndpoint,
		AccessKeyID:     cfg.StorageAccessKeyID,
		SecretAccessKey: cfg.StorageSecretAccessKey,
		Bucket:          cfg.StorageBucket,
		Region:          cfg.StorageRegion,
		PublicBaseURL:   cfg.StoragePublicBaseURL,
		UseSSL:          cfg.StorageUseSSL,
		PathStyle:       cfg.StoragePathStyle,
		PresignExpiry:   time.Duration(cfg.StoragePresignMinutes) * time.Minute,
	})
	if err != nil {
		logger.Error("Falha ao inicializar o armazenamento de midia", "error", err)
		os.Exit(1)
	}
	if cfg.StorageProvision {
		if err := mediaClient.EnsureBucket(ctx); err != nil {
			// Best-effort: em provedores gerenciados (R2) o acesso publico e
			// configurado fora da API e pode nao ser gerenciavel aqui.
			logger.Warn("Nao foi possivel provisionar o bucket de midia automaticamente", "error", err)
		} else {
			logger.Info("Bucket de midia pronto", "bucket", cfg.StorageBucket)
		}
	} else {
		// Provedor gerenciado (Neon Object Storage): o bucket e o acesso
		// public_read sao definidos no neon.ts / Neon Console, nao via API S3
		// (PutBucketPolicy responde 501). Aqui apenas configuramos o CORS para
		// liberar o PUT pre-assinado feito pelo navegador.
		logger.Info("Storage gerenciado detectado; provisionamento automatico desativado",
			"endpoint", cfg.StorageEndpoint, "bucket", cfg.StorageBucket)
		if err := mediaClient.EnsureCORS(ctx, allowedOrigins, []string{"GET", "PUT", "HEAD"}); err != nil {
			logger.Warn("Nao foi possivel configurar o CORS do bucket de midia", "error", err)
		}
	}

	profHandler := professionals.NewHandler(profService, mediaClient)

	availRepo := availability.NewRepository(dbPool)
	availService := availability.NewService(availRepo)
	availHandler := availability.NewHandler(availService, func(userID uuid.UUID) (uuid.UUID, error) {
		p, err := profRepo.GetByUserID(context.Background(), userID)
		if err != nil {
			return uuid.Nil, err
		}
		return p.ID, nil
	})

	var pixProvider payments.Provider
	switch strings.ToLower(cfg.PixProvider) {
	case "mock":
		if cfg.Environment == "production" {
			logger.Warn("PIX_PROVIDER=mock em producao: cobrancas NAO serao liquidadas de verdade")
		}
		pixProvider = payments.NewMockProvider(cfg.PixWebhookSecret)
	default:
		logger.Error("PIX_PROVIDER nao suportado", "provider", cfg.PixProvider)
		os.Exit(1)
	}

	appRepo := appointments.NewRepository(dbPool)
	appService := appointments.NewServiceWithPayments(
		appRepo,
		cfg.PlatformFeePercent,
		pixProvider,
		time.Duration(cfg.PixExpirationMinutes)*time.Minute,
		time.Duration(cfg.PixGraceMinutes)*time.Minute,
	)
	appHandler := appointments.NewHandler(appService)

	adminRepo := admin.NewRepository(dbPool)
	adminAuthService := admin.NewAuthService(authRepo, cfg.AdminJWTSecret, cfg.AdminJWTExpirationHours)
	adminHandler := admin.NewHandler(adminRepo, adminAuthService, cfg.CookieSecure)

	// Roteador Chi
	r := chi.NewRouter()

	// Middlewares globais
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(appMiddleware.Logger(logger))
	r.Use(chiMiddleware.Recoverer)

	// CORS configurado. As origens permitidas vem do ambiente (ALLOWED_ORIGINS,
	// separadas por virgula). Evitamos "*" para que o cookie de sessao
	// (AllowCredentials) funcione: o navegador exige a origem exata refletida.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Rota de Healthcheck
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		httputil.Success(w, http.StatusOK, map[string]interface{}{
			"status":      "ok",
			"app":         "FitConecta+ Core API",
			"environment": cfg.Environment,
			"timestamp":   time.Now().UTC(),
		})
	})

	// Sub-rotas da API v1
	r.Route("/api/v1", func(api chi.Router) {
		api.Mount("/auth", authHandler.Routes(jwtAuth))
		api.Mount("/professionals", profHandler.Routes(jwtAuth, availHandler.GetSlots))
		api.Mount("/specialties", profHandler.SpecialtiesRoutes())
		api.Mount("/catalogs", profHandler.CatalogsRoutes())
		api.Mount("/neighborhoods", profHandler.NeighborhoodsRoutes())
		api.Mount("/availability", availHandler.RoutesAvailability(jwtAuth))
		api.Mount("/appointments", appHandler.Routes(jwtAuth))
		api.Post("/payments/webhook/"+pixProvider.Name(), appHandler.Webhook(pixProvider))
		api.Mount("/admin", adminHandler.Routes(adminAuth))
	})

	// Servidor HTTP com graceful shutdown
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	jobsCtx, cancelJobs := context.WithCancel(context.Background())
	defer cancelJobs()
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-jobsCtx.Done():
				return
			case <-ticker.C:
				count, err := appService.ExpirePending(jobsCtx)
				if err != nil {
					logger.Error("Falha ao expirar pre-reservas", "error", err)
					continue
				}
				if count > 0 {
					logger.Info("Pre-reservas expiradas", "count", count)
				}
			}
		}
	}()

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info(fmt.Sprintf("Servidor ouvindo na porta :%s", cfg.Port))
		serverErrors <- server.ListenAndServe()
	}()

	// Escutar sinais de desligamento do SO
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Erro fatal no servidor HTTP", "error", err)
		}
	case sig := <-shutdown:
		logger.Info("Sinal de encerramento recebido", "signal", sig.String())
		cancelJobs()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("Erro ao encerrar servidor graciosamente", "error", err)
			_ = server.Close()
		}
		logger.Info("Servidor Core API encerrado com sucesso")
	}
}
