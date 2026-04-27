package admin

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/auth"
	"camp-scheduler/internal/camp"

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
	camps := rg.Group("/camps")
	camps.GET("", ctrl.List)
	camps.GET("/:id", ctrl.Get)
	camps.POST("", ctrl.Create)
	camps.PUT("/:id", ctrl.Update)
	camps.DELETE("/:id", ctrl.Delete)

	users := rg.Group("/users")
	users.PUT("/:user_id", ctrl.UpdateUser)
	users.POST("/:user_id/role", ctrl.SetUserRole)
	users.POST("/:user_id/camps/:camp_id", ctrl.AddMember)
	users.DELETE("/:user_id/camps/:camp_id", ctrl.RemoveMember)
}

type CreateCampRequest struct {
	Name     string  `json:"name" binding:"required"`
	Location *string `json:"location"`
}

type UpdateCampRequest = camp.UpdateCampRequest

func (ctrl *Controller) List(c *gin.Context) {
	log := auth.Logger(c)

	camps, err := ctrl.svc.List(c.Request.Context())
	if err != nil {
		log.With("error", err).Error("error listing camps")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, camps)
}

func (ctrl *Controller) Get(c *gin.Context) {
	log := auth.Logger(c)

	id := c.Param("id")

	camp, err := ctrl.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "camp not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.
			With("id", id).
			With("error", err).
			Error("error getting camp")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, camp)
}

func (ctrl *Controller) Create(c *gin.Context) {
	log := auth.Logger(c)

	var req CreateCampRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	camp, err := ctrl.svc.Create(c.Request.Context(), req)
	if err != nil {
		if api.IsUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "a camp with that name already exists"})
			return
		}
		log.With("error", err).Error("error creating camp")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, camp)
}

func (ctrl *Controller) Update(c *gin.Context) {
	log := auth.Logger(c)

	id := c.Param("id")

	var req UpdateCampRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	camp, err := ctrl.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "camp not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if api.IsUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "a camp with that name already exists"})
			return
		}
		log.
			With("id", id).
			With("error", err).
			Error("error updating camp")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, camp)
}

func (ctrl *Controller) Delete(c *gin.Context) {
	log := auth.Logger(c)

	id := c.Param("id")

	err := ctrl.svc.Delete(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "camp not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if api.IsFKViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "camp has dependent records and cannot be deleted"})
			return
		}
		log.
			With("id", id).
			With("error", err).
			Error("error deleting camp")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}

type AddMemberRequest struct {
	Role string `json:"role"`
}

// mapAuthServerError forwards an upstream auth-server failure to the client
// with a sensible HTTP status. Auth-server 4xx responses are passed through
// as-is so client-facing validation errors (duplicate email, user not found,
// etc.) reach the UI with the correct status; 5xx responses are also passed
// through verbatim so the failure mode of the upstream service is visible to
// the caller rather than masked behind a generic 500. Returns true when the
// error was handled.
func mapAuthServerError(c *gin.Context, log *slog.Logger, err error) bool {
	var ase *AuthServerError
	if !errors.As(err, &ase) {
		return false
	}
	if ase.Status >= 500 {
		log.With("status", ase.Status).With("error", err).Error("auth-server upstream error")
	}
	body := gin.H{"error": ase.Message}
	if ase.Message == "" {
		body["error"] = http.StatusText(ase.Status)
	}
	if ase.Code != "" {
		body["code"] = ase.Code
	}
	c.JSON(ase.Status, body)
	return true
}

func (ctrl *Controller) AddMember(c *gin.Context) {
	log := auth.Logger(c)

	userID := c.Param("user_id")
	campID := c.Param("camp_id")

	var req AddMemberRequest
	// Body is optional. An empty body uses the default role; any other parse
	// failure is a client error and must not silently fall through to the
	// default-role path.
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	role := req.Role
	if role == "" {
		role = "admin"
	}

	if err := ctrl.svc.AddMember(c.Request.Context(), userID, campID, role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "camp not found"})
			return
		}
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if mapAuthServerError(c, log, err) {
			return
		}
		log.
			With("user_id", userID).
			With("camp_id", campID).
			With("error", err).
			Error("error adding camp member")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}

func (ctrl *Controller) RemoveMember(c *gin.Context) {
	log := auth.Logger(c)

	userID := c.Param("user_id")
	campID := c.Param("camp_id")

	if err := ctrl.svc.RemoveMember(c.Request.Context(), userID, campID); err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if mapAuthServerError(c, log, err) {
			return
		}
		log.
			With("user_id", userID).
			With("camp_id", campID).
			With("error", err).
			Error("error removing camp member")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}

type UpdateUserRequestBody struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Email     *string `json:"email"`
}

func (ctrl *Controller) UpdateUser(c *gin.Context) {
	log := auth.Logger(c)

	userID := c.Param("user_id")
	var req UpdateUserRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := ctrl.svc.UpdateUser(c.Request.Context(), userID, UpdateUserRequest{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
	})
	if err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if mapAuthServerError(c, log, err) {
			return
		}
		log.
			With("user_id", userID).
			With("error", err).
			Error("error updating user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}

type SetRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

func (ctrl *Controller) SetUserRole(c *gin.Context) {
	log := auth.Logger(c)

	userID := c.Param("user_id")
	var req SetRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.svc.SetUserRole(c.Request.Context(), userID, req.Role); err != nil {
		if api.IsBadInput(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if mapAuthServerError(c, log, err) {
			return
		}
		log.
			With("user_id", userID).
			With("error", err).
			Error("error setting user role")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusOK)
}
