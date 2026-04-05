package preferences

import (
	"errors"
	"log/slog"
	"net/http"

	"camp-scheduler/internal/api"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type CocounselorController struct {
	svc *CocounselorService
}

func NewCocounselorController(svc *CocounselorService) *CocounselorController {
	return &CocounselorController{svc: svc}
}

func (ctrl *CocounselorController) RegisterRoutes(camps *gin.RouterGroup) {
	prefs := camps.Group("/:campId/sessions/:sessionId/counselors/:counselorId/cocounselor-preferences")
	prefs.GET("", ctrl.List)
	prefs.GET("/:id", ctrl.Get)
	prefs.POST("", ctrl.Create)
	prefs.PUT("/:id", ctrl.Update)
	prefs.DELETE("/:id", ctrl.Delete)
}

type CreateCocounselorPreferenceRequest struct {
	PreferredCounselorID string `json:"preferred_counselor_id" binding:"required"`
	Rank                 int32  `json:"rank" binding:"required,gt=0"`
}

type UpdateCocounselorPreferenceRequest struct {
	PreferredCounselorID string `json:"preferred_counselor_id" binding:"required"`
	Rank                 int32  `json:"rank" binding:"required,gt=0"`
}

type CocounselorPreferenceResponse struct {
	ID                   string `json:"id"`
	CampID               string `json:"camp_id"`
	CounselorID          string `json:"counselor_id"`
	SessionID            string `json:"session_id"`
	PreferredCounselorID string `json:"preferred_counselor_id"`
	Rank                 int32  `json:"rank"`
}

func (ctrl *CocounselorController) List(c *gin.Context) {
	campID := c.Param("campId")
	sessionID := c.Param("sessionId")
	counselorID := c.Param("counselorId")

	prefs, err := ctrl.svc.List(c.Request.Context(), campID, sessionID, counselorID)
	if err != nil {
		if api.IsInvalidUUID(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("session_id", sessionID).
			With("counselor_id", counselorID).
			With("error", err).
			Error("error listing co-counselor preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, prefs)
}

func (ctrl *CocounselorController) Get(c *gin.Context) {
	campID := c.Param("campId")
	sessionID := c.Param("sessionId")
	counselorID := c.Param("counselorId")
	id := c.Param("id")

	pref, err := ctrl.svc.GetByID(c.Request.Context(), campID, sessionID, counselorID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "co-counselor preference not found"})
			return
		}
		if api.IsInvalidUUID(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("session_id", sessionID).
			With("counselor_id", counselorID).
			With("id", id).
			With("error", err).
			Error("error getting co-counselor preference")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, pref)
}

func (ctrl *CocounselorController) Create(c *gin.Context) {
	campID := c.Param("campId")
	sessionID := c.Param("sessionId")
	counselorID := c.Param("counselorId")

	var req CreateCocounselorPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pref, err := ctrl.svc.Create(c.Request.Context(), campID, sessionID, counselorID, req)
	if err != nil {
		if api.IsInvalidUUID(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "referenced entity not found"})
			return
		}
		slog.
			With("camp_id", campID).
			With("session_id", sessionID).
			With("counselor_id", counselorID).
			With("error", err).
			Error("error creating co-counselor preference")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, pref)
}

func (ctrl *CocounselorController) Update(c *gin.Context) {
	campID := c.Param("campId")
	sessionID := c.Param("sessionId")
	counselorID := c.Param("counselorId")
	id := c.Param("id")

	var req UpdateCocounselorPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pref, err := ctrl.svc.Update(c.Request.Context(), campID, sessionID, counselorID, id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "co-counselor preference not found"})
			return
		}
		if api.IsInvalidUUID(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "referenced entity not found"})
			return
		}
		slog.
			With("camp_id", campID).
			With("session_id", sessionID).
			With("counselor_id", counselorID).
			With("id", id).
			With("error", err).
			Error("error updating co-counselor preference")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, pref)
}

func (ctrl *CocounselorController) Delete(c *gin.Context) {
	campID := c.Param("campId")
	sessionID := c.Param("sessionId")
	counselorID := c.Param("counselorId")
	id := c.Param("id")

	err := ctrl.svc.Delete(c.Request.Context(), campID, sessionID, counselorID, id)
	if err != nil {
		if errors.Is(err, ErrCocounselorPreferenceNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "co-counselor preference not found"})
			return
		}
		if api.IsInvalidUUID(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("session_id", sessionID).
			With("counselor_id", counselorID).
			With("id", id).
			With("error", err).
			Error("error deleting co-counselor preference")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}
