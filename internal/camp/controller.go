package camp

import (
	"errors"
	"log/slog"
	"net/http"

	"camp-scheduler/internal/api"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type Controller struct {
	svc *Service
}

func NewController(svc *Service) *Controller {
	return &Controller{svc: svc}
}

func (ctrl *Controller) RegisterRoutes(camps *gin.RouterGroup) {
	camps.GET("", ctrl.List)
	camps.GET("/:campId", ctrl.Get)
	camps.POST("", ctrl.Create)
	camps.PUT("/:campId", ctrl.Update)
	camps.DELETE("/:campId", ctrl.Delete)
}

type CreateCampRequest struct {
	Name     string  `json:"name" binding:"required"`
	Location *string `json:"location"`
}

type UpdateCampRequest struct {
	Name     string  `json:"name" binding:"required"`
	Location *string `json:"location"`
	Enabled  bool    `json:"enabled"`
}

type CampResponse struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Location *string `json:"location"`
	Enabled  bool    `json:"enabled"`
}

func (ctrl *Controller) List(c *gin.Context) {
	camps, err := ctrl.svc.List(c.Request.Context())
	if err != nil {
		slog.With("error", err).Error("error listing camps")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, camps)
}

func (ctrl *Controller) Get(c *gin.Context) {
	id := c.Param("campId")

	camp, err := ctrl.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "camp not found"})
			return
		}
		if api.IsInvalidUUID(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("id", id).
			With("error", err).
			Error("error getting camp")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, camp)
}

func (ctrl *Controller) Create(c *gin.Context) {
	var req CreateCampRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	camp, err := ctrl.svc.Create(c.Request.Context(), req)
	if err != nil {
		slog.With("error", err).Error("error creating camp")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, camp)
}

func (ctrl *Controller) Update(c *gin.Context) {
	id := c.Param("campId")

	var req UpdateCampRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	camp, err := ctrl.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "camp not found"})
			return
		}
		if api.IsInvalidUUID(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("id", id).
			With("error", err).
			Error("error updating camp")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, camp)
}

func (ctrl *Controller) Delete(c *gin.Context) {
	id := c.Param("campId")

	err := ctrl.svc.Delete(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "camp not found"})
			return
		}
		if api.IsInvalidUUID(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "camp has dependent records and cannot be deleted"})
			return
		}
		slog.
			With("id", id).
			With("error", err).
			Error("error deleting camp")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}
