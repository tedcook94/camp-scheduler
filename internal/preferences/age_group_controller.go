package preferences

import (
	"net/http"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/auth"

	"github.com/gin-gonic/gin"
)

type AgeGroupController struct {
	svc *AgeGroupService
}

func NewAgeGroupController(svc *AgeGroupService) *AgeGroupController {
	return &AgeGroupController{svc: svc}
}

func (ctrl *AgeGroupController) RegisterRoutes(rg *gin.RouterGroup) {
	prefs := rg.Group("/sessions/:sessionId/counselors/:counselorId/age-group-preferences")
	prefs.GET("", ctrl.List)
	prefs.PUT("", ctrl.ReplaceAll)
}

type AgeGroupPreferenceResponse struct {
	ID          string `json:"id"`
	CampID      string `json:"camp_id"`
	CounselorID string `json:"counselor_id"`
	SessionID   string `json:"session_id"`
	AgeGroupID  string `json:"age_group_id"`
	Rank        int32  `json:"rank"`
}

func (ctrl *AgeGroupController) List(c *gin.Context) {
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
			Error("error listing age group preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, prefs)
}

func (ctrl *AgeGroupController) ReplaceAll(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	counselorID := c.Param("counselorId")

	var items []AgeGroupPreferenceItem
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
		log.
			With("session_id", sessionID).
			With("counselor_id", counselorID).
			With("error", err).
			Error("error replacing age group preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, prefs)
}
