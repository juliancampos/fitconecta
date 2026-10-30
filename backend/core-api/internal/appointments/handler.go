package appointments

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"fitconecta/core-api/internal/payments"
	"fitconecta/core-api/internal/platform/httputil"
	"fitconecta/core-api/internal/platform/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	r.Group(func(authGroup chi.Router) {
		authGroup.Use(authMiddleware)

		authGroup.Post("/", h.Create)
		authGroup.Get("/", h.ListMy)
		authGroup.Get("/{id}", h.GetByID)
		authGroup.Post("/{id}/cancel", h.Cancel)
		authGroup.Post("/{id}/payment", h.CreatePixCharge)
	})

	return r
}

func (h *Handler) Webhook(provider payments.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		event, err := provider.ParseWebhook(r)
		if err != nil {
			if errors.Is(err, payments.ErrInvalidWebhook) {
				httputil.Error(w, http.StatusUnauthorized, "Webhook invalido")
				return
			}
			httputil.Error(w, http.StatusBadRequest, "Nao foi possivel ler o webhook")
			return
		}

		if err := h.service.HandleWebhook(r.Context(), event); err != nil {
			slog.Error("falha ao processar webhook de pagamento", "error", err, "provider", provider.Name())
			if errors.Is(err, payments.ErrProviderUnavailable) {
				httputil.Error(w, http.StatusServiceUnavailable, "Provedor de pagamento indisponivel")
				return
			}
			if errors.Is(err, payments.ErrInvalidWebhook) {
				httputil.Error(w, http.StatusBadRequest, "Evento de pagamento invalido")
				return
			}
			httputil.Error(w, http.StatusInternalServerError, "Erro ao processar webhook")
			return
		}

		httputil.Success(w, http.StatusOK, map[string]string{"status": "processed"})
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	authUser, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}

	var req CreateAppointmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "Payload JSON invalido")
		return
	}

	app, pay, err := h.service.Create(r.Context(), authUser, &req)
	if err != nil {
		if errors.Is(err, ErrSlotAlreadyBooked) {
			httputil.Error(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, ErrUnauthorized) {
			httputil.Error(w, http.StatusForbidden, err.Error())
			return
		}
		if errors.Is(err, ErrProfessionalUnavailable) {
			httputil.Error(w, http.StatusConflict, err.Error())
			return
		}
		httputil.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	type createResponse struct {
		Appointment *Appointment `json:"appointment"`
		Payment     *Payment     `json:"payment"`
		Message     string       `json:"message"`
	}

	resp := createResponse{
		Appointment: app,
		Payment:     pay,
		Message:     "Horario pre-reservado! Realize o pagamento para confirmar o agendamento.",
	}

	httputil.Success(w, http.StatusCreated, resp)
}

func (h *Handler) ListMy(w http.ResponseWriter, r *http.Request) {
	authUser, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}

	list, err := h.service.ListMyAppointments(r.Context(), authUser)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "Erro ao carregar agendamentos")
		return
	}
	if list == nil {
		list = []Appointment{}
	}

	httputil.Success(w, http.StatusOK, list)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	authUser, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "ID invalido")
		return
	}

	app, err := h.service.GetByID(r.Context(), authUser, id)
	if err != nil {
		if errors.Is(err, ErrAppointmentNotFound) {
			httputil.Error(w, http.StatusNotFound, "Agendamento nao encontrado")
			return
		}
		if errors.Is(err, ErrUnauthorized) {
			httputil.Error(w, http.StatusForbidden, "Acesso nao autorizado a este agendamento")
			return
		}
		httputil.Error(w, http.StatusInternalServerError, "Erro ao carregar detalhes do agendamento")
		return
	}

	httputil.Success(w, http.StatusOK, app)
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	authUser, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "ID invalido")
		return
	}

	var req CancelAppointmentRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	app, err := h.service.Cancel(r.Context(), authUser, id, req.Reason)
	if err != nil {
		if errors.Is(err, ErrInvalidStatus) {
			httputil.Error(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, ErrUnauthorized) {
			httputil.Error(w, http.StatusForbidden, "Acesso nao autorizado")
			return
		}
		httputil.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	httputil.Success(w, http.StatusOK, app, "Agendamento cancelado com sucesso")
}

func (h *Handler) CreatePixCharge(w http.ResponseWriter, r *http.Request) {
	authUser, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "ID invalido")
		return
	}

	charge, err := h.service.CreatePixCharge(r.Context(), authUser, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrAppointmentNotFound):
			httputil.Error(w, http.StatusNotFound, "Agendamento nao encontrado")
		case errors.Is(err, ErrUnauthorized):
			httputil.Error(w, http.StatusForbidden, "Acesso nao autorizado a este agendamento")
		case errors.Is(err, ErrInvalidStatus):
			httputil.Error(w, http.StatusConflict, "Agendamento nao esta aguardando pagamento")
		case errors.Is(err, ErrUnsupportedPaymentMethod):
			httputil.Error(w, http.StatusBadRequest, "Metodo de pagamento nao suportado")
		case errors.Is(err, payments.ErrProviderUnavailable):
			httputil.Error(w, http.StatusServiceUnavailable, "Provedor de pagamento indisponivel")
		default:
			httputil.Error(w, http.StatusInternalServerError, "Erro ao gerar cobranca Pix")
		}
		return
	}

	httputil.Success(w, http.StatusOK, charge, "Cobranca Pix gerada com sucesso")
}
