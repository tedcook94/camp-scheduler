package camper

import (
	"errors"
	"log/slog"
	"net/http"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type Controller struct {
	svc *Service
}

func NewController(svc *Service) *Controller {
	return &Controller{svc: svc}
}

func (ctrl *Controller) RegisterRoutes(rg *gin.RouterGroup) {
	campers := rg.Group("/campers")
	campers.GET("", ctrl.List)
	campers.GET("/:camperId", ctrl.Get)
	campers.POST("", ctrl.Create)
	campers.PUT("/:camperId", ctrl.Update)
	campers.DELETE("/:camperId", ctrl.Delete)
}

type CreateCamperRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateCamperRequest struct {
	Name string `json:"name" binding:"required"`
}

type CamperResponse struct {
	ID     string `json:"id"`
	CampID string `json:"camp_id"`
	Name   string `json:"name"`
}

func (ctrl *Controller) List(c *gin.Context) {
	campID := auth.GetCampID(c)

	campers, err := ctrl.svc.List(c.Request.Context(), campID)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("error", err).
			Error("error listing campers")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, campers)
}

func (ctrl *Controller) Get(c *gin.Context) {
	campID := auth.GetCampID(c)
	id := c.Param("camperId")

	camper, err := ctrl.svc.GetByID(c.Request.Context(), campID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "camper not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("id", id).
			With("error", err).
			Error("error getting camper")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, camper)
}

func (ctrl *Controller) Create(c *gin.Context) {
	campID := auth.GetCampID(c)

	var req CreateCamperRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	camper, err := ctrl.svc.Create(c.Request.Context(), campID, req)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("error", err).
			Error("error creating camper")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, camper)
}

func (ctrl *Controller) Update(c *gin.Context) {
	campID := auth.GetCampID(c)
	id := c.Param("camperId")

	var req UpdateCamperRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	camper, err := ctrl.svc.Update(c.Request.Context(), campID, id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "camper not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("id", id).
			With("error", err).
			Error("error updating camper")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, camper)
}

func (ctrl *Controller) Delete(c *gin.Context) {
	campID := auth.GetCampID(c)
	id := c.Param("camperId")

	err := ctrl.svc.Delete(c.Request.Context(), campID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "camper not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "camper has dependent records and cannot be deleted"})
			return
		}
		slog.
			With("camp_id", campID).
			With("id", id).
			With("error", err).
			Error("error deleting camper")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}
