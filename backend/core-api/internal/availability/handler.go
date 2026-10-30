package availability

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"fitconecta/core-api/internal/platform/httputil"
	"fitconecta/core-api/internal/platform/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ProfessionalProfileProvider interface {
	GetByUserID(r *http.Request) (uuid.UUID, error)
}

type Handler struct {
	service     Service
	getProfIDFn func(userID uuid.UUID) (uuid.UUID, error)
}

func NewHandler(service Service, getProfIDFn func(userID uuid.UUID) (uuid.UUID, error)) *Handler {
	return &Handler{
		service:     service,
		getProfIDFn: getProfIDFn,
	}
}

// RoutesSlots monta a rota publica de geracao de slots
func (h *Handler) RoutesSlots() chi.Router {
	r := chi.NewRouter()
	r.Get("/{id}/slots", h.GetSlots)
	return r
}

// RoutesAvailability monta as rotas autenticadas do profissional
func (h *Handler) RoutesAvailability(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	r.Group(func(authGroup chi.Router) {
		authGroup.Use(authMiddleware)
		authGroup.Use(middleware.RequireRole("professional"))

		authGroup.Get("/rules", h.ListRules)
		authGroup.Post("/rules", h.CreateRule)
		authGroup.Delete("/rules/{id}", h.DeleteRule)

		authGroup.Get("/exceptions", h.ListExceptions)
		authGroup.Post("/exceptions", h.CreateException)
		authGroup.Delete("/exceptions/{id}", h.DeleteException)
	})

	return r
}

func (h *Handler) GetSlots(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	profID, err := uuid.Parse(idStr)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "ID do profissional invalido")
		return
	}

	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	targetDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "Formato de data invalido (esperado: YYYY-MM-DD)")
		return
	}

	slots, err := h.service.GenerateSlots(r.Context(), profID, targetDate)
	if err != nil {
		if errors.Is(err, ErrProfessionalUnavailable) {
			httputil.Error(w, http.StatusNotFound, "Profissional indisponivel")
			return
		}
		httputil.Error(w, http.StatusInternalServerError, "Erro ao gerar slots de horarios")
		return
	}

	httputil.Success(w, http.StatusOK, slots)
}

func (h *Handler) ListRules(w http.ResponseWriter, r *http.Request) {
	profID, err := h.resolveProfID(r)
	if err != nil {
		httputil.Error(w, http.StatusForbidden, err.Error())
		return
	}

	rules, err := h.service.ListRules(r.Context(), profID)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "Erro ao listar regras de disponibilidade")
		return
	}
	if rules == nil {
		rules = []AvailabilityRule{}
	}
	httputil.Success(w, http.StatusOK, rules)
}

func (h *Handler) CreateRule(w http.ResponseWriter, r *http.Request) {
	profID, err := h.resolveProfID(r)
	if err != nil {
		httputil.Error(w, http.StatusForbidden, err.Error())
		return
	}

	var req CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "Payload JSON invalido")
		return
	}

	created, err := h.service.CreateRule(r.Context(), profID, &req)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	httputil.Success(w, http.StatusCreated, created, "Regra de disponibilidade criada com sucesso")
}

func (h *Handler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	profID, err := h.resolveProfID(r)
	if err != nil {
		httputil.Error(w, http.StatusForbidden, err.Error())
		return
	}

	ruleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "ID de regra invalido")
		return
	}

	if err := h.service.DeleteRule(r.Context(), profID, ruleID); err != nil {
		httputil.Error(w, http.StatusInternalServerError, "Erro ao excluir regra")
		return
	}

	httputil.Success(w, http.StatusOK, nil, "Regra removida com sucesso")
}

func (h *Handler) ListExceptions(w http.ResponseWriter, r *http.Request) {
	profID, err := h.resolveProfID(r)
	if err != nil {
		httputil.Error(w, http.StatusForbidden, err.Error())
		return
	}

	from := time.Now().AddDate(0, 0, -30)
	to := time.Now().AddDate(0, 3, 0)

	list, err := h.service.ListExceptions(r.Context(), profID, from, to)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "Erro ao listar excecoes")
		return
	}
	if list == nil {
		list = []AvailabilityException{}
	}
	httputil.Success(w, http.StatusOK, list)
}

func (h *Handler) CreateException(w http.ResponseWriter, r *http.Request) {
	profID, err := h.resolveProfID(r)
	if err != nil {
		httputil.Error(w, http.StatusForbidden, err.Error())
		return
	}

	var req CreateExceptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "Payload JSON invalido")
		return
	}

	created, err := h.service.CreateException(r.Context(), profID, &req)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	httputil.Success(w, http.StatusCreated, created, "Excecao/bloqueio adicionado com sucesso")
}

func (h *Handler) DeleteException(w http.ResponseWriter, r *http.Request) {
	profID, err := h.resolveProfID(r)
	if err != nil {
		httputil.Error(w, http.StatusForbidden, err.Error())
		return
	}

	excID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "ID de excecao invalido")
		return
	}

	if err := h.service.DeleteException(r.Context(), profID, excID); err != nil {
		httputil.Error(w, http.StatusInternalServerError, "Erro ao excluir excecao")
		return
	}

	httputil.Success(w, http.StatusOK, nil, "Excecao removida com sucesso")
}

func (h *Handler) resolveProfID(r *http.Request) (uuid.UUID, error) {
	authUser, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		return uuid.Nil, http.ErrNoCookie
	}
	return h.getProfIDFn(authUser.ID)
}
