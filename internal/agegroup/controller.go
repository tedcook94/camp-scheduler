package agegroup

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
	ageGroups := rg.Group("/age-groups")
	ageGroups.GET("", ctrl.List)
	ageGroups.GET("/archived", ctrl.ListArchived)
	ageGroups.GET("/:id", ctrl.Get)
	ageGroups.POST("", ctrl.Create)
	ageGroups.PUT("/:id", ctrl.Update)
	ageGroups.DELETE("/:id", ctrl.Delete)
	ageGroups.POST("/:id/archive", ctrl.Archive)
	ageGroups.POST("/:id/unarchive", ctrl.Unarchive)
}

type CreateAgeGroupRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateAgeGroupRequest struct {
	Name string `json:"name" binding:"required"`
}

type AgeGroupResponse struct {
	ID       string `json:"id"`
	CampID   string `json:"camp_id"`
	Name     string `json:"name"`
	Archived bool   `json:"archived"`
}

func (ctrl *Controller) List(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)

	groups, err := ctrl.svc.List(c.Request.Context(), campID)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("error", err).
			Error("error listing age groups")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, groups)
}

func (ctrl *Controller) Get(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	id := c.Param("id")

	group, err := ctrl.svc.GetByID(c.Request.Context(), campID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "age group not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("id", id).
			With("error", err).
			Error("error getting age group")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, group)
}

func (ctrl *Controller) Create(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)

	var req CreateAgeGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := ctrl.svc.Create(c.Request.Context(), campID, req)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("error", err).
			Error("error creating age group")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, group)
}

func (ctrl *Controller) Update(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	id := c.Param("id")

	var req UpdateAgeGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := ctrl.svc.Update(c.Request.Context(), campID, id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "age group not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("id", id).
			With("error", err).
			Error("error updating age group")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, group)
}

func (ctrl *Controller) Delete(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	id := c.Param("id")

	err := ctrl.svc.Delete(c.Request.Context(), campID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "age group not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "age group has dependent records and cannot be deleted"})
			return
		}
		log.
			With("id", id).
			With("error", err).
			Error("error deleting age group")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}

func (ctrl *Controller) ListArchived(c *gin.Context) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)

	groups, err := ctrl.svc.ListArchived(c.Request.Context(), campID)
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("error", err).
			Error("error listing archived age groups")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, groups)
}

func (ctrl *Controller) Archive(c *gin.Context) {
	ctrl.setArchived(c, true)
}

func (ctrl *Controller) Unarchive(c *gin.Context) {
	ctrl.setArchived(c, false)
}

func (ctrl *Controller) setArchived(c *gin.Context, archived bool) {
	log := auth.Logger(c)
	campID := auth.GetCampID(c)
	id := c.Param("id")

	var err error
	if archived {
		err = ctrl.svc.Archive(c.Request.Context(), campID, id)
	} else {
		err = ctrl.svc.Unarchive(c.Request.Context(), campID, id)
	}
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "age group not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("id", id).
			With("archived", archived).
			With("error", err).
			Error("error setting age group archived")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}
