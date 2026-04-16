package admin

import (
	"errors"
	"net/http"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type UserController struct {
	svc           *UserService
	authenticator auth.Authenticator
}

func NewUserController(svc *UserService, authenticator auth.Authenticator) *UserController {
	return &UserController{svc: svc, authenticator: authenticator}
}

func (ctrl *UserController) RegisterRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	users.GET("", ctrl.List)
	users.GET("/:id", ctrl.Get)
	users.POST("", ctrl.Create)
	users.PUT("/:id", ctrl.Update)
	users.PUT("/:id/password", ctrl.UpdatePassword)
	users.DELETE("/:id", ctrl.Delete)
	users.POST("/:id/impersonate", ctrl.Impersonate)
}

func (ctrl *UserController) List(c *gin.Context) {
	log := auth.Logger(c)

	campID := c.Query("camp_id")

	if campID != "" {
		users, err := ctrl.svc.ListByCamp(c.Request.Context(), campID)
		if err != nil {
			if api.IsBadInput(err) {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			log.
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
		log.With("error", err).Error("error listing users")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, users)
}

func (ctrl *UserController) Get(c *gin.Context) {
	log := auth.Logger(c)

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
		log.
			With("id", id).
			With("error", err).
			Error("error getting user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (ctrl *UserController) Create(c *gin.Context) {
	log := auth.Logger(c)

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
		log.With("error", err).Error("error creating user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (ctrl *UserController) Update(c *gin.Context) {
	log := auth.Logger(c)

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
		log.
			With("id", id).
			With("error", err).
			Error("error updating user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (ctrl *UserController) UpdatePassword(c *gin.Context) {
	log := auth.Logger(c)

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
		log.
			With("id", id).
			With("error", err).
			Error("error updating user password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	log.
		With("id", id).
		With("revoked_tokens", revoked).
		Info("password updated and refresh tokens revoked")

	c.Status(http.StatusOK)
}

func (ctrl *UserController) Delete(c *gin.Context) {
	log := auth.Logger(c)

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
		log.
			With("id", id).
			With("error", err).
			Error("error deleting user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}

func (ctrl *UserController) Impersonate(c *gin.Context) {
	log := auth.Logger(c)

	targetID := c.Param("id")
	claims := auth.GetClaims(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authentication claims"})
		return
	}

	accessToken, refreshToken, err := ctrl.authenticator.ImpersonateUser(c.Request.Context(), targetID, claims.UserID)
	if err != nil {
		if errors.Is(err, auth.ErrCannotImpersonate) {
			c.JSON(http.StatusForbidden, gin.H{"error": "cannot impersonate a super-admin user"})
			return
		}
		if errors.Is(err, auth.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("target_user_id", targetID).
			With("error", err).
			Error("error impersonating user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	log.
		With("target_user_id", targetID).
		Info("impersonation session started")

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}
