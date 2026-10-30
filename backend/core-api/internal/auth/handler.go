package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"fitconecta/core-api/internal/platform/httputil"
	"fitconecta/core-api/internal/platform/middleware"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service      Service
	secureCookie bool
}

func NewHandler(service Service, secureCookie bool) *Handler {
	return &Handler{service: service, secureCookie: secureCookie}
}

// setAuthCookie grava o JWT em um cookie HttpOnly usado pela area web logada.
// O app mobile continua usando o token devolvido no corpo da resposta.
func (h *Handler) setAuthCookie(w http.ResponseWriter, token string, expiresIn int64) {
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.AuthCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(expiresIn),
	})
}

// clearAuthCookie expira o cookie de sessao da area web.
func (h *Handler) clearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.AuthCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

// Routes monta o sub-roteador de autenticacao
func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/logout", h.Logout)

	// Rotas autenticadas
	r.Group(func(authGroup chi.Router) {
		authGroup.Use(authMiddleware)
		authGroup.Get("/me", h.Me)
	})

	return r
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "Payload JSON invalido")
		return
	}

	resp, err := h.service.Register(r.Context(), &req)
	if err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			httputil.Error(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, ErrInvalidInput) {
			httputil.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		httputil.Error(w, http.StatusInternalServerError, "Erro interno ao cadastrar usuario")
		return
	}

	h.setAuthCookie(w, resp.Token, resp.ExpiresIn)
	httputil.Success(w, http.StatusCreated, resp, "Cadastro realizado com sucesso")
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "Payload JSON invalido")
		return
	}

	resp, err := h.service.Login(r.Context(), &req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			httputil.Error(w, http.StatusUnauthorized, "E-mail ou senha incorretos")
			return
		}
		httputil.Error(w, http.StatusInternalServerError, "Erro interno ao processar autenticacao")
		return
	}

	h.setAuthCookie(w, resp.Token, resp.ExpiresIn)
	httputil.Success(w, http.StatusOK, resp, "Login realizado com sucesso")
}

// Logout expira o cookie de sessao da area web.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	h.clearAuthCookie(w)
	httputil.Success(w, http.StatusOK, nil, "Sessao encerrada com sucesso")
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	authUser, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}

	resp, err := h.service.GetMe(r.Context(), authUser.ID, Role(authUser.Role))
	if err != nil {
		httputil.Error(w, http.StatusNotFound, "Perfil nao encontrado")
		return
	}

	httputil.Success(w, http.StatusOK, resp)
}
