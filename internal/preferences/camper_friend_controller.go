package preferences

import (
	"errors"
	"net/http"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
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
	prefs.GET("/:id", ctrl.Get)
	prefs.POST("", ctrl.Create)
	prefs.PUT("/:id", ctrl.Update)
	prefs.DELETE("/:id", ctrl.Delete)
}

type CreateCamperFriendPreferenceRequest struct {
	PreferredCamperID string `json:"preferred_camper_id" binding:"required"`
	Rank              int32  `json:"rank" binding:"required,gt=0"`
}

type UpdateCamperFriendPreferenceRequest struct {
	PreferredCamperID string `json:"preferred_camper_id" binding:"required"`
	Rank              int32  `json:"rank" binding:"required,gt=0"`
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

func (ctrl *CamperFriendController) Get(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	camperID := c.Param("camperId")
	id := c.Param("id")

	pref, err := ctrl.svc.GetByID(c.Request.Context(), campID, sessionID, camperID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "camper friend preference not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("session_id", sessionID).
			With("camper_id", camperID).
			With("id", id).
			With("error", err).
			Error("error getting camper friend preference")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, pref)
}

func (ctrl *CamperFriendController) Create(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	camperID := c.Param("camperId")

	var req CreateCamperFriendPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pref, err := ctrl.svc.Create(c.Request.Context(), campID, sessionID, camperID, req)
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
			c.JSON(http.StatusConflict, gin.H{"error": "duplicate preference: camper+session+rank or camper+session+preferred_camper already exists"})
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
			Error("error creating camper friend preference")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, pref)
}

func (ctrl *CamperFriendController) Update(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	camperID := c.Param("camperId")
	id := c.Param("id")

	var req UpdateCamperFriendPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pref, err := ctrl.svc.Update(c.Request.Context(), campID, sessionID, camperID, id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "camper friend preference not found"})
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
			c.JSON(http.StatusConflict, gin.H{"error": "duplicate preference: camper+session+rank or camper+session+preferred_camper already exists"})
			return
		}
		if api.IsCheckViolation(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid preference: a camper cannot prefer themselves"})
			return
		}
		log.
			With("session_id", sessionID).
			With("camper_id", camperID).
			With("id", id).
			With("error", err).
			Error("error updating camper friend preference")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, pref)
}

func (ctrl *CamperFriendController) Delete(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	camperID := c.Param("camperId")
	id := c.Param("id")

	err := ctrl.svc.Delete(c.Request.Context(), campID, sessionID, camperID, id)
	if err != nil {
		if errors.Is(err, ErrCamperFriendPreferenceNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "camper friend preference not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("session_id", sessionID).
			With("camper_id", camperID).
			With("id", id).
			With("error", err).
			Error("error deleting camper friend preference")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}
