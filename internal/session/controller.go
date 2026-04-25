package session

import (
	"errors"
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
	sessions := rg.Group("/sessions")
	sessions.GET("", ctrl.List)
	sessions.GET("/archived", ctrl.ListArchived)
	sessions.GET("/:sessionId", ctrl.Get)
	sessions.POST("", ctrl.Create)
	sessions.PUT("/:sessionId", ctrl.Update)
	sessions.DELETE("/:sessionId", ctrl.Delete)
	sessions.POST("/:sessionId/archive", ctrl.Archive)
	sessions.POST("/:sessionId/unarchive", ctrl.Unarchive)
	sessions.POST("/:sessionId/copy", ctrl.Copy)
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

// CopySessionRequest creates a new session by structurally cloning an existing
// one (age groups, cabins, time slots, activities). The source session is
// identified by the URL.
type CopySessionRequest struct {
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
	Archived          bool    `json:"archived"`
}

func (ctrl *Controller) List(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)

	sessions, err := ctrl.svc.List(c.Request.Context(), campID)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.With("error", err).Error("error listing sessions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, sessions)
}

func (ctrl *Controller) Get(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	id := c.Param("sessionId")

	session, err := ctrl.svc.GetByID(c.Request.Context(), campID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("id", id).
			With("error", err).
			Error("error getting session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, session)
}

func (ctrl *Controller) Create(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)

	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := ctrl.svc.Create(c.Request.Context(), campID, req)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.With("error", err).Error("error creating session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, session)
}

func (ctrl *Controller) Update(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	id := c.Param("sessionId")

	var req UpdateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := ctrl.svc.Update(c.Request.Context(), campID, id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("id", id).
			With("error", err).
			Error("error updating session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, session)
}

func (ctrl *Controller) Delete(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	id := c.Param("sessionId")

	err := ctrl.svc.Delete(c.Request.Context(), campID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "session has dependent records and cannot be deleted"})
			return
		}
		log.
			With("id", id).
			With("error", err).
			Error("error deleting session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}

func (ctrl *Controller) Copy(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sourceID := c.Param("sessionId")

	var req CopySessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := ctrl.svc.Copy(c.Request.Context(), campID, sourceID, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if api.IsUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "a session with that name already exists in this season"})
			return
		}
		log.
			With("source_session_id", sourceID).
			With("error", err).
			Error("error copying session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, session)
}

func (ctrl *Controller) ListArchived(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)

	items, err := ctrl.svc.ListArchived(c.Request.Context(), campID)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("error", err).
			Error("error listing archived sessions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, items)
}

func (ctrl *Controller) Archive(c *gin.Context) {
	ctrl.setArchived(c, true)
}

func (ctrl *Controller) Unarchive(c *gin.Context) {
	ctrl.setArchived(c, false)
}

func (ctrl *Controller) setArchived(c *gin.Context, archived bool) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	id := c.Param("sessionId")

	var err error
	if archived {
		err = ctrl.svc.Archive(c.Request.Context(), campID, id)
	} else {
		err = ctrl.svc.Unarchive(c.Request.Context(), campID, id)
	}
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("id", id).
			With("archived", archived).
			With("error", err).
			Error("error setting session archived")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}
