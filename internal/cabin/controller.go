package cabin

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
	cabins := camps.Group("/:campId/cabins")
	cabins.GET("", ctrl.List)
	cabins.GET("/:id", ctrl.Get)
	cabins.POST("", ctrl.Create)
	cabins.PUT("/:id", ctrl.Update)
	cabins.DELETE("/:id", ctrl.Delete)
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

func (ctrl *Controller) List(c *gin.Context) {
	campID := c.Param("campId")

	cabins, err := ctrl.svc.List(c.Request.Context(), campID)
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

func (ctrl *Controller) Get(c *gin.Context) {
	id := c.Param("id")

	cabin, err := ctrl.svc.GetByID(c.Request.Context(), id)
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

func (ctrl *Controller) Create(c *gin.Context) {
	campID := c.Param("campId")

	var req CreateCabinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cabin, err := ctrl.svc.Create(c.Request.Context(), campID, req)
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

func (ctrl *Controller) Update(c *gin.Context) {
	id := c.Param("id")

	var req UpdateCabinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cabin, err := ctrl.svc.Update(c.Request.Context(), id, req)
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

func (ctrl *Controller) Delete(c *gin.Context) {
	id := c.Param("id")

	err := ctrl.svc.Delete(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "cabin not found"})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "cabin has dependent records and cannot be deleted"})
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
