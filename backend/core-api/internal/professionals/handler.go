package professionals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"fitconecta/core-api/internal/platform/httputil"
	"fitconecta/core-api/internal/platform/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// MediaStore abstrai o armazenamento de objetos S3-compativel (MinIO/R2).
type MediaStore interface {
	PresignPut(ctx context.Context, key string) (string, error)
	PublicURL(key string) string
	KeyFromURL(url string) string
	DeleteObject(ctx context.Context, key string) error
	ExpiresInSeconds() int64
}

type Handler struct {
	service Service
	media   MediaStore
}

func NewHandler(service Service, media MediaStore) *Handler {
	return &Handler{service: service, media: media}
}

// Tipos de imagem aceitos para upload de midia.
var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler, getSlotsHandler http.HandlerFunc) chi.Router {
	r := chi.NewRouter()

	// Rotas publicas (SEO, landing e catalogo)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	if getSlotsHandler != nil {
		r.Get("/{id}/slots", getSlotsHandler)
	}

	// Rotas restritas ao profissional autenticado
	r.Group(func(authGroup chi.Router) {
		authGroup.Use(authMiddleware)
		authGroup.Use(middleware.RequireRole("professional"))

		authGroup.Get("/me/profile", h.GetMyProfile)
		authGroup.Put("/me/profile", h.UpdateMyProfile)
		authGroup.Put("/me/specialties", h.SetMySpecialties)
		authGroup.Put("/me/objectives", h.SetMyObjectives)
		authGroup.Put("/me/audiences", h.SetMyAudiences)
		authGroup.Put("/me/locations", h.SetMyLocations)
		authGroup.Get("/me/neighborhoods", h.ListMyNeighborhoods)
		authGroup.Put("/me/neighborhoods", h.SetMyNeighborhoods)

		authGroup.Post("/me/media/presign", h.PresignMedia)
		authGroup.Put("/me/avatar", h.UpdateMyAvatar)
		authGroup.Get("/me/photos", h.ListMyPhotos)
		authGroup.Post("/me/photos", h.AddMyPhoto)
		authGroup.Delete("/me/photos/{id}", h.DeleteMyPhoto)
	})

	return r
}

func (h *Handler) SpecialtiesRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.ListSpecialties)
	return r
}

func (h *Handler) NeighborhoodsRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.ListNeighborhoods)
	return r
}

func (h *Handler) CatalogsRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.ListCatalogs)
	return r
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	specialtyID, _ := strconv.Atoi(q.Get("specialty_id"))
	neighborhoodID, _ := strconv.Atoi(q.Get("neighborhood_id"))

	f := FilterParams{
		City:           q.Get("city"),
		State:          q.Get("state"),
		SpecialtyID:    specialtyID,
		NeighborhoodID: neighborhoodID,
		Limit:          limit,
		Offset:         offset,
	}

	list, err := h.service.List(r.Context(), f)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "Erro ao buscar profissionais")
		return
	}

	if list == nil {
		list = []ProfessionalProfile{}
	}

	httputil.Success(w, http.StatusOK, list)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "ID do profissional invalido")
		return
	}

	prof, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrProfessionalNotFound) {
			httputil.Error(w, http.StatusNotFound, "Profissional nao encontrado")
			return
		}
		httputil.Error(w, http.StatusInternalServerError, "Erro ao carregar perfil do profissional")
		return
	}

	httputil.Success(w, http.StatusOK, prof)
}

func (h *Handler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	authUser, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}

	prof, err := h.service.GetMyProfile(r.Context(), authUser.ID)
	if err != nil {
		httputil.Error(w, http.StatusNotFound, "Perfil de profissional nao encontrado")
		return
	}

	httputil.Success(w, http.StatusOK, prof)
}

func (h *Handler) UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	authUser, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "Payload JSON invalido")
		return
	}

	updated, err := h.service.UpdateMyProfile(r.Context(), authUser.ID, &req)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	httputil.Success(w, http.StatusOK, updated, "Perfil atualizado com sucesso")
}

