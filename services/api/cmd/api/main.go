package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/yiguan/api/internal/auth"
	"github.com/yiguan/api/internal/content"
	"github.com/yiguan/api/internal/identity"
	"github.com/yiguan/api/internal/moderation"
	"github.com/yiguan/api/internal/platform/cache"
	"github.com/yiguan/api/internal/platform/config"
	"github.com/yiguan/api/internal/platform/database"
	"github.com/yiguan/api/internal/platform/httpx"
)

type application struct {
	cfg     *config.Config
	db      *pgxpool.Pool
	cache   *redis.Client
	logger  *slog.Logger
	limiter httpx.RateLimiter
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var opts []config.LoaderOption
	if _, err := os.Stat(".env"); err == nil {
		opts = append(opts, config.WithEnvFile(".env"))
	}

	cfg, err := config.Load(opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(1)
	}

	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	pool, redisClient, err := connectInfrastructure(ctx, cfg)
	if err != nil {
		logger.Error("infrastructure connection failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer database.Close(pool)
	defer cache.Close(redisClient)

	app := &application{
		cfg:     cfg,
		db:      pool,
		cache:   redisClient,
		logger:  logger,
		limiter: &httpx.StubLimiter{},
	}

	authRepo := auth.NewPostgresRepository(pool)
	authMailer := auth.NewMailerFromConfig(cfg, logger)
	authLimiter := auth.NewMemoryLimiter()
	authService := auth.NewService(cfg, authRepo, authMailer, authLimiter)
	authHandler := auth.NewHandler(authService, cfg)

	identityRepo := identity.NewPostgresRepository(pool)
	identityLimiter := auth.NewMemoryLimiter()
	identityService := identity.NewService(cfg, identityRepo, authRepo, authMailer, identityLimiter, nil)
	identityHandler := identity.NewHandler(identityService, authHandler, cfg)
	authService.IdentityCleanup = identityService.CleanupOnAccountDeletion

	moderationRepo := moderation.NewPostgresRepository(pool)
	moderationLimiter := auth.NewMemoryLimiter()
	moderationService := moderation.NewService(cfg, moderationRepo, identityRepo, moderationLimiter)
	moderationHandler := moderation.NewHandler(moderationService, authHandler, identityHandler, identityService, cfg)

	contentRepo := content.NewPostgresRepository(pool)
	contentLimiter := auth.NewMemoryLimiter()
	contentService := content.NewService(cfg, contentRepo, identityRepo, moderationService, contentLimiter)
	contentHandler := content.NewHandler(contentService, identityHandler, identityService, cfg)
	// Re-wire identity service now that content service exists.
	identityService = identity.NewService(cfg, identityRepo, authRepo, authMailer, identityLimiter, contentService)
	identityHandler = identity.NewHandler(identityService, authHandler, cfg)

	router := newRouter(app, authHandler, identityHandler, contentHandler, moderationHandler)

	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("starting api server", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		logger.Error("server error", slog.String("error", err.Error()))
		os.Exit(1)
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("server stopped gracefully")
}

func newRouter(app *application, authHandler *auth.Handler, identityHandler *identity.Handler, contentHandler *content.Handler, moderationHandler *moderation.Handler) http.Handler {
	r := chi.NewRouter()

	r.Use(httpx.RequestID)
	r.Use(httpx.Recovery(app.logger))
	r.Use(httpx.Logger(app.logger, app.cfg.RateLimitBehindProxy))
	r.Use(httpx.CORS(httpx.CORSConfig{
		AllowedOrigins: app.cfg.CORSAllowedOrigins,
		Environment:    app.cfg.Environment,
	}))

	r.Get("/healthz", app.healthCheck)
	r.Get("/v1/healthz", app.healthCheck)
	r.Get("/readyz", app.readinessCheck)

	r.Mount("/v1", newV1Router(app, authHandler, identityHandler, contentHandler, moderationHandler))

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		httpx.Error(r.Context(), w, http.StatusNotFound, "NOT_FOUND", "resource not found")
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		httpx.Error(r.Context(), w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	})

	return r
}

func newV1Router(app *application, authHandler *auth.Handler, identityHandler *identity.Handler, contentHandler *content.Handler, moderationHandler *moderation.Handler) http.Handler {
	r := chi.NewRouter()

	r.Use(httpx.RateLimit(app.limiter, httpx.RateLimitConfig{
		BehindProxy: app.cfg.RateLimitBehindProxy,
	}))

	if authHandler != nil {
		authHandler.Mount(r)
	}
	if contentHandler != nil {
		contentHandler.Mount(r)
	}
	if contentHandler != nil {
		contentHandler.Mount(r)
	}
	if identityHandler != nil {
		identityHandler.Mount(r)
	}
	if moderationHandler != nil {
		moderationHandler.Mount(r)
	}

	return r
}

func (app *application) healthCheck(w http.ResponseWriter, r *http.Request) {
	if err := httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"}); err != nil {
		app.logger.Error("health check response failed", slog.String("error", err.Error()))
	}
}

func (app *application) readinessCheck(w http.ResponseWriter, r *http.Request) {
	type healthResult struct {
		Status string `json:"status"`
		DB     string `json:"db"`
		Cache  string `json:"cache"`
	}

	result := healthResult{Status: "ok", DB: "ok", Cache: "ok"}
	var mu sync.Mutex
	setUnavailable := func(component string) {
		mu.Lock()
		defer mu.Unlock()
		result.Status = "unavailable"
		switch component {
		case "db":
			result.DB = "unavailable"
		case "cache":
			result.Cache = "unavailable"
		}
	}

	checkCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if app.db == nil {
			setUnavailable("db")
			return
		}
		if err := app.db.Ping(checkCtx); err != nil {
			app.logger.Error("readiness check database ping failed", slog.String("error", err.Error()))
			setUnavailable("db")
		}
	}()

	go func() {
		defer wg.Done()
		if app.cache == nil {
			setUnavailable("cache")
			return
		}
		if err := app.cache.Ping(checkCtx).Err(); err != nil {
			app.logger.Error("readiness check cache ping failed", slog.String("error", err.Error()))
			setUnavailable("cache")
		}
	}()

	wg.Wait()

	status := http.StatusOK
	if result.Status != "ok" {
		status = http.StatusServiceUnavailable
	}

	if err := httpx.WriteJSON(w, status, result); err != nil {
		app.logger.Error("readiness check response failed", slog.String("error", err.Error()))
	}
}

func connectInfrastructure(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, *redis.Client, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	pool, err := database.NewPool(dbCtx, cfg.DatabaseURL, database.PoolConfig{
		MaxConns:          cfg.DatabaseMaxConns,
		MinConns:          cfg.DatabaseMinConns,
		MaxConnLifetime:   cfg.DatabaseMaxConnLifetime,
		HealthCheckPeriod: cfg.DatabaseHealthCheckPeriod,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("connect to postgres: %w", err)
	}

	redisCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	redisClient, err := cache.NewClient(redisCtx, cfg.RedisURL)
	if err != nil {
		database.Close(pool)
		return nil, nil, fmt.Errorf("connect to redis: %w", err)
	}

	return pool, redisClient, nil
}

func newLogger(level string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: logLevel}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}
