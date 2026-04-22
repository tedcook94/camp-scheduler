package sessionconfig

import (
	"errors"
	"net/http"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type CounselorController struct {
	svc *CounselorService
}

func NewCounselorController(svc *CounselorService) *CounselorController {
	return &CounselorController{svc: svc}
}

func (ctrl *CounselorController) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/sessions/:sessionId/counselors")
	g.GET("", ctrl.List)
	g.POST("", ctrl.Add)
	g.DELETE("/:counselorId", ctrl.Remove)
}

func (ctrl *CounselorController) List(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")

	counselors, err := ctrl.svc.List(c.Request.Context(), campID, sessionID)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("session_id", sessionID).
			With("error", err).
			Error("error listing session counselors")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, counselors)
}

func (ctrl *CounselorController) Add(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")

	var req AddSessionCounselorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	counselor, err := ctrl.svc.Add(c.Request.Context(), campID, sessionID, req)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session or counselor not found"})
			return
		}
		if api.IsUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "counselor already in session"})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "referenced entity not found"})
			return
		}
		log.
			With("session_id", sessionID).
			With("counselor_id", req.CounselorID).
			With("error", err).
			Error("error adding session counselor")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, counselor)
}

func (ctrl *CounselorController) Remove(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	sessionID := c.Param("sessionId")
	counselorID := c.Param("counselorId")

	err := ctrl.svc.Remove(c.Request.Context(), campID, sessionID, counselorID)
	if err != nil {
		if errors.Is(err, ErrSessionCounselorNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session counselor not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("session_id", sessionID).
			With("counselor_id", counselorID).
			With("error", err).
			Error("error removing session counselor")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}
