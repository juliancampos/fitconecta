package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"fitconecta/core-api/internal/cref"
	"fitconecta/core-api/internal/platform/httputil"
	"fitconecta/core-api/internal/platform/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	repo          Repository
	auth          AuthService
	cookieSecure  bool
	crefValidator cref.Validator
}

func NewHandler(repo Repository, authService AuthService, cookieSecure bool, crefValidator cref.Validator) *Handler {
	return &Handler{repo: repo, auth: authService, cookieSecure: cookieSecure, crefValidator: crefValidator}
}

func (h *Handler) Routes(adminAuth func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Post("/auth/login", h.Login)
	r.Post("/auth/logout", h.Logout)

	r.Group(func(g chi.Router) {
		g.Use(adminAuth)
		g.Get("/auth/me", h.Me)
		g.Get("/appointments", h.ListAppointments)
		g.Get("/appointments/{id}", h.GetAppointment)
		g.Post("/appointments/{id}/cancel", h.CancelAppointment)
		g.Get("/payments", h.ListPayments)
		g.Get("/payments/summary", h.SummaryPayments)
		g.Get("/users", h.ListUsers)
		g.Get("/users/{id}", h.GetUser)
		g.Patch("/users/{id}/status", h.UpdateUserStatus)
		g.Post("/users/{id}/cref/revalidate", h.RevalidateUserCREF)
	})

	return r
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "Payload JSON invalido")
		return
	}

	resp, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidAdminLogin) {
			httputil.Error(w, http.StatusUnauthorized, "E-mail ou senha incorretos")
			return
		}
		httputil.Error(w, http.StatusInternalServerError, "Erro ao autenticar")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.AdminCookieName,
		Value:    resp.Token,
		Path:     "/",
		MaxAge:   int(resp.ExpiresIn),
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
	httputil.Success(w, http.StatusOK, resp, "Login realizado com sucesso")
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.AdminCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
	httputil.Success(w, http.StatusOK, map[string]bool{"logged_out": true})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	u, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}
	httputil.Success(w, http.StatusOK, u)
}

func (h *Handler) ListAppointments(w http.ResponseWriter, r *http.Request) {
	f, err := parseAppointmentFilters(r.URL.Query())
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	page, err := h.repo.ListAppointments(r.Context(), *f)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	httputil.Success(w, http.StatusOK, page)
}

func (h *Handler) GetAppointment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "ID invalido")
		return
	}
	item, err := h.repo.GetAppointment(r.Context(), id)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	httputil.Success(w, http.StatusOK, item)
}

type cancelRequest struct {
	Reason string `json:"reason"`
}

func (h *Handler) CancelAppointment(w http.ResponseWriter, r *http.Request) {
	u, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "ID invalido")
		return
	}
	var req cancelRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if strings.TrimSpace(req.Reason) == "" {
		httputil.Error(w, http.StatusUnprocessableEntity, "Informe o motivo do cancelamento")
		return
	}

	item, err := h.repo.CancelAppointment(r.Context(), u.ID, id, strings.TrimSpace(req.Reason), requestIP(r), r.UserAgent())
	if err != nil {
		writeRepoError(w, err)
		return
	}
	httputil.Success(w, http.StatusOK, item, "Agendamento cancelado pelo administrador")
}

func (h *Handler) ListPayments(w http.ResponseWriter, r *http.Request) {
	f, err := parsePaymentFilters(r.URL.Query())
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	page, err := h.repo.ListPayments(r.Context(), *f)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	httputil.Success(w, http.StatusOK, page)
}

func (h *Handler) SummaryPayments(w http.ResponseWriter, r *http.Request) {
	f, err := parsePaymentFilters(r.URL.Query())
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	summary, err := h.repo.SummaryPayments(r.Context(), *f)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	httputil.Success(w, http.StatusOK, summary)
}

