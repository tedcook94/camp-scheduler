package preferences

import (
	"net/http"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/auth"

	"github.com/gin-gonic/gin"
)

type CamperFriendController struct {
	svc *CamperFriendService
}

func NewCamperFriendController(svc *CamperFriendService) *CamperFriendController {
	return &CamperFriendController{svc: svc}
}

func (ctrl *CamperFriendController) RegisterRoutes(rg *gin.RouterGroup) {
	prefs := rg.Group("/sessions/:sessionId/campers/:camperId/friend-preferences")
	prefs.GET("", ctrl.List)
	prefs.PUT("", ctrl.ReplaceAll)
}

type CamperFriendPreferenceResponse struct {
	ID                string `json:"id"`
	CampID            string `json:"camp_id"`
	CamperID          string `json:"camper_id"`
	SessionID         string `json:"session_id"`
	PreferredCamperID string `json:"preferred_camper_id"`
	Rank              int32  `json:"rank"`
}

func (ctrl *CamperFriendController) List(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	camperID := c.Param("camperId")

	prefs, err := ctrl.svc.List(c.Request.Context(), campID, sessionID, camperID)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("session_id", sessionID).
			With("camper_id", camperID).
			With("error", err).
			Error("error listing camper friend preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, prefs)
}

func (ctrl *CamperFriendController) ReplaceAll(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	camperID := c.Param("camperId")

	var items []CamperFriendPreferenceItem
	if err := c.ShouldBindJSON(&items); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prefs, err := ctrl.svc.ReplaceAll(c.Request.Context(), campID, sessionID, camperID, items)
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
			c.JSON(http.StatusConflict, gin.H{"error": "duplicate preference"})
			return
		}
		if api.IsCheckViolation(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid preference: a camper cannot prefer themselves"})
			return
		}
		log.
			With("session_id", sessionID).
			With("camper_id", camperID).
			With("error", err).
			Error("error replacing camper friend preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, prefs)
}
