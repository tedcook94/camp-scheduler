package season

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(camps *gin.RouterGroup) {
	seasons := camps.Group("/:campId/seasons")
	seasons.GET("", h.List)
	seasons.GET("/:id", h.Get)
	seasons.POST("", h.Create)
	seasons.PUT("/:id", h.Update)
	seasons.DELETE("/:id", h.Delete)
}

type CreateSeasonRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateSeasonRequest struct {
	Name string `json:"name" binding:"required"`
}

type SeasonResponse struct {
	ID     string `json:"id"`
	CampID string `json:"camp_id"`
	Name   string `json:"name"`
}

func (h *Handler) List(c *gin.Context) {
	campID := c.Param("campId")

	seasons, err := h.svc.List(c.Request.Context(), campID)
	if err != nil {
		slog.
			With("camp_id", campID).
			With("error", err).
			Error("error listing seasons")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, seasons)
}

func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")

	season, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "season not found"})
			return
		}
		slog.
			With("id", id).
			With("error", err).
			Error("error getting season")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, season)
}

func (h *Handler) Create(c *gin.Context) {
	campID := c.Param("campId")

	var req CreateSeasonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	season, err := h.svc.Create(c.Request.Context(), campID, req)
	if err != nil {
		slog.
			With("camp_id", campID).
			With("error", err).
			Error("error creating season")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, season)
}

func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")

	var req UpdateSeasonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	season, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "season not found"})
			return
		}
		slog.
			With("id", id).
			With("error", err).
			Error("error updating season")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, season)
}

func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")

	err := h.svc.Delete(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "season not found"})
			return
		}
		slog.
			With("id", id).
			With("error", err).
			Error("error deleting season")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}
