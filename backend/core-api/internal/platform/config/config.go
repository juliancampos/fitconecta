package config

import (
	"os"
	"strconv"
	"strings"
)

// Config armazena as configuracoes da aplicacao carregadas do ambiente.
type Config struct {
	Port                    string
	DatabaseURL             string
	JWTSecret               string
	JWTExpirationHours      int
	AdminJWTSecret          string
	AdminJWTExpirationHours int
	Environment             string
	AllowedOrigins          string
	PlatformFeePercent      int
	CookieSecure            bool

	// Pagamentos Pix (ADR 0010)
	PixProvider          string
	PixWebhookSecret     string
	PixExpirationMinutes int
	PixGraceMinutes      int

	// Armazenamento de midia (S3-compativel: MinIO local / Cloudflare R2 em producao)
	StorageEndpoint        string
	StorageRegion          string
	StorageAccessKeyID     string
	StorageSecretAccessKey string
	StorageBucket          string
	StoragePublicBaseURL   string
	StorageUseSSL          bool
	StoragePathStyle       bool
	StoragePresignMinutes  int
	// StorageProvision indica se a API deve criar/configurar o bucket via API
	// S3. Fica false em provedores gerenciados (Neon Object Storage), onde o
	// bucket e o acesso public_read sao definidos via neon.ts / Neon Console.
	StorageProvision bool
}

// Load carrega as variaveis de ambiente com valores padrao para desenvolvimento local.
func Load() *Config {
	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://fitconecta_user:fitconecta_password@localhost:5433/fitconecta?sslmode=disable"),
		JWTSecret:          getEnv("JWT_SECRET", "fitconecta_dev_secret_key_minimum_32_characters_long_1234"),
		JWTExpirationHours: getEnvAsInt("JWT_EXPIRATION_HOURS", 24),
		// O painel administrativo usa um segredo e um cookie proprios. Nunca
		// reutilize JWT_SECRET em producao: tokens do app publico nao devem
		// ser aceitos no app admin.
		AdminJWTSecret:          getEnv("ADMIN_JWT_SECRET", "fitconecta_dev_admin_secret_key_minimum_32_characters_1234"),
		AdminJWTExpirationHours: getEnvAsInt("ADMIN_JWT_EXPIRATION_HOURS", 8),
		Environment:             getEnv("ENV", "development"),
		// Lista de origens permitidas (separadas por virgula). Nao usar "*"
		// quando AllowCredentials estiver ativo, pois o navegador rejeita
		// Access-Control-Allow-Origin: * junto com credenciais/cookies.
		AllowedOrigins:     getEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000"),
		PlatformFeePercent: getEnvAsInt("PLATFORM_FEE_PERCENT", 10),
		// Cookies com flag Secure exigem HTTPS. Em desenvolvimento local (HTTP)
		// deve permanecer false; em producao defina COOKIE_SECURE=true.
		CookieSecure: getEnv("COOKIE_SECURE", "false") == "true",

		// Pagamentos. "mock" e aceito apenas para desenvolvimento local/testes;
		// em producao use o identificador de um PSP real.
		PixProvider:          getEnv("PIX_PROVIDER", "mock"),
		PixWebhookSecret:     getEnv("PIX_WEBHOOK_SECRET", "fitconecta_dev_pix_webhook_secret"),
		PixExpirationMinutes: getEnvAsInt("PIX_EXPIRATION_MINUTES", 10),
		PixGraceMinutes:      getEnvAsInt("PIX_GRACE_MINUTES", 2),

		// Midia. Padroes apontam para o MinIO local (docker-compose).
		// Em producao (Cloudflare R2): STORAGE_ENDPOINT=<account>.r2.cloudflarestorage.com,
		// STORAGE_REGION=auto, STORAGE_USE_SSL=true, STORAGE_PATH_STYLE=false e
		// STORAGE_PUBLIC_BASE_URL=https://<dominio-publico>.
		StorageEndpoint:        getEnv("STORAGE_ENDPOINT", "localhost:9000"),
		StorageRegion:          getEnv("STORAGE_REGION", "us-east-1"),
		StorageAccessKeyID:     getEnv("STORAGE_ACCESS_KEY_ID", "fitconecta_minio"),
		StorageSecretAccessKey: getEnv("STORAGE_SECRET_ACCESS_KEY", "fitconecta_minio_secret"),
		StorageBucket:          getEnv("STORAGE_BUCKET", "fitconecta-media"),
		StoragePublicBaseURL:   getEnv("STORAGE_PUBLIC_BASE_URL", "http://localhost:9000/fitconecta-media"),
		StorageUseSSL:          getEnv("STORAGE_USE_SSL", "false") == "true",
		StoragePathStyle:       getEnv("STORAGE_PATH_STYLE", "true") == "true",
		StoragePresignMinutes:  getEnvAsInt("STORAGE_PRESIGN_MINUTES", 15),
		StorageProvision:       true,
	}

	applyNeonStorage(cfg)
	return cfg
}

