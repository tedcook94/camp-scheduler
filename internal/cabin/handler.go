package cabin

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
	cabins := camps.Group("/:campId/cabins")
	cabins.GET("", h.List)
	cabins.GET("/:id", h.Get)
	cabins.POST("", h.Create)
	cabins.PUT("/:id", h.Update)
	cabins.DELETE("/:id", h.Delete)
}

type CreateCabinRequest struct {
	Name       string `json:"name" binding:"required"`
	AgeGroupID string `json:"age_group_id" binding:"required"`
}

type UpdateCabinRequest struct {
	Name       string `json:"name" binding:"required"`
	AgeGroupID string `json:"age_group_id" binding:"required"`
}

type CabinResponse struct {
	ID         string `json:"id"`
	CampID     string `json:"camp_id"`
	AgeGroupID string `json:"age_group_id"`
	Name       string `json:"name"`
}

func (h *Handler) List(c *gin.Context) {
	campID := c.Param("campId")

	cabins, err := h.svc.List(c.Request.Context(), campID)
	if err != nil {
		slog.
			With("camp_id", campID).
			With("error", err).
			Error("error listing cabins")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, cabins)
}

func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")

	cabin, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "cabin not found"})
			return
		}
		slog.
			With("id", id).
			With("error", err).
			Error("error getting cabin")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, cabin)
}

func (h *Handler) Create(c *gin.Context) {
	campID := c.Param("campId")

	var req CreateCabinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cabin, err := h.svc.Create(c.Request.Context(), campID, req)
	if err != nil {
		slog.
			With("camp_id", campID).
			With("error", err).
			Error("error creating cabin")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, cabin)
}

func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")

	var req UpdateCabinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cabin, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "cabin not found"})
			return
		}
		slog.
			With("id", id).
			With("error", err).
			Error("error updating cabin")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, cabin)
}

func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")

	err := h.svc.Delete(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "cabin not found"})
			return
		}
		slog.
			With("id", id).
			With("error", err).
			Error("error deleting cabin")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}
