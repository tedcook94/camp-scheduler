package preferences

import (
	"net/http"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/auth"

	"github.com/gin-gonic/gin"
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
	prefs.PUT("", ctrl.ReplaceAll)
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

func (ctrl *ActivityPreferenceController) ReplaceAll(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	counselorID := c.Param("counselorId")

	var items []ActivityPreferenceItem
	if err := c.ShouldBindJSON(&items); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prefs, err := ctrl.svc.ReplaceAll(c.Request.Context(), campID, sessionID, counselorID, items)
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
			c.JSON(http.StatusConflict, gin.H{"error": "duplicate activity preference"})
			return
		}
		log.
			With("session_id", sessionID).
			With("counselor_id", counselorID).
			With("error", err).
			Error("error replacing activity preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, prefs)
}
