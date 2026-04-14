package camp

import (
	"errors"
	"log/slog"
	"net/http"

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
	camp := rg.Group("/camp")
	camp.GET("", ctrl.Get)
	camp.PUT("", ctrl.Update)
}

type UpdateCampRequest struct {
	Name     string  `json:"name" binding:"required"`
	Location *string `json:"location"`
	Enabled  bool    `json:"enabled"`
}

// CampResponse is the shared response type for camp data, used by both
// the camp-scoped and admin endpoints.
type CampResponse struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Location *string `json:"location"`
	Enabled  bool    `json:"enabled"`
}

func (ctrl *Controller) Get(c *gin.Context) {
	campID := auth.GetCampID(c)

	camp, err := ctrl.svc.GetByID(c.Request.Context(), campID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "camp not found"})
			return
		}
		slog.
			With("id", campID).
			With("error", err).
			Error("error getting camp")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, camp)
}

func (ctrl *Controller) Update(c *gin.Context) {
	campID := auth.GetCampID(c)

	var req UpdateCampRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	camp, err := ctrl.svc.Update(c.Request.Context(), campID, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "camp not found"})
			return
		}
		slog.
			With("id", campID).
			With("error", err).
			Error("error updating camp")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, camp)
}