// applyNeonStorage detecta as credenciais do Neon Object Storage (injetadas por
// `neon deploy`/`neon env pull` como AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY,
// AWS_ENDPOINT_URL_S3 e AWS_REGION) e reconfigura o cliente de midia.
//
// A selecao do provedor e feita por STORAGE_PROVIDER:
//   - "neon": usa o Neon mesmo com um STORAGE_ENDPOINT explicito;
//   - "minio"/"r2"/"s3": usa apenas as variaveis STORAGE_*;
//   - indefinido: usa o Neon se AWS_ENDPOINT_URL_S3 existir e nao houver
//     STORAGE_ENDPOINT explicito.
func applyNeonStorage(cfg *Config) {
	endpoint := os.Getenv("AWS_ENDPOINT_URL_S3")

	switch strings.ToLower(os.Getenv("STORAGE_PROVIDER")) {
	case "minio", "r2", "s3":
		return
	case "neon":
		if endpoint == "" {
			return
		}
	default:
		if endpoint == "" || os.Getenv("STORAGE_ENDPOINT") != "" {
			return
		}
	}

	host, useSSL := splitS3Endpoint(endpoint)
	cfg.StorageEndpoint = host
	cfg.StorageUseSSL = useSSL
	cfg.StoragePathStyle = true // Neon Object Storage usa somente path-style
	cfg.StorageRegion = getEnv("AWS_REGION", cfg.StorageRegion)
	cfg.StorageAccessKeyID = getEnv("AWS_ACCESS_KEY_ID", cfg.StorageAccessKeyID)
	cfg.StorageSecretAccessKey = getEnv("AWS_SECRET_ACCESS_KEY", cfg.StorageSecretAccessKey)
	cfg.StorageBucket = getEnv("STORAGE_BUCKET", cfg.StorageBucket)
	cfg.StoragePublicBaseURL = getEnv("STORAGE_PUBLIC_BASE_URL", strings.TrimRight(endpoint, "/")+"/"+cfg.StorageBucket)
	// O bucket e o modo public_read sao provisionados via neon.ts (`neon deploy`)
	// / Neon Console: a API S3 nao permite definir a politica (PutBucketPolicy
	// responde 501 Not Implemented).
	cfg.StorageProvision = false
}

// splitS3Endpoint separa o host do esquema de um endpoint S3. Sem esquema
// assume HTTPS (padrao dos provedores gerenciados).
func splitS3Endpoint(raw string) (host string, useSSL bool) {
	host = strings.TrimSpace(raw)
	switch {
	case strings.HasPrefix(host, "https://"):
		return strings.TrimSuffix(strings.TrimPrefix(host, "https://"), "/"), true
	case strings.HasPrefix(host, "http://"):
		return strings.TrimSuffix(strings.TrimPrefix(host, "http://"), "/"), false
	default:
		return strings.TrimSuffix(host, "/"), true
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}
