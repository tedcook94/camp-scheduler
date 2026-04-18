package preferences

import (
	"net/http"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/auth"

	"github.com/gin-gonic/gin"
)

type CocounselorController struct {
	svc *CocounselorService
}

func NewCocounselorController(svc *CocounselorService) *CocounselorController {
	return &CocounselorController{svc: svc}
}

func (ctrl *CocounselorController) RegisterRoutes(rg *gin.RouterGroup) {
	prefs := rg.Group("/sessions/:sessionId/counselors/:counselorId/cocounselor-preferences")
	prefs.GET("", ctrl.List)
	prefs.PUT("", ctrl.ReplaceAll)
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
			Error("error listing co-counselor preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, prefs)
}

func (ctrl *CocounselorController) ReplaceAll(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	counselorID := c.Param("counselorId")

	var items []CocounselorPreferenceItem
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
		if api.IsCheckViolation(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "a counselor cannot prefer themselves"})
			return
		}
		log.
			With("session_id", sessionID).
			With("counselor_id", counselorID).
			With("error", err).
			Error("error replacing co-counselor preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, prefs)
}
