package agegroup

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(camps *gin.RouterGroup) {
	ageGroups := camps.Group("/:campId/age-groups")
	ageGroups.GET("", h.List)
	ageGroups.GET("/:id", h.Get)
	ageGroups.POST("", h.Create)
	ageGroups.PUT("/:id", h.Update)
	ageGroups.DELETE("/:id", h.Delete)
}

type CreateAgeGroupRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateAgeGroupRequest struct {
	Name string `json:"name" binding:"required"`
}

type AgeGroupResponse struct {
	ID     string `json:"id"`
	CampID string `json:"camp_id"`
	Name   string `json:"name"`
}

func (h *Handler) List(c *gin.Context) {
	campID := c.Param("campId")

	groups, err := h.svc.List(c.Request.Context(), campID)
	if err != nil {
		slog.
			With("camp_id", campID).
			With("error", err).
			Error("error listing age groups")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, groups)
}

func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")

	group, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "age group not found"})
			return
		}
		slog.
			With("id", id).
			With("error", err).
			Error("error getting age group")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, group)
}

func (h *Handler) Create(c *gin.Context) {
	campID := c.Param("campId")

	var req CreateAgeGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := h.svc.Create(c.Request.Context(), campID, req)
	if err != nil {
		slog.
			With("camp_id", campID).
			With("error", err).
			Error("error creating age group")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, group)
}

func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")

	var req UpdateAgeGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "age group not found"})
			return
		}
		slog.
			With("id", id).
			With("error", err).
			Error("error updating age group")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, group)
}

func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")

	err := h.svc.Delete(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "age group not found"})
			return
		}
		slog.
			With("id", id).
			With("error", err).
			Error("error deleting age group")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}