func (h *Handler) SetMySpecialties(w http.ResponseWriter, r *http.Request) {
	authUser, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}

	var req struct {
		SpecialtyIDs []int `json:"specialty_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "Payload JSON invalido")
		return
	}

	if err := h.service.SetMySpecialties(r.Context(), authUser.ID, req.SpecialtyIDs); err != nil {
		httputil.Error(w, http.StatusInternalServerError, "Erro ao associar especialidades")
		return
	}

	httputil.Success(w, http.StatusOK, nil, "Especialidades salvas com sucesso")
}

func (h *Handler) ListSpecialties(w http.ResponseWriter, r *http.Request) {
	specs, err := h.service.ListSpecialties(r.Context())
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "Erro ao listar modalidades")
		return
	}
	if specs == nil {
		specs = []Specialty{}
	}
	httputil.Success(w, http.StatusOK, specs)
}

func (h *Handler) ListCatalogs(w http.ResponseWriter, r *http.Request) {
	catalogs, err := h.service.ListCatalogs(r.Context())
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "Erro ao carregar catalogos")
		return
	}
	if catalogs.Services == nil {
		catalogs.Services = []Specialty{}
	}
	if catalogs.Objectives == nil {
		catalogs.Objectives = []Objective{}
	}
	if catalogs.Audiences == nil {
		catalogs.Audiences = []Audience{}
	}
	if catalogs.Locations == nil {
		catalogs.Locations = []ServiceLocation{}
	}
	httputil.Success(w, http.StatusOK, catalogs)
}

// writeCatalogValidation responde 400 para erros de catalogo/publico e devolve true.
func writeCatalogValidation(w http.ResponseWriter, err error) bool {
	if errors.Is(err, ErrInvalidCatalogItem) || errors.Is(err, ErrAudienceOtherRequired) {
		httputil.Error(w, http.StatusBadRequest, err.Error())
		return true
	}
	return false
}

func (h *Handler) SetMyObjectives(w http.ResponseWriter, r *http.Request) {
	authUser, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}

	var req struct {
		ObjectiveIDs []int `json:"objective_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "Payload JSON invalido")
		return
	}

	if err := h.service.SetMyObjectives(r.Context(), authUser.ID, req.ObjectiveIDs); err != nil {
		if writeCatalogValidation(w, err) {
			return
		}
		httputil.Error(w, http.StatusInternalServerError, "Erro ao associar objetivos")
		return
	}

	httputil.Success(w, http.StatusOK, nil, "Objetivos salvos com sucesso")
}

func (h *Handler) SetMyAudiences(w http.ResponseWriter, r *http.Request) {
	authUser, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}

	var req SetAudiencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "Payload JSON invalido")
		return
	}

	if err := h.service.SetMyAudiences(r.Context(), authUser.ID, &req); err != nil {
		if writeCatalogValidation(w, err) {
			return
		}
		httputil.Error(w, http.StatusInternalServerError, "Erro ao associar publicos")
		return
	}

	httputil.Success(w, http.StatusOK, nil, "Publicos salvos com sucesso")
}

func (h *Handler) SetMyLocations(w http.ResponseWriter, r *http.Request) {
	authUser, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}

	var req struct {
		LocationIDs []int `json:"location_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "Payload JSON invalido")
		return
	}

	if err := h.service.SetMyLocations(r.Context(), authUser.ID, req.LocationIDs); err != nil {
		if writeCatalogValidation(w, err) {
			return
		}
		httputil.Error(w, http.StatusInternalServerError, "Erro ao associar locais")
		return
	}

	httputil.Success(w, http.StatusOK, nil, "Locais salvos com sucesso")
}

func (h *Handler) ListNeighborhoods(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	list, err := h.service.ListNeighborhoods(r.Context(), q.Get("city"), q.Get("state"))
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "Erro ao listar bairros")
		return
	}
	if list == nil {
		list = []Neighborhood{}
	}
	httputil.Success(w, http.StatusOK, list)
}

func (h *Handler) ListMyNeighborhoods(w http.ResponseWriter, r *http.Request) {
	authUser, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}

	list, err := h.service.ListMyNeighborhoods(r.Context(), authUser.ID)
	if err != nil {
		httputil.Error(w, http.StatusNotFound, "Perfil de profissional nao encontrado")
		return
	}
	if list == nil {
		list = []Neighborhood{}
	}
	httputil.Success(w, http.StatusOK, list)
}

func (h *Handler) SetMyNeighborhoods(w http.ResponseWriter, r *http.Request) {
	authUser, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}

	var req struct {
		NeighborhoodIDs []int `json:"neighborhood_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "Payload JSON invalido")
		return
	}

	if err := h.service.SetMyNeighborhoods(r.Context(), authUser.ID, req.NeighborhoodIDs); err != nil {
		httputil.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	httputil.Success(w, http.StatusOK, nil, "Bairros salvos com sucesso")
}

