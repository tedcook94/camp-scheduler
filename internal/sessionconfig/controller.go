package sessionconfig

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
	ageGroups := rg.Group("/sessions/:sessionId/age-groups")
	ageGroups.GET("", ctrl.ListAgeGroups)
	ageGroups.GET("/:id", ctrl.GetAgeGroup)
	ageGroups.POST("", ctrl.CreateAgeGroup)
	ageGroups.PUT("/:id", ctrl.UpdateAgeGroup)
	ageGroups.DELETE("/:id", ctrl.DeleteAgeGroup)

	cabins := rg.Group("/sessions/:sessionId/cabins")
	cabins.GET("", ctrl.ListCabins)
	cabins.GET("/:id", ctrl.GetCabin)
	cabins.POST("", ctrl.CreateCabin)
	cabins.PUT("/:id", ctrl.UpdateCabin)
	cabins.DELETE("/:id", ctrl.DeleteCabin)
}

// Session age group types

type CreateSessionAgeGroupRequest struct {
	AgeGroupID string `json:"age_group_id" binding:"required"`
	GroupSize  *int32 `json:"group_size"`
}

type UpdateSessionAgeGroupRequest struct {
	AgeGroupID string `json:"age_group_id" binding:"required"`
	GroupSize  *int32 `json:"group_size"`
}

type SessionAgeGroupResponse struct {
	ID         string `json:"id"`
	CampID     string `json:"camp_id"`
	SessionID  string `json:"session_id"`
	AgeGroupID string `json:"age_group_id"`
	GroupSize  *int32 `json:"group_size"`
}

// Session cabin types

type CreateSessionCabinRequest struct {
	SessionAgeGroupID  string `json:"session_age_group_id" binding:"required"`
	CabinID            string `json:"cabin_id" binding:"required"`
	GroupSize          *int32 `json:"group_size"`
	RequiredCounselors *int32 `json:"required_counselors"`
}

type UpdateSessionCabinRequest struct {
	CabinID            string `json:"cabin_id" binding:"required"`
	GroupSize          *int32 `json:"group_size"`
	RequiredCounselors *int32 `json:"required_counselors"`
}

type SessionCabinResponse struct {
	ID                 string `json:"id"`
	CampID             string `json:"camp_id"`
	SessionID          string `json:"session_id"`
	SessionAgeGroupID  string `json:"session_age_group_id"`
	CabinID            string `json:"cabin_id"`
	GroupSize          *int32 `json:"group_size"`
	RequiredCounselors *int32 `json:"required_counselors"`
}

// Session age group handlers

func (ctrl *Controller) ListAgeGroups(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")

	groups, err := ctrl.svc.ListAgeGroups(c.Request.Context(), campID, sessionID)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("session_id", sessionID).
			With("error", err).
			Error("error listing session age groups")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, groups)
}

func (ctrl *Controller) GetAgeGroup(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	id := c.Param("id")

	group, err := ctrl.svc.GetAgeGroup(c.Request.Context(), campID, sessionID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session age group not found"})
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
			Error("error getting session age group")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, group)
}

func (ctrl *Controller) CreateAgeGroup(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")

	var req CreateSessionAgeGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := ctrl.svc.CreateAgeGroup(c.Request.Context(), campID, sessionID, req)
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
			With("error", err).
			Error("error creating session age group")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, group)
}

func (ctrl *Controller) UpdateAgeGroup(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	id := c.Param("id")

	var req UpdateSessionAgeGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := ctrl.svc.UpdateAgeGroup(c.Request.Context(), campID, sessionID, id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session age group not found"})
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
		log.
			With("session_id", sessionID).
			With("id", id).
			With("error", err).
			Error("error updating session age group")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, group)
}

func (ctrl *Controller) DeleteAgeGroup(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	id := c.Param("id")

	err := ctrl.svc.DeleteAgeGroup(c.Request.Context(), campID, sessionID, id)
	if err != nil {
		if errors.Is(err, ErrAgeGroupNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session age group not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "session age group has dependent records and cannot be deleted"})
			return
		}
		log.
			With("session_id", sessionID).
			With("id", id).
			With("error", err).
			Error("error deleting session age group")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}

// Session cabin handlers

func (ctrl *Controller) ListCabins(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")

	cabins, err := ctrl.svc.ListCabins(c.Request.Context(), campID, sessionID)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("session_id", sessionID).
			With("error", err).
			Error("error listing session cabins")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, cabins)
}

func (ctrl *Controller) GetCabin(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	id := c.Param("id")

	cabin, err := ctrl.svc.GetCabin(c.Request.Context(), campID, sessionID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session cabin not found"})
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
			Error("error getting session cabin")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, cabin)
}

func (ctrl *Controller) CreateCabin(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")

	var req CreateSessionCabinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cabin, err := ctrl.svc.CreateCabin(c.Request.Context(), campID, sessionID, req)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session age group not found for this session"})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "referenced entity not found"})
			return
		}
		log.
			With("session_id", sessionID).
			With("error", err).
			Error("error creating session cabin")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, cabin)
}

func (ctrl *Controller) UpdateCabin(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	id := c.Param("id")

	var req UpdateSessionCabinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cabin, err := ctrl.svc.UpdateCabin(c.Request.Context(), campID, sessionID, id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session cabin not found"})
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
		log.
			With("session_id", sessionID).
			With("id", id).
			With("error", err).
			Error("error updating session cabin")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, cabin)
}

func (ctrl *Controller) DeleteCabin(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	id := c.Param("id")

	err := ctrl.svc.DeleteCabin(c.Request.Context(), campID, sessionID, id)
	if err != nil {
		if errors.Is(err, ErrCabinNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session cabin not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "session cabin has dependent records and cannot be deleted"})
			return
		}
		log.
			With("session_id", sessionID).
			With("id", id).
			With("error", err).
			Error("error deleting session cabin")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}
