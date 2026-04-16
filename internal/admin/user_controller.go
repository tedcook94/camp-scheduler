package admin

import (
	"errors"
	"log/slog"
	"net/http"

	"camp-scheduler/internal/api"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type UserController struct {
	svc *UserService
}

func NewUserController(svc *UserService) *UserController {
	return &UserController{svc: svc}
}

func (ctrl *UserController) RegisterRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	users.GET("", ctrl.List)
	users.GET("/:id", ctrl.Get)
	users.POST("", ctrl.Create)
	users.PUT("/:id", ctrl.Update)
	users.PUT("/:id/password", ctrl.UpdatePassword)
	users.DELETE("/:id", ctrl.Delete)
}

func (ctrl *UserController) List(c *gin.Context) {
	campID := c.Query("camp_id")

	if campID != "" {
		users, err := ctrl.svc.ListByCamp(c.Request.Context(), campID)
		if err != nil {
			if api.IsBadInput(err) {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			slog.
				With("camp_id", campID).
				With("error", err).
				Error("error listing users by camp")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		c.JSON(http.StatusOK, users)
		return
	}

	users, err := ctrl.svc.List(c.Request.Context())
	if err != nil {
		slog.With("error", err).Error("error listing users")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, users)
}

func (ctrl *UserController) Get(c *gin.Context) {
	id := c.Param("id")

	user, err := ctrl.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("id", id).
			With("error", err).
			Error("error getting user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (ctrl *UserController) Create(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := ctrl.svc.Create(c.Request.Context(), req)
	if err != nil {
		if api.IsUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "a user with that username or email already exists"})
			return
		}
		if api.IsCheckViolation(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "super_admin role requires no camp_id; admin role requires a camp_id"})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "camp not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.With("error", err).Error("error creating user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (ctrl *UserController) Update(c *gin.Context) {
	id := c.Param("id")

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := ctrl.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if api.IsUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "a user with that username or email already exists"})
			return
		}
		if api.IsCheckViolation(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "super_admin role requires no camp_id; admin role requires a camp_id"})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "camp not found"})
			return
		}
		slog.
			With("id", id).
			With("error", err).
			Error("error updating user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (ctrl *UserController) UpdatePassword(c *gin.Context) {
	id := c.Param("id")

	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	revoked, err := ctrl.svc.UpdatePassword(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("id", id).
			With("error", err).
			Error("error updating user password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	slog.
		With("id", id).
		With("revoked_tokens", revoked).
		Info("password updated and refresh tokens revoked")

	c.Status(http.StatusOK)
}

func (ctrl *UserController) Delete(c *gin.Context) {
	id := c.Param("id")

	err := ctrl.svc.Delete(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.
			With("id", id).
			With("error", err).
			Error("error deleting user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}