// PresignMedia gera uma URL pre-assinada para o profissional enviar (PUT) uma
// imagem diretamente ao storage, sem passar pela API.
func (h *Handler) PresignMedia(w http.ResponseWriter, r *http.Request) {
	authUser, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}
	if h.media == nil {
		httputil.Error(w, http.StatusServiceUnavailable, "Armazenamento de midia nao configurado")
		return
	}

	var req struct {
		Kind        string `json:"kind"`
		ContentType string `json:"content_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "Payload JSON invalido")
		return
	}

	ext, ok := allowedImageTypes[req.ContentType]
	if !ok {
		httputil.Error(w, http.StatusBadRequest, "Tipo de imagem nao suportado (use JPEG, PNG, WEBP ou GIF)")
		return
	}

	var folder string
	switch req.Kind {
	case "avatar":
		folder = "avatar"
	case "photo":
		folder = "photos"
	default:
		httputil.Error(w, http.StatusBadRequest, "Campo kind deve ser 'avatar' ou 'photo'")
		return
	}

	prof, err := h.service.GetMyProfile(r.Context(), authUser.ID)
	if err != nil {
		httputil.Error(w, http.StatusNotFound, "Perfil do profissional nao encontrado")
		return
	}

	key := fmt.Sprintf("professionals/%s/%s/%s%s", prof.ID, folder, uuid.NewString(), ext)
	uploadURL, err := h.media.PresignPut(r.Context(), key)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "Erro ao gerar URL de upload")
		return
	}

	httputil.Success(w, http.StatusOK, map[string]interface{}{
		"key":        key,
		"upload_url": uploadURL,
		"public_url": h.media.PublicURL(key),
		"expires_in": h.media.ExpiresInSeconds(),
	})
}

func (h *Handler) UpdateMyAvatar(w http.ResponseWriter, r *http.Request) {
	authUser, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}

	var req UpdateAvatarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "Payload JSON invalido")
		return
	}

	updated, err := h.service.UpdateMyAvatar(r.Context(), authUser.ID, req.AvatarURL)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	httputil.Success(w, http.StatusOK, updated, "Avatar atualizado com sucesso")
}

func (h *Handler) ListMyPhotos(w http.ResponseWriter, r *http.Request) {
	authUser, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}

	photos, err := h.service.ListMyPhotos(r.Context(), authUser.ID)
	if err != nil {
		httputil.Error(w, http.StatusNotFound, "Perfil do profissional nao encontrado")
		return
	}
	if photos == nil {
		photos = []ProfessionalPhoto{}
	}

	httputil.Success(w, http.StatusOK, photos)
}

func (h *Handler) AddMyPhoto(w http.ResponseWriter, r *http.Request) {
	authUser, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}

	var req AddPhotoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "Payload JSON invalido")
		return
	}

	photo, err := h.service.AddMyPhoto(r.Context(), authUser.ID, req.URL)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	httputil.Success(w, http.StatusCreated, photo, "Foto adicionada com sucesso")
}

func (h *Handler) DeleteMyPhoto(w http.ResponseWriter, r *http.Request) {
	authUser, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		httputil.Error(w, http.StatusUnauthorized, "Nao autenticado")
		return
	}

	photoID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "ID da foto invalido")
		return
	}

	if err := h.service.DeleteMyPhoto(r.Context(), authUser.ID, photoID); err != nil {
		httputil.Error(w, http.StatusNotFound, "Foto nao encontrada")
		return
	}

	httputil.Success(w, http.StatusOK, nil, "Foto removida com sucesso")
}
