package preferences

import (
	"errors"
	"log/slog"
	"net/http"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/auth"

	"github.com/gin-gonic/gin"
)

type CounselorCertificationController struct {
	svc *CounselorCertificationService
}

func NewCounselorCertificationController(svc *CounselorCertificationService) *CounselorCertificationController {
	return &CounselorCertificationController{svc: svc}
}

func (ctrl *CounselorCertificationController) RegisterRoutes(rg *gin.RouterGroup) {
	certs := rg.Group("/counselors/:counselorId/certifications")
	certs.GET("", ctrl.List)
	certs.POST("", ctrl.Add)
	certs.DELETE("/:id", ctrl.Remove)
}

type AddCounselorCertificationRequest struct {
	CertificationID string `json:"certification_id" binding:"required"`
}

type CounselorCertificationResponse struct {
	ID                string `json:"id"`
	CampID            string `json:"camp_id"`
	CounselorID       string `json:"counselor_id"`
	CertificationID   string `json:"certification_id"`
	CertificationName string `json:"certification_name,omitempty"`
}

func (ctrl *CounselorCertificationController) List(c *gin.Context) {
	campID := auth.GetCampID(c)
	counselorID := c.Param("counselorId")

	certs, err := ctrl.svc.List(c.Request.Context(), campID, counselorID)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("camp_id", campID).
			With("counselor_id", counselorID).
			With("error", err).
			Error("error listing counselor certifications")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, certs)
}

func (ctrl *CounselorCertificationController) Add(c *gin.Context) {
	campID := auth.GetCampID(c)
	counselorID := c.Param("counselorId")

	var req AddCounselorCertificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cert, err := ctrl.svc.Add(c.Request.Context(), campID, counselorID, req)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "counselor or certification not found"})
			return
		}
		if api.IsUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "counselor already has this certification"})
			return
		}
		slog.
			With("camp_id", campID).
			With("counselor_id", counselorID).
			With("error", err).
			Error("error adding counselor certification")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, cert)
}

func (ctrl *CounselorCertificationController) Remove(c *gin.Context) {
	campID := auth.GetCampID(c)
	counselorID := c.Param("counselorId")
	id := c.Param("id")

	err := ctrl.svc.Remove(c.Request.Context(), campID, counselorID, id)
	if err != nil {
		if errors.Is(err, ErrCounselorCertificationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "counselor certification not found"})
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
			Error("error removing counselor certification")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}
