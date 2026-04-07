package activity

import (
	"errors"
	"log/slog"
	"net/http"

	"camp-scheduler/internal/api"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type Controller struct {
	svc *Service
}

func NewController(svc *Service) *Controller {
	return &Controller{svc: svc}
}

func (ctrl *Controller) RegisterRoutes(camps *gin.RouterGroup) {
	activities := camps.Group("/:campId/activities")
	activities.GET("", ctrl.List)
	activities.GET("/:activityId", ctrl.Get)
	activities.POST("", ctrl.Create)
	activities.PUT("/:activityId", ctrl.Update)
	activities.DELETE("/:activityId", ctrl.Delete)

	certs := activities.Group("/:activityId/certifications")
	certs.GET("", ctrl.ListCertifications)
	certs.POST("", ctrl.AddCertification)
	certs.DELETE("/:certId", ctrl.RemoveCertification)
}

type CreateActivityRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateActivityRequest struct {
	Name string `json:"name" binding:"required"`
}

type ActivityResponse struct {
	ID     string `json:"id"`
	CampID string `json:"camp_id"`
	Name   string `json:"name"`
}

type AddCertificationRequest struct {
	CertificationID string `json:"certification_id" binding:"required"`
}

type ActivityCertificationResponse struct {
	ID                string `json:"id"`
	CampID            string `json:"camp_id"`
	ActivityID        string `json:"activity_id"`
	CertificationID   string `json:"certification_id"`
	CertificationName string `json:"certification_name,omitempty"`
}

func (ctrl *Controller) List(c *gin.Context) {
	campID := c.Param("campId")

	activities, err := ctrl.svc.List(c.Request.Context(), campID)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("error", err).
			Error("error listing activities")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, activities)
}

func (ctrl *Controller) Get(c *gin.Context) {
	campID := c.Param("campId")
	id := c.Param("activityId")

	activity, err := ctrl.svc.GetByID(c.Request.Context(), campID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "activity not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("id", id).
			With("error", err).
			Error("error getting activity")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, activity)
}

func (ctrl *Controller) Create(c *gin.Context) {
	campID := c.Param("campId")

	var req CreateActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	activity, err := ctrl.svc.Create(c.Request.Context(), campID, req)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("error", err).
			Error("error creating activity")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, activity)
}

func (ctrl *Controller) Update(c *gin.Context) {
	campID := c.Param("campId")
	id := c.Param("activityId")

	var req UpdateActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	activity, err := ctrl.svc.Update(c.Request.Context(), campID, id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "activity not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("id", id).
			With("error", err).
			Error("error updating activity")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, activity)
}

func (ctrl *Controller) Delete(c *gin.Context) {
	campID := c.Param("campId")
	id := c.Param("activityId")

	err := ctrl.svc.Delete(c.Request.Context(), campID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "activity not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "activity has dependent records and cannot be deleted"})
			return
		}
		slog.
			With("camp_id", campID).
			With("id", id).
			With("error", err).
			Error("error deleting activity")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}

func (ctrl *Controller) ListCertifications(c *gin.Context) {
	campID := c.Param("campId")
	activityID := c.Param("activityId")

	certs, err := ctrl.svc.ListCertifications(c.Request.Context(), campID, activityID)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("activity_id", activityID).
			With("error", err).
			Error("error listing activity certifications")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, certs)
}

func (ctrl *Controller) AddCertification(c *gin.Context) {
	campID := c.Param("campId")
	activityID := c.Param("activityId")

	var req AddCertificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cert, err := ctrl.svc.AddCertification(c.Request.Context(), campID, activityID, req)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "activity or certification not found"})
			return
		}
		if api.IsUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "certification already assigned to activity"})
			return
		}
		slog.
			With("camp_id", campID).
			With("activity_id", activityID).
			With("error", err).
			Error("error adding certification to activity")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, cert)
}

func (ctrl *Controller) RemoveCertification(c *gin.Context) {
	campID := c.Param("campId")
	activityID := c.Param("activityId")
	id := c.Param("certId")

	err := ctrl.svc.RemoveCertification(c.Request.Context(), campID, activityID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "activity certification not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("id", id).
			With("error", err).
			Error("error removing certification from activity")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}
