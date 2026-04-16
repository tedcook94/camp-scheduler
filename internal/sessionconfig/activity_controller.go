package sessionconfig

import (
	"errors"
	"net/http"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type ActivityController struct {
	svc *ActivityService
}

func NewActivityController(svc *ActivityService) *ActivityController {
	return &ActivityController{svc: svc}
}

func (ctrl *ActivityController) RegisterRoutes(rg *gin.RouterGroup) {
	timeSlots := rg.Group("/sessions/:sessionId/time-slots")
	timeSlots.GET("", ctrl.ListTimeSlots)
	timeSlots.GET("/:timeSlotId", ctrl.GetTimeSlot)
	timeSlots.POST("", ctrl.CreateTimeSlot)
	timeSlots.PUT("/:timeSlotId", ctrl.UpdateTimeSlot)
	timeSlots.DELETE("/:timeSlotId", ctrl.DeleteTimeSlot)

	activities := timeSlots.Group("/:timeSlotId/activities")
	activities.GET("", ctrl.ListActivities)
	activities.GET("/:activityId", ctrl.GetActivity)
	activities.POST("", ctrl.CreateActivity)
	activities.PUT("/:activityId", ctrl.UpdateActivity)
	activities.DELETE("/:activityId", ctrl.DeleteActivity)
}

type CreateSessionTimeSlotRequest struct {
	TimeSlotID string `json:"time_slot_id" binding:"required"`
	SortOrder  int32  `json:"sort_order"`
}

type UpdateSessionTimeSlotRequest struct {
	TimeSlotID string `json:"time_slot_id" binding:"required"`
	SortOrder  int32  `json:"sort_order"`
}

type SessionTimeSlotResponse struct {
	ID         string `json:"id"`
	CampID     string `json:"camp_id"`
	SessionID  string `json:"session_id"`
	TimeSlotID string `json:"time_slot_id"`
	SortOrder  int32  `json:"sort_order"`
}

type CreateSessionActivityRequest struct {
	ActivityID         string `json:"activity_id" binding:"required"`
	Capacity           int32  `json:"capacity" binding:"required,gt=0"`
	RequiredCounselors int32  `json:"required_counselors" binding:"required,gt=0"`
}

type UpdateSessionActivityRequest struct {
	ActivityID         string `json:"activity_id" binding:"required"`
	Capacity           int32  `json:"capacity" binding:"required,gt=0"`
	RequiredCounselors int32  `json:"required_counselors" binding:"required,gt=0"`
}

type SessionActivityResponse struct {
	ID                 string `json:"id"`
	CampID             string `json:"camp_id"`
	SessionTimeSlotID  string `json:"session_time_slot_id"`
	ActivityID         string `json:"activity_id"`
	Capacity           int32  `json:"capacity"`
	RequiredCounselors int32  `json:"required_counselors"`
}

func (ctrl *ActivityController) ListTimeSlots(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")

	slots, err := ctrl.svc.ListTimeSlots(c.Request.Context(), campID, sessionID)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("session_id", sessionID).
			With("error", err).
			Error("error listing session time slots")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, slots)
}

func (ctrl *ActivityController) GetTimeSlot(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	id := c.Param("timeSlotId")

	slot, err := ctrl.svc.GetTimeSlot(c.Request.Context(), campID, sessionID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session time slot not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("session_id", sessionID).
			With("id", id).
			With("error", err).
			Error("error getting session time slot")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, slot)
}

func (ctrl *ActivityController) CreateTimeSlot(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")

	var req CreateSessionTimeSlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slot, err := ctrl.svc.CreateTimeSlot(c.Request.Context(), campID, sessionID, req)
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
			c.JSON(http.StatusConflict, gin.H{"error": "time slot already added to session"})
			return
		}
		log.
			With("session_id", sessionID).
			With("error", err).
			Error("error creating session time slot")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, slot)
}

func (ctrl *ActivityController) UpdateTimeSlot(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	id := c.Param("timeSlotId")

	var req UpdateSessionTimeSlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slot, err := ctrl.svc.UpdateTimeSlot(c.Request.Context(), campID, sessionID, id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session time slot not found"})
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
			c.JSON(http.StatusConflict, gin.H{"error": "time slot already added to session"})
			return
		}
		log.
			With("session_id", sessionID).
			With("id", id).
			With("error", err).
			Error("error updating session time slot")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, slot)
}

func (ctrl *ActivityController) DeleteTimeSlot(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	id := c.Param("timeSlotId")

	err := ctrl.svc.DeleteTimeSlot(c.Request.Context(), campID, sessionID, id)
	if err != nil {
		if errors.Is(err, ErrTimeSlotNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session time slot not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "session time slot has dependent records and cannot be deleted"})
			return
		}
		log.
			With("session_id", sessionID).
			With("id", id).
			With("error", err).
			Error("error deleting session time slot")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}

func (ctrl *ActivityController) ListActivities(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	timeSlotID := c.Param("timeSlotId")

	activities, err := ctrl.svc.ListActivities(c.Request.Context(), campID, sessionID, timeSlotID)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session time slot not found for this session"})
			return
		}
		log.
			With("session_id", sessionID).
			With("time_slot_id", timeSlotID).
			With("error", err).
			Error("error listing session activities")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, activities)
}

func (ctrl *ActivityController) GetActivity(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	id := c.Param("activityId")

	activity, err := ctrl.svc.GetActivity(c.Request.Context(), campID, sessionID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session activity not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("session_id", sessionID).
			With("id", id).
			With("error", err).
			Error("error getting session activity")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, activity)
}

func (ctrl *ActivityController) CreateActivity(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	timeSlotID := c.Param("timeSlotId")

	var req CreateSessionActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	activity, err := ctrl.svc.CreateActivity(c.Request.Context(), campID, sessionID, timeSlotID, req)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session time slot not found for this session"})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "referenced entity not found"})
			return
		}
		if api.IsUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "activity already added to this time slot"})
			return
		}
		if api.IsCheckViolation(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "capacity and required_counselors must be positive"})
			return
		}
		log.
			With("session_id", sessionID).
			With("time_slot_id", timeSlotID).
			With("error", err).
			Error("error creating session activity")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, activity)
}

func (ctrl *ActivityController) UpdateActivity(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	id := c.Param("activityId")

	var req UpdateSessionActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	activity, err := ctrl.svc.UpdateActivity(c.Request.Context(), campID, sessionID, id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session activity not found"})
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
		if api.IsCheckViolation(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "capacity and required_counselors must be positive"})
			return
		}
		if api.IsUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "activity already added to this time slot"})
			return
		}
		log.
			With("session_id", sessionID).
			With("id", id).
			With("error", err).
			Error("error updating session activity")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, activity)
}

func (ctrl *ActivityController) DeleteActivity(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	id := c.Param("activityId")

	err := ctrl.svc.DeleteActivity(c.Request.Context(), campID, sessionID, id)
	if err != nil {
		if errors.Is(err, ErrSessionActivityNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session activity not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "session activity has dependent records and cannot be deleted"})
			return
		}
		log.
			With("session_id", sessionID).
			With("id", id).
			With("error", err).
			Error("error deleting session activity")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}
