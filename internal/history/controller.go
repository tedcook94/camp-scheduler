package history

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
	history := camps.Group("/:campId/counselors/:counselorId/session-history")
	history.GET("", ctrl.List)
	history.GET("/:id", ctrl.Get)
	history.POST("", ctrl.Create)
	history.PUT("/:id", ctrl.Update)
	history.DELETE("/:id", ctrl.Delete)

	counselor := camps.Group("/:campId/counselors/:counselorId")
	counselor.GET("/history", ctrl.GetHistorySummary)
}

type CreateSessionHistoryRequest struct {
	SessionID  string  `json:"session_id" binding:"required"`
	AgeGroupID string  `json:"age_group_id" binding:"required"`
	CabinID    *string `json:"cabin_id"`
}

type UpdateSessionHistoryRequest struct {
	SessionID  string  `json:"session_id" binding:"required"`
	AgeGroupID string  `json:"age_group_id" binding:"required"`
	CabinID    *string `json:"cabin_id"`
}

type SessionHistoryResponse struct {
	ID          string  `json:"id"`
	CampID      string  `json:"camp_id"`
	CounselorID string  `json:"counselor_id"`
	SessionID   string  `json:"session_id"`
	AgeGroupID  string  `json:"age_group_id"`
	CabinID     *string `json:"cabin_id"`
}

type HistorySummaryEntry struct {
	ID           string  `json:"id"`
	SessionName  string  `json:"session_name"`
	SeasonID     string  `json:"season_id"`
	SeasonName   string  `json:"season_name"`
	AgeGroupName string  `json:"age_group_name"`
	CabinName    *string `json:"cabin_name"`
}

func (ctrl *Controller) List(c *gin.Context) {
	campID := c.Param("campId")
	counselorID := c.Param("counselorId")

	entries, err := ctrl.svc.List(c.Request.Context(), campID, counselorID)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("counselor_id", counselorID).
			With("error", err).
			Error("error listing session history")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, entries)
}

func (ctrl *Controller) Get(c *gin.Context) {
	campID := c.Param("campId")
	counselorID := c.Param("counselorId")
	id := c.Param("id")

	entry, err := ctrl.svc.GetByID(c.Request.Context(), campID, counselorID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session history entry not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("counselor_id", counselorID).
			With("id", id).
			With("error", err).
			Error("error getting session history entry")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, entry)
}

func (ctrl *Controller) Create(c *gin.Context) {
	campID := c.Param("campId")
	counselorID := c.Param("counselorId")

	var req CreateSessionHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entry, err := ctrl.svc.Create(c.Request.Context(), campID, counselorID, req)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "referenced entity not found"})
			return
		}
		slog.
			With("camp_id", campID).
			With("counselor_id", counselorID).
			With("error", err).
			Error("error creating session history entry")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, entry)
}

func (ctrl *Controller) Update(c *gin.Context) {
	campID := c.Param("campId")
	counselorID := c.Param("counselorId")
	id := c.Param("id")

	var req UpdateSessionHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entry, err := ctrl.svc.Update(c.Request.Context(), campID, counselorID, id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session history entry not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "referenced entity not found"})
			return
		}
		slog.
			With("camp_id", campID).
			With("counselor_id", counselorID).
			With("id", id).
			With("error", err).
			Error("error updating session history entry")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, entry)
}

func (ctrl *Controller) Delete(c *gin.Context) {
	campID := c.Param("campId")
	counselorID := c.Param("counselorId")
	id := c.Param("id")

	err := ctrl.svc.Delete(c.Request.Context(), campID, counselorID, id)
	if err != nil {
		if errors.Is(err, ErrSessionHistoryNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session history entry not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("counselor_id", counselorID).
			With("id", id).
			With("error", err).
			Error("error deleting session history entry")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}

func (ctrl *Controller) GetHistorySummary(c *gin.Context) {
	campID := c.Param("campId")
	counselorID := c.Param("counselorId")

	var seasonID *string
	if s := c.Query("season_id"); s != "" {
		seasonID = &s
	}

	entries, err := ctrl.svc.GetHistory(c.Request.Context(), campID, counselorID, seasonID)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("counselor_id", counselorID).
			With("error", err).
			Error("error getting counselor history summary")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, entries)
}
