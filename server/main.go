package main

import (
	"camp-scheduler/server/config"
	"camp-scheduler/server/domains/camp"
	"camp-scheduler/server/repository"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

type Controller interface {
	Bind(*gin.RouterGroup)
}

type server struct {
	cfg         config.Config
	engine      *gin.Engine
	controllers []Controller
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		logrus.Fatal(err)
	}

	server, err := bootstrap(cfg)
	if err != nil {
		logrus.Fatal(err)
	}

	server.configureRoutes()

	logrus.Infof("Starting server on port %v in %v mode...", cfg.Server.Port, cfg.Server.Mode)

	logrus.Fatal(
		server.engine.Run(fmt.Sprintf(":%v", cfg.Server.Port)),
	)
}

func bootstrap(cfg config.Config) (server, error) {
	server := initServer(cfg)
	logger := initLogger(cfg)
	db, err := initDatabase(cfg)
	if err != nil {
		return server, err
	}
	if cfg.Database.DebugMode {
		db.AddQueryHook(bundebug.NewQueryHook())
	}

	campRepo := repository.New[camp.Camp](db)
	campService := camp.NewService(campRepo, logger)
	campController := camp.NewController(campService, logger)

	server.controllers = append(server.controllers, campController)

	return server, nil
}

func initServer(cfg config.Config) server {
	engine := gin.New()
	engine.Use(gin.Recovery())
	if cfg.Server.Mode != "local" {
		gin.SetMode(gin.ReleaseMode)
	}
	return server{
		cfg:         cfg,
		engine:      engine,
		controllers: []Controller{},
	}
}

func initLogger(cfg config.Config) *logrus.Logger {
	level, err := logrus.ParseLevel(cfg.Server.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger := logrus.StandardLogger()
	logger.SetLevel(level)
	if cfg.Server.Mode != "local" {
		logger.SetFormatter(new(logrus.JSONFormatter))
	}
	return logger
}

func initDatabase(cfg config.Config) (*bun.DB, error) {
	dbOpts := []pgdriver.Option{
		pgdriver.WithAddr(fmt.Sprintf("%v:%v", cfg.Database.Host, cfg.Database.Port)),
		pgdriver.WithUser(cfg.Database.User),
		pgdriver.WithDatabase(cfg.Database.Name),
		pgdriver.WithTimeout(cfg.Database.Timeout),
	}
	if cfg.Database.Password != "" {
		dbOpts = append(dbOpts, pgdriver.WithPassword(cfg.Database.Password))
	}
	switch cfg.Database.SSLMode {
	case "disable":
		dbOpts = append(dbOpts, pgdriver.WithTLSConfig(nil))
	default:
		return nil, errors.Errorf("unknown ssl mode %s", cfg.Database.SSLMode)
	}
	pgconn := pgdriver.NewConnector(dbOpts...)
	db := bun.NewDB(sql.OpenDB(pgconn), pgdialect.New())
	return db, nil
}

func (s *server) configureRoutes() {
	apiGroup := s.engine.Group("/api")
	for _, ctrl := range s.controllers {
		ctrl.Bind(apiGroup)
	}
	s.engine.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	s.engine.GET("/ready", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
}
