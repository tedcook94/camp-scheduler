package camp

import (
	"camp-scheduler/server/repository"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
)

type Controller struct {
	svc      *Service
	validate *validator.Validate
	logger   logrus.FieldLogger
}

func NewController(
	svc *Service,
	logger logrus.FieldLogger,
) *Controller {
	return &Controller{
		svc:      svc,
		validate: validator.New(),
		logger:   logger,
	}
}

func (c Controller) Bind(r *gin.RouterGroup) {
	v1 := r.Group("/v1")
	v1.POST("/camps", c.Create)
	v1.GET("/camps", c.Get)
	v1.GET("/camps/:id", c.GetByID)
	v1.PUT("/camps", c.Update)
	v1.DELETE("/camps/:id", c.Delete)
}

func (c Controller) Create(ginCtx *gin.Context) {
	ctx := ginCtx.Request.Context()
	var camp Camp
	if err := ginCtx.ShouldBindJSON(&camp); err != nil {
		c.logger.WithError(err).Warn("failed to parse camp from request body")
		ginCtx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := c.validate.Struct(camp); err != nil {
		c.logger.WithError(err).Warn("cannot create invalid camp")
		ginCtx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vm, err := c.svc.Create(ctx, camp)
	if err != nil {
		ginCtx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	ginCtx.JSON(http.StatusCreated, vm)
}

func (c Controller) Get(ginCtx *gin.Context) {
	ctx := ginCtx.Request.Context()
	camps, err := c.svc.Get(ctx)
	if err != nil {
		ginCtx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	ginCtx.JSON(http.StatusOK, camps)
}

func (c Controller) GetByID(ginCtx *gin.Context) {
	ctx := ginCtx.Request.Context()
	id := ginCtx.Param("id")
	camp, err := c.svc.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			ginCtx.AbortWithStatus(http.StatusNotFound)
			return
		}
		ginCtx.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	ginCtx.JSON(http.StatusOK, camp)
}

func (c Controller) Update(ginCtx *gin.Context) {
	ctx := ginCtx.Request.Context()
	var camp Camp
	if err := ginCtx.ShouldBindJSON(&camp); err != nil {
		c.logger.WithError(err).Warn("failed to parse camp from request body")
		ginCtx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log := c.logger.WithField("campID", camp.ID)
	if err := c.validate.Struct(camp); err != nil {
		log.WithError(err).Warn("camp update failed validation")
		ginCtx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vm, err := c.svc.Update(ctx, camp)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			ginCtx.AbortWithStatus(http.StatusNotFound)
			return
		}
		ginCtx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	ginCtx.JSON(http.StatusOK, vm)
}

func (ic Controller) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	if err := ic.svc.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusOK)
}
