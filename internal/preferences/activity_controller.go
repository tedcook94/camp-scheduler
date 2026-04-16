package preferences

import (
	"errors"
	"net/http"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type ActivityPreferenceController struct {
	svc *ActivityPreferenceService
}

func NewActivityPreferenceController(svc *ActivityPreferenceService) *ActivityPreferenceController {
	return &ActivityPreferenceController{svc: svc}
}

func (ctrl *ActivityPreferenceController) RegisterRoutes(rg *gin.RouterGroup) {
	prefs := rg.Group("/sessions/:sessionId/counselors/:counselorId/activity-preferences")
	prefs.GET("", ctrl.List)
	prefs.GET("/:id", ctrl.Get)
	prefs.POST("", ctrl.Create)
	prefs.PUT("/:id", ctrl.Update)
	prefs.DELETE("/:id", ctrl.Delete)
}

type CreateActivityPreferenceRequest struct {
	ActivityID string `json:"activity_id" binding:"required"`
	Rank       int32  `json:"rank" binding:"required,gt=0"`
}

type UpdateActivityPreferenceRequest struct {
	ActivityID string `json:"activity_id" binding:"required"`
	Rank       int32  `json:"rank" binding:"required,gt=0"`
}

type ActivityPreferenceResponse struct {
	ID          string `json:"id"`
	CampID      string `json:"camp_id"`
	CounselorID string `json:"counselor_id"`
	SessionID   string `json:"session_id"`
	ActivityID  string `json:"activity_id"`
	Rank        int32  `json:"rank"`
}

func (ctrl *ActivityPreferenceController) List(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	counselorID := c.Param("counselorId")

	prefs, err := ctrl.svc.List(c.Request.Context(), campID, sessionID, counselorID)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("session_id", sessionID).
			With("counselor_id", counselorID).
			With("error", err).
			Error("error listing activity preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, prefs)
}

func (ctrl *ActivityPreferenceController) Get(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	counselorID := c.Param("counselorId")
	id := c.Param("id")

	pref, err := ctrl.svc.GetByID(c.Request.Context(), campID, sessionID, counselorID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "activity preference not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("session_id", sessionID).
			With("counselor_id", counselorID).
			With("id", id).
			With("error", err).
			Error("error getting activity preference")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, pref)
}

func (ctrl *ActivityPreferenceController) Create(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	counselorID := c.Param("counselorId")

	var req CreateActivityPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pref, err := ctrl.svc.Create(c.Request.Context(), campID, sessionID, counselorID, req)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "referenced entity not found"})
			return
		}
		if api.IsUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "activity preference already exists for this counselor and activity"})
			return
		}
		log.
			With("session_id", sessionID).
			With("counselor_id", counselorID).
			With("error", err).
			Error("error creating activity preference")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, pref)
}

func (ctrl *ActivityPreferenceController) Update(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	counselorID := c.Param("counselorId")
	id := c.Param("id")

	var req UpdateActivityPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pref, err := ctrl.svc.Update(c.Request.Context(), campID, sessionID, counselorID, id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "activity preference not found"})
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
		if api.IsUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "activity preference already exists for this counselor and activity"})
			return
		}
		log.
			With("session_id", sessionID).
			With("counselor_id", counselorID).
			With("id", id).
			With("error", err).
			Error("error updating activity preference")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, pref)
}

func (ctrl *ActivityPreferenceController) Delete(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	counselorID := c.Param("counselorId")
	id := c.Param("id")

	err := ctrl.svc.Delete(c.Request.Context(), campID, sessionID, counselorID, id)
	if err != nil {
		if errors.Is(err, ErrActivityPreferenceNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "activity preference not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("session_id", sessionID).
			With("counselor_id", counselorID).
			With("id", id).
			With("error", err).
			Error("error deleting activity preference")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}
