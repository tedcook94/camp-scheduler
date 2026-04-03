package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"camp-scheduler/internal/agegroup"
	"camp-scheduler/internal/cabin"
	"camp-scheduler/internal/camp"
	"camp-scheduler/internal/config"
	"camp-scheduler/internal/db"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	cfg    config.Config
	pool   *pgxpool.Pool
	router *gin.Engine
}

func New(cfg config.Config) (*Server, error) {
	if cfg.Server.Mode != "local" {
		gin.SetMode(gin.ReleaseMode)
	}

	pool, err := initDB(cfg)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	s := &Server{
		cfg:    cfg,
		pool:   pool,
		router: gin.New(),
	}

	s.router.Use(gin.Recovery())
	if cfg.Server.Mode == "local" {
		s.router.Use(gin.Logger())
	}

	s.routes()
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) Shutdown() {
	s.pool.Close()
}

func (s *Server) Addr() string {
	return fmt.Sprintf(":%d", s.cfg.Server.Port)
}

func (s *Server) routes() {
	queries := db.New(s.pool)

	s.router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	s.router.GET("/ready", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	v1 := s.router.Group("/api/v1")

	camps := v1.Group("/camps")
	campService := camp.NewService(queries)
	campHandler := camp.NewHandler(campService)
	campHandler.RegisterRoutes(camps)

	ageGroupService := agegroup.NewService(queries)
	ageGroupHandler := agegroup.NewHandler(ageGroupService)
	ageGroupHandler.RegisterRoutes(camps)

	cabinService := cabin.NewService(queries)
	cabinHandler := cabin.NewHandler(cabinService)
	cabinHandler.RegisterRoutes(camps)
}

func initDB(cfg config.Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&connect_timeout=%d",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.SSLMode,
		int(cfg.Database.Timeout.Seconds()),
	)

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parsing database config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	slog.
		With("host", cfg.Database.Host).
		With("name", cfg.Database.Name).
		Info("connected to database")
	return pool, nil
}