// RevalidateUserCREF consulta o portal do CREF6 de novo e, se o registro
// estiver ativo, grava a data da verificacao. Responde 200 com o resultado
// para o painel exibir a mensagem sem tratar como erro de requisicao.
func (h *Handler) RevalidateUserCREF(w http.ResponseWriter, r *http.Request) {
	admin, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "ID invalido")
		return
	}
	user, err := h.repo.GetUser(r.Context(), id)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	if user.Role != "professional" || user.ProfessionalID == nil {
		httputil.Error(w, http.StatusUnprocessableEntity, "Usuario nao e um profissional")
		return
	}
	number := ""
	if user.CREFNumber != nil {
		number = strings.TrimSpace(*user.CREFNumber)
	}
	if number == "" {
		httputil.Success(w, http.StatusOK, CREFCheck{Reason: "no_cref", Message: "Profissional sem CREF cadastrado."})
		return
	}

	check := CREFCheck{Verified: true, Reason: "ok", Message: "CREF verificado no portal do CREF6."}
	if _, err := h.crefValidator.Validate(r.Context(), number); err != nil {
		check = crefCheckError(err)
	}

	write, verifiedAt := decideCREFUpdate(check, time.Now())
	if !write {
		httputil.Success(w, http.StatusOK, check)
		return
	}

	applied, err := h.repo.SetProfessionalCREFVerified(
		r.Context(), admin.ID, user.ID, number, verifiedAt, check.Reason, requestIP(r), r.UserAgent(),
	)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	if !applied {
		httputil.Success(w, http.StatusOK, CREFCheck{
			Reason:  "cref_changed",
			Message: "O CREF foi alterado durante a consulta. Revalide o registro.",
		})
		return
	}
	if !check.Verified {
		check.Message += " O selo de verificacao foi removido."
	}
	httputil.Success(w, http.StatusOK, check)
}

// decideCREFUpdate diz se o resultado da consulta deve ser persistido e com
// qual data: positivo grava agora; negativo remove o selo antigo (o portal eh a
// fonte da verdade). Portal indisponivel nao altera nada.
func decideCREFUpdate(check CREFCheck, now time.Time) (bool, *time.Time) {
	switch check.Reason {
	case "ok":
		return true, &now
	case "not_found", "inactive":
		return true, nil
	default:
		return false, nil
	}
}

func crefCheckError(err error) CREFCheck {
	switch {
	case errors.Is(err, cref.ErrNotFound):
		return CREFCheck{Reason: "not_found", Message: "CREF nao encontrado no portal do CREF6 (MG)."}
	case errors.Is(err, cref.ErrInactive):
		return CREFCheck{Reason: "inactive", Message: "CREF encontrado, mas nao esta ativo no portal do CREF6."}
	default:
		return CREFCheck{Reason: "unavailable", Message: "Nao foi possivel consultar o portal do CREF6 agora. Tente novamente em instantes."}
	}
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	f, err := parseUserFilters(r.URL.Query())
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	page, err := h.repo.ListUsers(r.Context(), *f)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	httputil.Success(w, http.StatusOK, page)
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "ID invalido")
		return
	}
	user, err := h.repo.GetUser(r.Context(), id)
	if err != nil {
		writeRepoError(w, err)
		return
	}
	httputil.Success(w, http.StatusOK, user)
}

type updateUserStatusRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

func (h *Handler) UpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	u, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "ID invalido")
		return
	}
	var req updateUserStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "Payload JSON invalido")
		return
	}
	req.Status = strings.TrimSpace(req.Status)
	if !validUserStatus(req.Status) {
		httputil.Error(w, http.StatusUnprocessableEntity, "Status deve ser active, blocked ou inactive")
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)

	resp, err := h.repo.UpdateUserStatus(r.Context(), u.ID, id, req.Status, req.Reason, requestIP(r), r.UserAgent())
	if err != nil {
		writeRepoError(w, err)
		return
	}
	httputil.Success(w, http.StatusOK, resp, "Status do usuario atualizado")
}

func parseAppointmentFilters(q url.Values) (*AppointmentFilters, error) {
	page, pageSize := parsePagination(q)
	from, to, err := parseDateRange(q)
	if err != nil {
		return nil, err
	}
	status := strings.TrimSpace(q.Get("status"))
	if status != "" && !validAppointmentStatus(status) {
		return nil, fmt.Errorf("%w: status de agendamento desconhecido", ErrInvalidFilter)
	}
	sort := strings.TrimSpace(q.Get("sort"))
	if sort != "" && sort != "starts_at_asc" && sort != "starts_at_desc" && sort != "created_at_desc" {
		return nil, fmt.Errorf("%w: ordenacao desconhecida", ErrInvalidFilter)
	}

	f := &AppointmentFilters{
		ClientQuery:       strings.TrimSpace(q.Get("client_q")),
		ProfessionalQuery: strings.TrimSpace(q.Get("professional_q")),
		Query:             strings.TrimSpace(q.Get("q")),
		Status:            status,
		Sort:              sort,
		DateFrom:          from,
		DateTo:            to,
		Page:              page,
		PageSize:          pageSize,
	}
	if f.ClientID, err = parseUUIDQuery(q, "client_id"); err != nil {
		return nil, err
	}
	if f.ProfessionalID, err = parseUUIDQuery(q, "professional_id"); err != nil {
		return nil, err
	}
	return f, nil
}

