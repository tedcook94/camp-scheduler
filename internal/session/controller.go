package session

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
	sessions := camps.Group("/:campId/sessions")
	sessions.GET("", ctrl.List)
	sessions.GET("/:id", ctrl.Get)
	sessions.POST("", ctrl.Create)
	sessions.PUT("/:id", ctrl.Update)
	sessions.DELETE("/:id", ctrl.Delete)
}

type CreateSessionRequest struct {
	Name              string  `json:"name" binding:"required"`
	SeasonID          string  `json:"season_id" binding:"required"`
	PreviousSessionID *string `json:"previous_session_id"`
}

type UpdateSessionRequest struct {
	Name              string  `json:"name" binding:"required"`
	SeasonID          string  `json:"season_id" binding:"required"`
	PreviousSessionID *string `json:"previous_session_id"`
}

type SessionResponse struct {
	ID                string  `json:"id"`
	CampID            string  `json:"camp_id"`
	SeasonID          string  `json:"season_id"`
	Name              string  `json:"name"`
	PreviousSessionID *string `json:"previous_session_id"`
}

func (ctrl *Controller) List(c *gin.Context) {
	campID := c.Param("campId")

	sessions, err := ctrl.svc.List(c.Request.Context(), campID)
	if err != nil {
		slog.
			With("camp_id", campID).
			With("error", err).
			Error("error listing sessions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, sessions)
}

func (ctrl *Controller) Get(c *gin.Context) {
	id := c.Param("id")

	session, err := ctrl.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		slog.
			With("id", id).
			With("error", err).
			Error("error getting session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, session)
}

func (ctrl *Controller) Create(c *gin.Context) {
	campID := c.Param("campId")

	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := ctrl.svc.Create(c.Request.Context(), campID, req)
	if err != nil {
		slog.
			With("camp_id", campID).
			With("error", err).
			Error("error creating session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, session)
}

func (ctrl *Controller) Update(c *gin.Context) {
	id := c.Param("id")

	var req UpdateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := ctrl.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		slog.
			With("id", id).
			With("error", err).
			Error("error updating session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, session)
}

func (ctrl *Controller) Delete(c *gin.Context) {
	id := c.Param("id")

	err := ctrl.svc.Delete(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "session has dependent records and cannot be deleted"})
			return
		}
		slog.
			With("id", id).
			With("error", err).
			Error("error deleting session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}
