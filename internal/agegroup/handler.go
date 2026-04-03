package agegroup

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"camp-scheduler/internal/api"

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
	mux.HandleFunc("GET /api/v1/camps/{campId}/age-groups", h.List)
	mux.HandleFunc("GET /api/v1/camps/{campId}/age-groups/{id}", h.Get)
	mux.HandleFunc("POST /api/v1/camps/{campId}/age-groups", h.Create)
	mux.HandleFunc("PUT /api/v1/camps/{campId}/age-groups/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/camps/{campId}/age-groups/{id}", h.Delete)
}

type CreateAgeGroupRequest struct {
	Name string `json:"name" validate:"required"`
}

type UpdateAgeGroupRequest struct {
	Name string `json:"name" validate:"required"`
}

type AgeGroupResponse struct {
	ID     string `json:"id"`
	CampID string `json:"camp_id"`
	Name   string `json:"name"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	campID := r.PathValue("campId")

	groups, err := h.svc.List(r.Context(), campID)
	if err != nil {
		slog.Error("listing age groups", "error", err, "camp_id", campID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	api.WriteJSON(w, http.StatusOK, groups)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	group, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "age group not found", http.StatusNotFound)
			return
		}
		slog.Error("getting age group", "error", err, "id", id)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	api.WriteJSON(w, http.StatusOK, group)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	campID := r.PathValue("campId")

	var req CreateAgeGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	group, err := h.svc.Create(r.Context(), campID, req)
	if err != nil {
		slog.Error("creating age group", "error", err, "camp_id", campID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	api.WriteJSON(w, http.StatusCreated, group)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req UpdateAgeGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	group, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "age group not found", http.StatusNotFound)
			return
		}
		slog.Error("updating age group", "error", err, "id", id)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	api.WriteJSON(w, http.StatusOK, group)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	err := h.svc.Delete(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "age group not found", http.StatusNotFound)
			return
		}
		slog.Error("deleting age group", "error", err, "id", id)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
