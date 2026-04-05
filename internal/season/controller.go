package season

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
	seasons := camps.Group("/:campId/seasons")
	seasons.GET("", ctrl.List)
	seasons.GET("/:id", ctrl.Get)
	seasons.POST("", ctrl.Create)
	seasons.PUT("/:id", ctrl.Update)
	seasons.DELETE("/:id", ctrl.Delete)
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

func (ctrl *Controller) List(c *gin.Context) {
	campID := c.Param("campId")

	seasons, err := ctrl.svc.List(c.Request.Context(), campID)
	if err != nil {
		if api.IsInvalidUUID(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("error", err).
			Error("error listing seasons")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, seasons)
}

func (ctrl *Controller) Get(c *gin.Context) {
	campID := c.Param("campId")
	id := c.Param("id")

	season, err := ctrl.svc.GetByID(c.Request.Context(), campID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "season not found"})
			return
		}
		if api.IsInvalidUUID(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("id", id).
			With("error", err).
			Error("error getting season")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, season)
}

func (ctrl *Controller) Create(c *gin.Context) {
	campID := c.Param("campId")

	var req CreateSeasonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	season, err := ctrl.svc.Create(c.Request.Context(), campID, req)
	if err != nil {
		if api.IsInvalidUUID(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("error", err).
			Error("error creating season")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, season)
}

func (ctrl *Controller) Update(c *gin.Context) {
	campID := c.Param("campId")
	id := c.Param("id")

	var req UpdateSeasonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	season, err := ctrl.svc.Update(c.Request.Context(), campID, id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "season not found"})
			return
		}
		if api.IsInvalidUUID(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("id", id).
			With("error", err).
			Error("error updating season")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, season)
}

func (ctrl *Controller) Delete(c *gin.Context) {
	campID := c.Param("campId")
	id := c.Param("id")

	err := ctrl.svc.Delete(c.Request.Context(), campID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "season not found"})
			return
		}
		if api.IsInvalidUUID(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "season has dependent records and cannot be deleted"})
			return
		}
		slog.
			With("camp_id", campID).
			With("id", id).
			With("error", err).
			Error("error deleting season")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}
