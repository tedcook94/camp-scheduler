package camp

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
)

type Handler struct {
	svc      *Service
	validate *validator.Validate
}

func NewHandler(svc *Service) *Handler {
	return &Handler{
		svc:      svc,
		validate: validator.New(),
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/camps", h.List)
	mux.HandleFunc("GET /api/v1/camps/{id}", h.Get)
	mux.HandleFunc("POST /api/v1/camps", h.Create)
	mux.HandleFunc("PUT /api/v1/camps/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/camps/{id}", h.Delete)
}

type CreateCampRequest struct {
	Name     string  `json:"name" validate:"required"`
	Location *string `json:"location"`
}

type UpdateCampRequest struct {
	Name     string  `json:"name" validate:"required"`
	Location *string `json:"location"`
	Enabled  bool    `json:"enabled"`
}

type CampResponse struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Location *string `json:"location"`
	Enabled  bool    `json:"enabled"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	camps, err := h.svc.List(r.Context())
	if err != nil {
		slog.Error("listing camps", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, camps)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	camp, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "camp not found", http.StatusNotFound)
			return
		}
		slog.Error("getting camp", "error", err, "id", id)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, camp)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateCampRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	camp, err := h.svc.Create(r.Context(), req)
	if err != nil {
		slog.Error("creating camp", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, camp)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req UpdateCampRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	camp, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "camp not found", http.StatusNotFound)
			return
		}
		slog.Error("updating camp", "error", err, "id", id)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, camp)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	err := h.svc.Delete(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "camp not found", http.StatusNotFound)
			return
		}
		slog.Error("deleting camp", "error", err, "id", id)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("writing json response", "error", err)
	}
}
