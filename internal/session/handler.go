package session

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
	sessions := camps.Group("/:campId/sessions")
	sessions.GET("", h.List)
	sessions.GET("/:id", h.Get)
	sessions.POST("", h.Create)
	sessions.PUT("/:id", h.Update)
	sessions.DELETE("/:id", h.Delete)
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

func (h *Handler) List(c *gin.Context) {
	campID := c.Param("campId")

	sessions, err := h.svc.List(c.Request.Context(), campID)
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

func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")

	session, err := h.svc.GetByID(c.Request.Context(), id)
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

func (h *Handler) Create(c *gin.Context) {
	campID := c.Param("campId")

	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := h.svc.Create(c.Request.Context(), campID, req)
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

func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")

	var req UpdateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := h.svc.Update(c.Request.Context(), id, req)
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

func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")

	err := h.svc.Delete(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
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
