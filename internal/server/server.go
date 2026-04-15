package server

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"

	"camp-scheduler/internal/activity"
	"camp-scheduler/internal/admin"
	"camp-scheduler/internal/agegroup"
	"camp-scheduler/internal/assignment"
	"camp-scheduler/internal/auth"
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
	cfg      config.Config
	pool     *pgxpool.Pool
	router   *gin.Engine
	staticFS fs.FS
}

func New(cfg config.Config, staticFS fs.FS) (*Server, error) {
	pool, err := initDB(cfg)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}
	return NewWithPool(cfg, pool, staticFS), nil
}

// NewWithPool creates a server with an existing database connection pool,
// allowing callers (such as integration tests) to supply their own pool.
func NewWithPool(cfg config.Config, pool *pgxpool.Pool, staticFS fs.FS) *Server {
	if cfg.Server.Mode == "local" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	s := &Server{
		cfg:      cfg,
		pool:     pool,
		router:   gin.New(),
		staticFS: staticFS,
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

	authenticator := auth.NewJWTAuthenticator(s.pool, queries, auth.JWTConfig{
		SigningKey:      []byte(s.cfg.JWT.Secret),
		AccessTokenTTL:  s.cfg.JWT.AccessTokenTTL,
		RefreshTokenTTL: s.cfg.JWT.RefreshTokenTTL,
	})

	authController := auth.NewController(authenticator)
	authController.RegisterRoutes(v1)

	protected := v1.Group("")
	protected.Use(auth.Middleware(authenticator), auth.RequireCampScope())
	campService := camp.NewService(queries)
	campController := camp.NewController(campService)
	campController.RegisterRoutes(protected)

	ageGroupService := agegroup.NewService(queries)
	ageGroupController := agegroup.NewController(ageGroupService)
	ageGroupController.RegisterRoutes(protected)

	cabinService := cabin.NewService(queries)
	cabinController := cabin.NewController(cabinService)
	cabinController.RegisterRoutes(protected)

	seasonService := season.NewService(queries)
	seasonController := season.NewController(seasonService)
	seasonController.RegisterRoutes(protected)

	sessionService := session.NewService(queries)
	sessionController := session.NewController(sessionService)
	sessionController.RegisterRoutes(protected)

	counselorService := counselor.NewService(queries)
	counselorController := counselor.NewController(counselorService)
	counselorController.RegisterRoutes(protected)

	ageGroupPrefService := preferences.NewAgeGroupService(queries)
	ageGroupPrefController := preferences.NewAgeGroupController(ageGroupPrefService)
	ageGroupPrefController.RegisterRoutes(protected)

	cocounselorPrefService := preferences.NewCocounselorService(queries)
	cocounselorPrefController := preferences.NewCocounselorController(cocounselorPrefService)
	cocounselorPrefController.RegisterRoutes(protected)

	counselorCertService := preferences.NewCounselorCertificationService(queries)
	counselorCertController := preferences.NewCounselorCertificationController(counselorCertService)
	counselorCertController.RegisterRoutes(protected)

	activityPrefService := preferences.NewActivityPreferenceService(queries)
	activityPrefController := preferences.NewActivityPreferenceController(activityPrefService)
	activityPrefController.RegisterRoutes(protected)

	camperFriendPrefService := preferences.NewCamperFriendService(queries)
	camperFriendPrefController := preferences.NewCamperFriendController(camperFriendPrefService)
	camperFriendPrefController.RegisterRoutes(protected)

	historyService := history.NewService(queries)
	historyController := history.NewController(historyService)
	historyController.RegisterRoutes(protected)

	sessionConfigService := sessionconfig.NewService(queries)
	sessionConfigController := sessionconfig.NewController(sessionConfigService)
	sessionConfigController.RegisterRoutes(protected)

	activityConfigService := sessionconfig.NewActivityService(queries)
	activityConfigController := sessionconfig.NewActivityController(activityConfigService)
	activityConfigController.RegisterRoutes(protected)

	camperService := camper.NewService(queries)
	camperController := camper.NewController(camperService)
	camperController.RegisterRoutes(protected)

	enrollmentService := enrollment.NewService(queries)
	enrollmentController := enrollment.NewController(enrollmentService)
	enrollmentController.RegisterRoutes(protected)

	certificationService := certification.NewService(queries)
	certificationController := certification.NewController(certificationService)
	certificationController.RegisterRoutes(protected)

	activityService := activity.NewService(queries)
	activityController := activity.NewController(activityService)
	activityController.RegisterRoutes(protected)

	timeSlotService := timeslot.NewService(queries)
	timeSlotController := timeslot.NewController(timeSlotService)
	timeSlotController.RegisterRoutes(protected)

	assignmentService := assignment.NewService(queries, s.pool)
	assignmentController := assignment.NewController(assignmentService)
	assignmentController.RegisterRoutes(protected)

	superAdmin := v1.Group("/admin")
	superAdmin.Use(auth.Middleware(authenticator))
	superAdmin.Use(auth.RequireSuperAdmin())

	adminService := admin.NewService(queries)
	adminController := admin.NewController(adminService)
	adminController.RegisterRoutes(superAdmin)

	userService := admin.NewUserService(queries)
	userController := admin.NewUserController(userService)
	userController.RegisterRoutes(superAdmin)

	if s.staticFS != nil {
		s.serveSPA()
	}
}

// serveSPA configures the router to serve the embedded SPA under /admin.
// Requests for static assets are served directly; all other /admin paths
// fall back to index.html so that SvelteKit handles client-side routing.
func (s *Server) serveSPA() {
	indexHTML, err := fs.ReadFile(s.staticFS, "index.html")
	if err != nil {
		slog.With("error", err).Error("error reading index.html from static FS")
		return
	}

	fileServer := http.FileServer(http.FS(s.staticFS))

	s.router.NoRoute(func(c *gin.Context) {
		reqPath := c.Request.URL.Path

		if reqPath != "/admin" && !strings.HasPrefix(reqPath, "/admin/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		// Strip the /admin prefix to get the path within the static FS
		fsPath := strings.TrimPrefix(reqPath, "/admin")
		if fsPath == "" {
			fsPath = "/"
		}

		// Try to stat the file to check if it exists and is not a directory.
		// Directories must not be served directly to avoid exposing a listing
		// of the embedded files.
		filePath := strings.TrimPrefix(fsPath, "/")
		if filePath == "" {
			filePath = "."
		}
		if info, err := fs.Stat(s.staticFS, filePath); err == nil && !info.IsDir() {
			c.Request.URL.Path = fsPath
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		// If the path has a file extension it's a missing asset (JS, CSS, etc.)
		// — return 404 instead of falling back to index.html, which would cause
		// MIME-type errors in the browser.
		if path.Ext(fsPath) != "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		// Route-like path — serve index.html directly for client-side routing.
		// We serve the cached content instead of delegating to FileServer
		// because FileServer redirects /index.html requests to /.
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})
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
