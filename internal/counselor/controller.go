package counselor

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
	counselors := camps.Group("/:campId/counselors")
	counselors.GET("", ctrl.List)
	counselors.GET("/:counselorId", ctrl.Get)
	counselors.POST("", ctrl.Create)
	counselors.PUT("/:counselorId", ctrl.Update)
	counselors.DELETE("/:counselorId", ctrl.Delete)
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

func (ctrl *Controller) List(c *gin.Context) {
	campID := c.Param("campId")

	counselors, err := ctrl.svc.List(c.Request.Context(), campID)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("error", err).
			Error("error listing counselors")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, counselors)
}

func (ctrl *Controller) Get(c *gin.Context) {
	campID := c.Param("campId")
	id := c.Param("counselorId")

	counselor, err := ctrl.svc.GetByID(c.Request.Context(), campID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "counselor not found"})
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
			Error("error getting counselor")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, counselor)
}

func (ctrl *Controller) Create(c *gin.Context) {
	campID := c.Param("campId")

	var req CreateCounselorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	counselor, err := ctrl.svc.Create(c.Request.Context(), campID, req)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("error", err).
			Error("error creating counselor")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, counselor)
}

func (ctrl *Controller) Update(c *gin.Context) {
	campID := c.Param("campId")
	id := c.Param("counselorId")

	var req UpdateCounselorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	counselor, err := ctrl.svc.Update(c.Request.Context(), campID, id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "counselor not found"})
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
			Error("error updating counselor")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, counselor)
}

func (ctrl *Controller) Delete(c *gin.Context) {
	campID := c.Param("campId")
	id := c.Param("counselorId")

	err := ctrl.svc.Delete(c.Request.Context(), campID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "counselor not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "counselor has dependent records and cannot be deleted"})
			return
		}
		slog.
			With("camp_id", campID).
			With("id", id).
			With("error", err).
			Error("error deleting counselor")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}
