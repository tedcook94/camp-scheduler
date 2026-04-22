package enrollment

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
	enrollments := rg.Group("/sessions/:sessionId/enrollments")
	enrollments.GET("", ctrl.List)
	enrollments.GET("/:enrollmentId", ctrl.Get)
	enrollments.POST("", ctrl.Create)
	enrollments.DELETE("/:enrollmentId", ctrl.Delete)

	camperEnrollments := rg.Group("/campers/:camperId/enrollments")
	camperEnrollments.GET("", ctrl.ListByCamper)
}

type CreateEnrollmentRequest struct {
	CamperID          string `json:"camper_id" binding:"required"`
	SessionAgeGroupID string `json:"session_age_group_id" binding:"required"`
}

type EnrollmentResponse struct {
	ID                string `json:"id"`
	CampID            string `json:"camp_id"`
	CamperID          string `json:"camper_id"`
	SessionAgeGroupID string `json:"session_age_group_id"`
	CamperName        string `json:"camper_name"`
	SessionID         string `json:"session_id"`
	AgeGroupID        string `json:"age_group_id"`
}

func (ctrl *Controller) List(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")

	enrollments, err := ctrl.svc.ListBySession(c.Request.Context(), campID, sessionID)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("session_id", sessionID).
			With("error", err).
			Error("error listing enrollments")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, enrollments)
}

func (ctrl *Controller) ListByCamper(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	camperID := c.Param("camperId")

	enrollments, err := ctrl.svc.ListByCamper(c.Request.Context(), campID, camperID)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("camper_id", camperID).
			With("error", err).
			Error("error listing camper enrollments")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, enrollments)
}

func (ctrl *Controller) Get(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	id := c.Param("enrollmentId")

	enrollment, err := ctrl.svc.GetByID(c.Request.Context(), campID, sessionID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "enrollment not found"})
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
			Error("error getting enrollment")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, enrollment)
}

func (ctrl *Controller) Create(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")

	var req CreateEnrollmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	enrollment, err := ctrl.svc.Create(c.Request.Context(), campID, sessionID, req)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "referenced entity not found"})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "referenced entity not found"})
			return
		}
		if api.IsUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "camper is already enrolled in this session"})
			return
		}
		log.
			With("session_id", sessionID).
			With("error", err).
			Error("error creating enrollment")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, enrollment)
}

func (ctrl *Controller) Delete(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	id := c.Param("enrollmentId")

	err := ctrl.svc.Delete(c.Request.Context(), campID, sessionID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "enrollment not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "enrollment has dependent records and cannot be deleted"})
			return
		}
		log.
			With("session_id", sessionID).
			With("id", id).
			With("error", err).
			Error("error deleting enrollment")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}
