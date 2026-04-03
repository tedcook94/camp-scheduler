package camp

import (
	"errors"
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
	camps.GET("", h.List)
	camps.GET("/:id", h.Get)
	camps.POST("", h.Create)
	camps.PUT("/:id", h.Update)
	camps.DELETE("/:id", h.Delete)
}

type CreateCampRequest struct {
	Name     string  `json:"name" binding:"required"`
	Location *string `json:"location"`
}

type UpdateCampRequest struct {
	Name     string  `json:"name" binding:"required"`
	Location *string `json:"location"`
	Enabled  bool    `json:"enabled"`
}

type CampResponse struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Location *string `json:"location"`
	Enabled  bool    `json:"enabled"`
}

func (h *Handler) List(c *gin.Context) {
	camps, err := h.svc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, camps)
}

func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")

	camp, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "camp not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, camp)
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateCampRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	camp, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, camp)
}

func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")

	var req UpdateCampRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	camp, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "camp not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, camp)
}

func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")

	err := h.svc.Delete(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "camp not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}
