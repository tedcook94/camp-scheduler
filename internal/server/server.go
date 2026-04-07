package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"camp-scheduler/internal/activity"
	"camp-scheduler/internal/agegroup"
	"camp-scheduler/internal/assignment"
	"camp-scheduler/internal/cabin"
	"camp-scheduler/internal/camp"
	"camp-scheduler/internal/camper"
	"camp-scheduler/internal/certification"
	"camp-scheduler/internal/config"
	"camp-scheduler/internal/counselor"
	"camp-scheduler/internal/db"
	"camp-scheduler/internal/enrollment"
	"camp-scheduler/internal/history"
	"camp-scheduler/internal/preferences"
	"camp-scheduler/internal/season"
	"camp-scheduler/internal/session"
	"camp-scheduler/internal/sessionconfig"
	"camp-scheduler/internal/timeslot"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	cfg    config.Config
	pool   *pgxpool.Pool
	router *gin.Engine
}

func New(cfg config.Config) (*Server, error) {
	pool, err := initDB(cfg)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}
	return NewWithPool(cfg, pool), nil
}

// NewWithPool creates a server with an existing database connection pool,
// allowing callers (such as integration tests) to supply their own pool.
func NewWithPool(cfg config.Config, pool *pgxpool.Pool) *Server {
	if cfg.Server.Mode == "local" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
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
	return s
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
		if err := s.pool.Ping(c.Request.Context()); err != nil {
			c.Status(http.StatusServiceUnavailable)
			return
		}
		c.Status(http.StatusOK)
	})

	v1 := s.router.Group("/api/v1")

	camps := v1.Group("/camps")
	campService := camp.NewService(queries)
	campController := camp.NewController(campService)
	campController.RegisterRoutes(camps)

	ageGroupService := agegroup.NewService(queries)
	ageGroupController := agegroup.NewController(ageGroupService)
	ageGroupController.RegisterRoutes(camps)

	cabinService := cabin.NewService(queries)
	cabinController := cabin.NewController(cabinService)
	cabinController.RegisterRoutes(camps)

	seasonService := season.NewService(queries)
	seasonController := season.NewController(seasonService)
	seasonController.RegisterRoutes(camps)

	sessionService := session.NewService(queries)
	sessionController := session.NewController(sessionService)
	sessionController.RegisterRoutes(camps)

	counselorService := counselor.NewService(queries)
	counselorController := counselor.NewController(counselorService)
	counselorController.RegisterRoutes(camps)

	ageGroupPrefService := preferences.NewAgeGroupService(queries)
	ageGroupPrefController := preferences.NewAgeGroupController(ageGroupPrefService)
	ageGroupPrefController.RegisterRoutes(camps)

	cocounselorPrefService := preferences.NewCocounselorService(queries)
	cocounselorPrefController := preferences.NewCocounselorController(cocounselorPrefService)
	cocounselorPrefController.RegisterRoutes(camps)

	counselorCertService := preferences.NewCounselorCertificationService(queries)
	counselorCertController := preferences.NewCounselorCertificationController(counselorCertService)
	counselorCertController.RegisterRoutes(camps)

	activityPrefService := preferences.NewActivityPreferenceService(queries)
	activityPrefController := preferences.NewActivityPreferenceController(activityPrefService)
	activityPrefController.RegisterRoutes(camps)

	camperFriendPrefService := preferences.NewCamperFriendService(queries)
	camperFriendPrefController := preferences.NewCamperFriendController(camperFriendPrefService)
	camperFriendPrefController.RegisterRoutes(camps)

	historyService := history.NewService(queries)
	historyController := history.NewController(historyService)
	historyController.RegisterRoutes(camps)

	sessionConfigService := sessionconfig.NewService(queries)
	sessionConfigController := sessionconfig.NewController(sessionConfigService)
	sessionConfigController.RegisterRoutes(camps)

	activityConfigService := sessionconfig.NewActivityService(queries)
	activityConfigController := sessionconfig.NewActivityController(activityConfigService)
	activityConfigController.RegisterRoutes(camps)

	camperService := camper.NewService(queries)
	camperController := camper.NewController(camperService)
	camperController.RegisterRoutes(camps)

	enrollmentService := enrollment.NewService(queries)
	enrollmentController := enrollment.NewController(enrollmentService)
	enrollmentController.RegisterRoutes(camps)

	certificationService := certification.NewService(queries)
	certificationController := certification.NewController(certificationService)
	certificationController.RegisterRoutes(camps)

	activityService := activity.NewService(queries)
	activityController := activity.NewController(activityService)
	activityController.RegisterRoutes(camps)

	timeSlotService := timeslot.NewService(queries)
	timeSlotController := timeslot.NewController(timeSlotService)
	timeSlotController.RegisterRoutes(camps)

	assignmentService := assignment.NewService(queries, s.pool)
	assignmentController := assignment.NewController(assignmentService)
	assignmentController.RegisterRoutes(camps)
}

func initDB(cfg config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.Database.DSN())
	if err != nil {
		return nil, fmt.Errorf("error parsing database config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, fmt.Errorf("error creating connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("error pinging database: %w", err)
	}

	slog.
		With("host", cfg.Database.Host).
		With("name", cfg.Database.Name).
		Info("connected to database")
	return pool, nil
}
