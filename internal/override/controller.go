// Package override manages admin-pinned assignments that the solver treats as
// forced placements. Three subtypes share a single REST surface scoped to a
// session: counselor-cabin, camper-cabin, and counselor-activity overrides.
package override

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
	overrides := rg.Group("/sessions/:sessionId/overrides")
	overrides.GET("", ctrl.List)

	overrides.POST("/counselor-cabin", ctrl.CreateCounselorCabin)
	overrides.DELETE("/counselor-cabin/:overrideId", ctrl.DeleteCounselorCabin)

	overrides.POST("/camper-cabin", ctrl.CreateCamperCabin)
	overrides.DELETE("/camper-cabin/:overrideId", ctrl.DeleteCamperCabin)

	overrides.POST("/counselor-activity", ctrl.CreateCounselorActivity)
	overrides.DELETE("/counselor-activity/:overrideId", ctrl.DeleteCounselorActivity)
}

type CreateCounselorCabinRequest struct {
	CounselorID            string `json:"counselor_id" binding:"required"`
	SessionAgeGroupCabinID string `json:"session_age_group_cabin_id" binding:"required"`
}

type CreateCamperCabinRequest struct {
	CamperID               string `json:"camper_id" binding:"required"`
	SessionAgeGroupCabinID string `json:"session_age_group_cabin_id" binding:"required"`
}

type CreateCounselorActivityRequest struct {
	CounselorID       string `json:"counselor_id" binding:"required"`
	SessionActivityID string `json:"session_activity_id" binding:"required"`
}

type CounselorCabinOverrideResponse struct {
	ID                     string `json:"id"`
	CampID                 string `json:"camp_id"`
	SessionID              string `json:"session_id"`
	CounselorID            string `json:"counselor_id"`
	SessionAgeGroupCabinID string `json:"session_age_group_cabin_id"`
	CabinID                string `json:"cabin_id"`
	CounselorFirstName     string `json:"counselor_first_name"`
	CounselorLastName      string `json:"counselor_last_name"`
	CounselorName          string `json:"counselor_name"`
	CabinName              string `json:"cabin_name"`
	AgeGroupName           string `json:"age_group_name"`
}

type CamperCabinOverrideResponse struct {
	ID                     string `json:"id"`
	CampID                 string `json:"camp_id"`
	SessionID              string `json:"session_id"`
	CamperID               string `json:"camper_id"`
	SessionAgeGroupCabinID string `json:"session_age_group_cabin_id"`
	CabinID                string `json:"cabin_id"`
	CamperFirstName        string `json:"camper_first_name"`
	CamperLastName         string `json:"camper_last_name"`
	CamperName             string `json:"camper_name"`
	CabinName              string `json:"cabin_name"`
	AgeGroupName           string `json:"age_group_name"`
}

type CounselorActivityOverrideResponse struct {
	ID                 string `json:"id"`
	CampID             string `json:"camp_id"`
	SessionID          string `json:"session_id"`
	CounselorID        string `json:"counselor_id"`
	SessionActivityID  string `json:"session_activity_id"`
	SessionTimeSlotID  string `json:"session_time_slot_id"`
	CounselorFirstName string `json:"counselor_first_name"`
	CounselorLastName  string `json:"counselor_last_name"`
	CounselorName      string `json:"counselor_name"`
	ActivityName       string `json:"activity_name"`
	TimeSlotName       string `json:"time_slot_name"`
}

type SessionOverridesResponse struct {
	CounselorCabin     []CounselorCabinOverrideResponse    `json:"counselor_cabin"`
	CamperCabin        []CamperCabinOverrideResponse       `json:"camper_cabin"`
	CounselorActivity  []CounselorActivityOverrideResponse `json:"counselor_activity"`
}

func (ctrl *Controller) List(c *gin.Context) {
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")

	resp, err := ctrl.svc.ListBySession(c.Request.Context(), campID, sessionID)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		auth.Logger(c).
			With("session_id", sessionID).
			With("error", err).
			Error("error listing overrides")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (ctrl *Controller) CreateCounselorCabin(c *gin.Context) {
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")

	var req CreateCounselorCabinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := ctrl.svc.CreateCounselorCabin(c.Request.Context(), campID, sessionID, req)
	if err != nil {
		handleCreateError(c, "counselor-cabin override", sessionID, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (ctrl *Controller) DeleteCounselorCabin(c *gin.Context) {
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	id := c.Param("overrideId")

	if err := ctrl.svc.DeleteCounselorCabin(c.Request.Context(), campID, sessionID, id); err != nil {
		handleDeleteError(c, "counselor-cabin override", sessionID, id, err)
		return
	}
	c.Status(http.StatusOK)
}

func (ctrl *Controller) CreateCamperCabin(c *gin.Context) {
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")

	var req CreateCamperCabinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := ctrl.svc.CreateCamperCabin(c.Request.Context(), campID, sessionID, req)
	if err != nil {
		handleCreateError(c, "camper-cabin override", sessionID, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (ctrl *Controller) DeleteCamperCabin(c *gin.Context) {
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	id := c.Param("overrideId")

	if err := ctrl.svc.DeleteCamperCabin(c.Request.Context(), campID, sessionID, id); err != nil {
		handleDeleteError(c, "camper-cabin override", sessionID, id, err)
		return
	}
	c.Status(http.StatusOK)
}

func (ctrl *Controller) CreateCounselorActivity(c *gin.Context) {
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")

	var req CreateCounselorActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := ctrl.svc.CreateCounselorActivity(c.Request.Context(), campID, sessionID, req)
	if err != nil {
		handleCreateError(c, "counselor-activity override", sessionID, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (ctrl *Controller) DeleteCounselorActivity(c *gin.Context) {
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	id := c.Param("overrideId")

	if err := ctrl.svc.DeleteCounselorActivity(c.Request.Context(), campID, sessionID, id); err != nil {
		handleDeleteError(c, "counselor-activity override", sessionID, id, err)
		return
	}
	c.Status(http.StatusOK)
}

func handleCreateError(c *gin.Context, label, sessionID string, err error) {
	if api.IsBadInput(err) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "referenced entity not found"})
		return
	}
	if api.IsFKViolation(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "referenced entity not found"})
		return
	}
	if api.IsUniqueViolation(err) {
		c.JSON(http.StatusConflict, gin.H{"error": label + " already exists for this entity in this session"})
		return
	}
	auth.Logger(c).
		With("session_id", sessionID).
		With("error", err).
		Error("error creating " + label)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}

func handleDeleteError(c *gin.Context, label, sessionID, id string, err error) {
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": label + " not found"})
		return
	}
	if api.IsBadInput(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	auth.Logger(c).
		With("session_id", sessionID).
		With("id", id).
		With("error", err).
		Error("error deleting " + label)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