func parsePaymentFilters(q url.Values) (*PaymentFilters, error) {
	page, pageSize := parsePagination(q)
	from, to, err := parseDateRange(q)
	if err != nil {
		return nil, err
	}
	status := strings.TrimSpace(q.Get("status"))
	if status != "" && !validPaymentStatus(status) {
		return nil, fmt.Errorf("%w: status de pagamento desconhecido", ErrInvalidFilter)
	}
	method := strings.TrimSpace(q.Get("method"))
	if method != "" && method != "pix" && method != "credit_card" {
		return nil, fmt.Errorf("%w: metodo de pagamento desconhecido", ErrInvalidFilter)
	}
	dateField := strings.TrimSpace(q.Get("date_field"))
	if dateField != "" && dateField != "created_at" && dateField != "paid_at" {
		return nil, fmt.Errorf("%w: campo de data desconhecido", ErrInvalidFilter)
	}

	f := &PaymentFilters{
		ClientQuery:       strings.TrimSpace(q.Get("client_q")),
		ProfessionalQuery: strings.TrimSpace(q.Get("professional_q")),
		Query:             strings.TrimSpace(q.Get("q")),
		Status:            status,
		Method:            method,
		DateField:         dateField,
		DateFrom:          from,
		DateTo:            to,
		Page:              page,
		PageSize:          pageSize,
	}
	if f.ClientID, err = parseUUIDQuery(q, "client_id"); err != nil {
		return nil, err
	}
	if f.ProfessionalID, err = parseUUIDQuery(q, "professional_id"); err != nil {
		return nil, err
	}
	return f, nil
}

func parseUserFilters(q url.Values) (*UserFilters, error) {
	page, pageSize := parsePagination(q)
	role := strings.TrimSpace(q.Get("role"))
	if role != "" && role != "client" && role != "professional" {
		return nil, fmt.Errorf("%w: papel desconhecido", ErrInvalidFilter)
	}
	status := strings.TrimSpace(q.Get("status"))
	if status != "" && !validUserStatus(status) {
		return nil, fmt.Errorf("%w: status de usuario desconhecido", ErrInvalidFilter)
	}
	return &UserFilters{
		Query:    strings.TrimSpace(q.Get("q")),
		Role:     role,
		Status:   status,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func parsePagination(q url.Values) (int, int) {
	page := 1
	pageSize := 20
	if v := q.Get("page"); v != "" {
		if n, err := parsePositiveInt(v); err == nil {
			page = n
		}
	}
	if v := q.Get("page_size"); v != "" {
		if n, err := parsePositiveInt(v); err == nil {
			pageSize = n
		}
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func parsePositiveInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("numero invalido")
	}
	return n, nil
}

func parseDateRange(q url.Values) (*time.Time, *time.Time, error) {
	from, err := parseDate(q.Get("date_from"), false)
	if err != nil {
		return nil, nil, err
	}
	to, err := parseDate(q.Get("date_to"), true)
	if err != nil {
		return nil, nil, err
	}
	if from != nil && to != nil && !from.Before(*to) {
		return nil, nil, fmt.Errorf("%w: date_from deve ser anterior a date_to", ErrInvalidFilter)
	}
	return from, to, nil
}

func parseDate(value string, endOfDay bool) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		loc = time.UTC
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return &t, nil
	}
	t, err := time.ParseInLocation("2006-01-02", value, loc)
	if err != nil {
		return nil, fmt.Errorf("%w: data deve estar no formato YYYY-MM-DD", ErrInvalidFilter)
	}
	if endOfDay {
		t = t.AddDate(0, 0, 1)
	}
	return &t, nil
}

func parseUUIDQuery(q url.Values, key string) (*uuid.UUID, error) {
	value := strings.TrimSpace(q.Get(key))
	if value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %s invalido", ErrInvalidFilter, key)
	}
	return &id, nil
}

func validUserStatus(status string) bool {
	return status == "active" || status == "blocked" || status == "inactive"
}

func validAppointmentStatus(status string) bool {
	switch status {
	case "pending_payment", "confirmed", "cancelled_by_client", "cancelled_by_professional",
		"cancelled_by_admin", "completed", "no_show", "expired":
		return true
	default:
		return false
	}
}

func validPaymentStatus(status string) bool {
	switch status {
	case "pending", "paid", "failed", "refund_pending", "refunded", "expired":
		return true
	default:
		return false
	}
}

func requestIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func writeRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httputil.Error(w, http.StatusNotFound, "Registro nao encontrado")
	case errors.Is(err, ErrInvalidStatus):
		httputil.Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrInvalidTarget):
		httputil.Error(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, ErrReasonRequired):
		httputil.Error(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, ErrInvalidFilter):
		httputil.Error(w, http.StatusBadRequest, err.Error())
	default:
		httputil.Error(w, http.StatusInternalServerError, "Erro interno ao processar a solicitacao")
	}
}
