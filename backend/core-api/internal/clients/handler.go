package clients

import (
	"encoding/json"
	"errors"
	"net/http"

	"fitconecta/core-api/internal/platform/httputil"
	"fitconecta/core-api/internal/platform/middleware"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Routes monta /clients/me/profile. O alvo e sempre o perfil do token, nunca
// um id na URL (ADR 0012 §2 item 7).
func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	r.Group(func(authGroup chi.Router) {
		authGroup.Use(authMiddleware)
		authGroup.Use(middleware.RequireRole("client"))

		authGroup.Get("/me/profile", h.GetMyProfile)
		authGroup.Put("/me/profile", h.UpdateMyProfile)
	})

	return r
}

func (h *Handler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	authUser, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}

	profile, err := h.service.GetMyProfile(r.Context(), authUser.ID)
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			httputil.Error(w, http.StatusNotFound, "Perfil nao encontrado")
			return
		}
		httputil.Error(w, http.StatusInternalServerError, "Erro ao carregar seus dados")
		return
	}

	httputil.Success(w, http.StatusOK, profile)
}

func (h *Handler) UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	authUser, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}

	var req UpdatePersonalDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "Payload JSON invalido")
		return
	}

	profile, err := h.service.UpdateMyProfile(r.Context(), authUser.ID, &req)
	if err != nil {
		var validation *ValidationError
		switch {
		case errors.As(err, &validation):
			httputil.Error(w, http.StatusBadRequest, validation.Message)
		case errors.Is(err, ErrProfileNotFound):
			httputil.Error(w, http.StatusNotFound, "Perfil nao encontrado")
		default:
			httputil.Error(w, http.StatusInternalServerError, "Erro ao salvar seus dados")
		}
		return
	}

	httputil.Success(w, http.StatusOK, profile)
}
