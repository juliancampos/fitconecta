package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"fitconecta/core-api/internal/platform/httputil"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const (
	UserContextKey contextKey = "authUser"

	// AuthCookieName e o nome do cookie HttpOnly usado pela area web logada.
	AuthCookieName = "fitconecta_jwt"

	// AdminCookieName e o cookie host-only do app administrativo. Ele nunca e
	// enviado ao app publico nem aceito pelas rotas publicas.
	AdminCookieName = "fitconecta_admin_jwt"

	// PublicAudience isola os tokens emitidos para o app publico.
	PublicAudience = "fitconecta-web"

	// AdminAudience isola os tokens emitidos para o painel administrativo.
	AdminAudience = "fitconecta-admin"
)

// AuthUser representa o usuario extraido do token JWT no contexto da requisicao.
type AuthUser struct {
	ID     uuid.UUID `json:"id"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
	Status string    `json:"status"`
}

// UserStatusReader consulta o status atual da conta. A checagem no middleware
// garante que contas inativadas percam acesso mesmo com um JWT ainda valido.
type UserStatusReader interface {
	GetStatus(ctx context.Context, userID uuid.UUID) (string, error)
}

// JWTClaims sao os claims customizados do FitConecta+.
type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// AuthMiddleware valida o token do app publico sem checar o status da conta.
// Mantido para testes e usos que nao tenham acesso ao banco.
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return AuthMiddlewareWithStatus(jwtSecret, nil)
}

// AuthMiddlewareWithStatus valida o JWT publico e rejeita contas inativas.
// Contas bloqueadas continuam autenticadas, mas nao podem criar ou receber
// novos agendamentos.
func AuthMiddlewareWithStatus(jwtSecret string, statusReader UserStatusReader) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := extractToken(r)
			if tokenString == "" {
				httputil.Error(w, http.StatusUnauthorized, "Token de autorizacao nao informado")
				return
			}

			claims, err := parseToken(tokenString, jwtSecret, PublicAudience)
			if err != nil {
				httputil.Error(w, http.StatusUnauthorized, "Token invalido ou expirado")
				return
			}

			authUser, err := authUserFromClaims(claims)
			if err != nil {
				httputil.Error(w, http.StatusUnauthorized, "ID de usuario invalido no token")
				return
			}

			status, err := readStatus(r.Context(), statusReader, authUser.ID)
			if err != nil {
				httputil.Error(w, http.StatusUnauthorized, "Nao foi possivel validar a conta")
				return
			}
			if status == "inactive" {
				httputil.Error(w, http.StatusUnauthorized, "Conta inativa")
				return
			}
			authUser.Status = status

			ctx := context.WithValue(r.Context(), UserContextKey, authUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdminAuthMiddleware valida exclusivamente tokens do painel administrativo.
// O cookie e o segredo sao diferentes do app publico e a audience e obrigatoria.
func AdminAuthMiddleware(jwtSecret string, statusReader UserStatusReader) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := extractAdminToken(r)
			if tokenString == "" {
				httputil.Error(w, http.StatusUnauthorized, "Token de autorizacao nao informado")
				return
			}

			claims, err := parseToken(tokenString, jwtSecret, AdminAudience)
			if err != nil {
				httputil.Error(w, http.StatusUnauthorized, "Token invalido ou expirado")
				return
			}

			authUser, err := authUserFromClaims(claims)
			if err != nil {
				httputil.Error(w, http.StatusUnauthorized, "ID de usuario invalido no token")
				return
			}
			if authUser.Role != "admin" {
				httputil.Error(w, http.StatusForbidden, "Acesso restrito a administradores")
				return
			}

			status, err := readStatus(r.Context(), statusReader, authUser.ID)
			if err != nil {
				httputil.Error(w, http.StatusUnauthorized, "Nao foi possivel validar a conta")
				return
			}
			if status != "active" {
				httputil.Error(w, http.StatusUnauthorized, "Conta administrativa inativa")
				return
			}
			authUser.Status = status

			ctx := context.WithValue(r.Context(), UserContextKey, authUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func parseToken(tokenString, secret, audience string) (*JWTClaims, error) {
	claims := &JWTClaims{}
	options := []jwt.ParserOption{jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()})}
	if audience != "" {
		options = append(options, jwt.WithAudience(audience))
	}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("algoritmo de assinatura inesperado")
		}
		return []byte(secret), nil
	}, options...)
	if err != nil || !token.Valid {
		return nil, errors.New("token invalido")
	}
	return claims, nil
}

func authUserFromClaims(claims *JWTClaims) (AuthUser, error) {
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return AuthUser{}, err
	}
	return AuthUser{ID: userID, Email: claims.Email, Role: claims.Role}, nil
}

func readStatus(ctx context.Context, reader UserStatusReader, userID uuid.UUID) (string, error) {
	if reader == nil {
		return "", nil
	}
	return reader.GetStatus(ctx, userID)
}

// extractToken obtem o JWT a partir do header Authorization (prioritario) ou
// do cookie HttpOnly usado pela area web logada.
func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			return parts[1]
		}
		return ""
	}

	if cookie, err := r.Cookie(AuthCookieName); err == nil {
		return cookie.Value
	}

	return ""
}

// extractAdminToken aceita apenas o cookie do admin. O Authorization tambem e
// aceito para facilitar testes/automacao, mas nunca o cookie publico.
func extractAdminToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			return parts[1]
		}
		return ""
	}

	if cookie, err := r.Cookie(AdminCookieName); err == nil {
		return cookie.Value
	}

	return ""
}

// GetAuthUser recupera o usuario autenticado do contexto.
func GetAuthUser(ctx context.Context) (AuthUser, bool) {
	u, ok := ctx.Value(UserContextKey).(AuthUser)
	return u, ok
}

// RequireRole cria um middleware que exige um papel especifico (ex.: 'professional').
// Administradores continuam podendo acessar as rotas operacionais.
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, ok := GetAuthUser(r.Context())
			if !ok {
				httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
				return
			}

			if u.Role != role && u.Role != "admin" {
				httputil.Error(w, http.StatusForbidden, "Acesso restrito para o seu perfil")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin aceita somente o papel admin.
func RequireAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, ok := GetAuthUser(r.Context())
			if !ok {
				httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
				return
			}
			if u.Role != "admin" {
				httputil.Error(w, http.StatusForbidden, "Acesso restrito a administradores")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
