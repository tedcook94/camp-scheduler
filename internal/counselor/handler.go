package counselor

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
	counselors := camps.Group("/:campId/counselors")
	counselors.GET("", h.List)
	counselors.GET("/:id", h.Get)
	counselors.POST("", h.Create)
	counselors.PUT("/:id", h.Update)
	counselors.DELETE("/:id", h.Delete)
}

type CreateCounselorRequest struct {
	Name            string `json:"name" binding:"required"`
	JuniorCounselor bool   `json:"junior_counselor"`
}

type UpdateCounselorRequest struct {
	Name            string `json:"name" binding:"required"`
	JuniorCounselor bool   `json:"junior_counselor"`
	Enabled         bool   `json:"enabled"`
}

type CounselorResponse struct {
	ID              string `json:"id"`
	CampID          string `json:"camp_id"`
	Name            string `json:"name"`
	JuniorCounselor bool   `json:"junior_counselor"`
	Enabled         bool   `json:"enabled"`
}

func (h *Handler) List(c *gin.Context) {
	campID := c.Param("campId")

	counselors, err := h.svc.List(c.Request.Context(), campID)
	if err != nil {
		slog.
			With("camp_id", campID).
			With("error", err).
			Error("error listing counselors")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, counselors)
}

func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")

	counselor, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "counselor not found"})
			return
		}
		slog.
			With("id", id).
			With("error", err).
			Error("error getting counselor")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, counselor)
}

func (h *Handler) Create(c *gin.Context) {
	campID := c.Param("campId")

	var req CreateCounselorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	counselor, err := h.svc.Create(c.Request.Context(), campID, req)
	if err != nil {
		slog.
			With("camp_id", campID).
			With("error", err).
			Error("error creating counselor")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, counselor)
}

func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")

	var req UpdateCounselorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	counselor, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "counselor not found"})
			return
		}
		slog.
			With("id", id).
			With("error", err).
			Error("error updating counselor")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, counselor)
}

func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")

	err := h.svc.Delete(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "counselor not found"})
			return
		}
		slog.
			With("id", id).
			With("error", err).
			Error("error deleting counselor")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}
