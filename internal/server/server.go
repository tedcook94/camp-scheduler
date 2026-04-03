package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"camp-scheduler/internal/camp"
	"camp-scheduler/internal/config"
	"camp-scheduler/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	cfg  config.Config
	pool *pgxpool.Pool
	mux  *http.ServeMux
}

func New(cfg config.Config) (*Server, error) {
	pool, err := initDB(cfg)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	s := &Server{
		cfg:  cfg,
		pool: pool,
		mux:  http.NewServeMux(),
	}

	s.routes()
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) Shutdown() {
	s.pool.Close()
}

func (s *Server) Addr() string {
	return fmt.Sprintf(":%d", s.cfg.Server.Port)
}

func (s *Server) routes() {
	queries := db.New(s.pool)

	campService := camp.NewService(queries)
	campHandler := camp.NewHandler(campService)
	campHandler.RegisterRoutes(s.mux)

	s.mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s.mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
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

	slog.Info("connected to database", "host", cfg.Database.Host, "name", cfg.Database.Name)
	return pool, nil
}
