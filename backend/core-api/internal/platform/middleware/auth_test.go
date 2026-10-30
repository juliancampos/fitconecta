package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type fakeStatusReader map[uuid.UUID]string

func (f fakeStatusReader) GetStatus(_ context.Context, userID uuid.UUID) (string, error) {
	return f[userID], nil
}

func signedToken(t *testing.T, secret, audience, role string, userID uuid.UUID) string {
	t.Helper()
	claims := &JWTClaims{
		UserID: userID.String(),
		Email:  "user@example.com",
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{audience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("erro ao assinar token: %v", err)
	}
	return signed
}

func runAuth(t *testing.T, middleware func(http.Handler) http.Handler, token, cookieName string) (int, AuthUser, bool) {
	t.Helper()
	var got AuthUser
	var ok bool
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, ok = GetAuthUser(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if cookieName != "" {
		req.AddCookie(&http.Cookie{Name: cookieName, Value: token})
	} else if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec.Code, got, ok
}

func TestPublicMiddlewareRejectsInactiveUser(t *testing.T) {
	userID := uuid.New()
	mw := AuthMiddlewareWithStatus("public-secret", fakeStatusReader{userID: "inactive"})
	token := signedToken(t, "public-secret", "fitconecta-web", "client", userID)

	if code, _, _ := runAuth(t, mw, token, AuthCookieName); code != http.StatusUnauthorized {
		t.Fatalf("esperava 401 para conta inativa, obteve %d", code)
	}
}

func TestPublicMiddlewareKeepsBlockedUserAuthenticated(t *testing.T) {
	userID := uuid.New()
	mw := AuthMiddlewareWithStatus("public-secret", fakeStatusReader{userID: "blocked"})
	token := signedToken(t, "public-secret", "fitconecta-web", "client", userID)

	code, user, ok := runAuth(t, mw, token, AuthCookieName)
	if code != http.StatusOK || !ok {
		t.Fatalf("esperava 200 com usuario no contexto, obteve %d", code)
	}
	if user.Status != "blocked" {
		t.Fatalf("status esperado blocked, obteve %q", user.Status)
	}
}

func TestPublicMiddlewareRejectsAdminAudience(t *testing.T) {
	userID := uuid.New()
	mw := AuthMiddlewareWithStatus("public-secret", fakeStatusReader{userID: "active"})
	token := signedToken(t, "public-secret", AdminAudience, "admin", userID)

	if code, _, _ := runAuth(t, mw, token, AuthCookieName); code != http.StatusUnauthorized {
		t.Fatalf("esperava 401 para token com audience admin, obteve %d", code)
	}
}

func TestAdminMiddlewareRejectsPublicAudience(t *testing.T) {
	userID := uuid.New()
	mw := AdminAuthMiddleware("admin-secret", fakeStatusReader{userID: "active"})
	token := signedToken(t, "admin-secret", "fitconecta-web", "admin", userID)

	if code, _, _ := runAuth(t, mw, token, AdminCookieName); code != http.StatusUnauthorized {
		t.Fatalf("esperava 401 para token com audience publica, obteve %d", code)
	}
}

func TestAdminMiddlewareRejectsInactiveAdmin(t *testing.T) {
	userID := uuid.New()
	mw := AdminAuthMiddleware("admin-secret", fakeStatusReader{userID: "inactive"})
	token := signedToken(t, "admin-secret", AdminAudience, "admin", userID)

	if code, _, _ := runAuth(t, mw, token, AdminCookieName); code != http.StatusUnauthorized {
		t.Fatalf("esperava 401 para admin inativo, obteve %d", code)
	}
}

func TestAdminMiddlewareAcceptsAdminToken(t *testing.T) {
	userID := uuid.New()
	mw := AdminAuthMiddleware("admin-secret", fakeStatusReader{userID: "active"})
	token := signedToken(t, "admin-secret", AdminAudience, "admin", userID)

	code, user, ok := runAuth(t, mw, token, AdminCookieName)
	if code != http.StatusOK || !ok {
		t.Fatalf("esperava 200 com admin no contexto, obteve %d", code)
	}
	if user.Role != "admin" || user.ID != userID {
		t.Fatalf("usuario inesperado no contexto: %+v", user)
	}
}

func TestAdminMiddlewareDoesNotAcceptPublicCookie(t *testing.T) {
	userID := uuid.New()
	mw := AdminAuthMiddleware("admin-secret", fakeStatusReader{userID: "active"})
	token := signedToken(t, "admin-secret", AdminAudience, "admin", userID)

	if code, _, _ := runAuth(t, mw, token, AuthCookieName); code != http.StatusUnauthorized {
		t.Fatalf("esperava 401 para cookie publico, obteve %d", code)
	}
}
