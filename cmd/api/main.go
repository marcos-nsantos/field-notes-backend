package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	_ "github.com/marcos-nsantos/field-notes-backend/docs"
	"github.com/marcos-nsantos/field-notes-backend/internal/adapter/handler"
	"github.com/marcos-nsantos/field-notes-backend/internal/adapter/repository/postgres"
	"github.com/marcos-nsantos/field-notes-backend/internal/infrastructure/auth"
	"github.com/marcos-nsantos/field-notes-backend/internal/infrastructure/cache"
	"github.com/marcos-nsantos/field-notes-backend/internal/infrastructure/config"
	"github.com/marcos-nsantos/field-notes-backend/internal/infrastructure/database"
	"github.com/marcos-nsantos/field-notes-backend/internal/infrastructure/middleware"
	"github.com/marcos-nsantos/field-notes-backend/internal/infrastructure/observability"
	"github.com/marcos-nsantos/field-notes-backend/internal/infrastructure/server"
	"github.com/marcos-nsantos/field-notes-backend/internal/infrastructure/storage"
	authUC "github.com/marcos-nsantos/field-notes-backend/internal/usecase/auth"
	"github.com/marcos-nsantos/field-notes-backend/internal/usecase/note"
	"github.com/marcos-nsantos/field-notes-backend/internal/usecase/sync"
	"github.com/marcos-nsantos/field-notes-backend/internal/usecase/upload"
)

//	@title			Field Notes API
//	@version		1.0
//	@description	Backend API for Field Notes mobile app - offline-first note taking with sync.

//	@contact.name	Marcos Santos
//	@contact.url	https://github.com/marcos-nsantos

//	@license.name	MIT

//	@host		localhost:8080
//	@BasePath	/api/v1

//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Enter "Bearer {token}" to authenticate

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger, err := observability.NewLogger(cfg.Log.Level, cfg.Log.Format)
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	ctx := context.Background()

	pool, err := database.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	// Run database migrations at startup to ensure schema is up-to-date
	if err := database.RunMigrations(ctx, pool, "migrations"); err != nil {
		logger.Fatal("failed to run migrations", zap.Error(err))
	}

	// Repositories
	userRepo := postgres.NewUserRepo(pool)
	noteRepo := postgres.NewNoteRepo(pool)
	photoRepo := postgres.NewPhotoRepo(pool)
	deviceRepo := postgres.NewDeviceRepo(pool)
	refreshTokenRepo := postgres.NewRefreshTokenRepo(pool)

	// Infrastructure services
	jwtSvc := auth.NewJWTService(cfg.JWT.SecretKey, cfg.JWT.AccessTokenTTL)
	passwordHasher := auth.NewPasswordHasher(12)

	s3Storage, err := storage.NewS3Storage(cfg.S3)
	if err != nil {
		logger.Fatal("failed to create s3 storage", zap.Error(err))
	}
	imageProcessor := storage.NewImageProcessor()

	// Rate limiter
	var rateLimiter *middleware.RateLimiter
	if cfg.RateLimit.Enabled {
		redisClient, err := cache.NewRedisClient(cfg.Redis)
		if err != nil {
			logger.Fatal("failed to connect to redis", zap.Error(err))
		}
		defer redisClient.Close()
		rateLimiter = middleware.NewRateLimiter(redisClient, cfg.RateLimit)
	}

	// Use cases
	authSvc := authUC.NewService(userRepo, deviceRepo, refreshTokenRepo, jwtSvc, passwordHasher, cfg.JWT.RefreshTokenTTL)
	noteSvc := note.NewService(noteRepo, photoRepo)
	syncSvc := sync.NewService(noteRepo, deviceRepo)
	uploadSvc := upload.NewService(photoRepo, noteRepo, s3Storage, imageProcessor)

	// Handlers
	authHandler := handler.NewAuthHandler(authSvc)
	noteHandler := handler.NewNoteHandler(noteSvc)
	syncHandler := handler.NewSyncHandler(syncSvc)
	uploadHandler := handler.NewUploadHandler(uploadSvc)

	// Middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtSvc)

	// Router
	router := server.NewRouter(server.RouterConfig{
		AuthHandler:     authHandler,
		NoteHandler:     noteHandler,
		SyncHandler:     syncHandler,
		UploadHandler:   uploadHandler,
		AuthMiddleware:  authMiddleware,
		RateLimiter:     rateLimiter,
		RateLimitEnable: cfg.RateLimit.Enabled,
		Logger:          logger,
		Environment:     cfg.Server.Environment,
	})

	// Server
	srv := server.NewServer(server.ServerConfig{
		Port:            cfg.Server.Port,
		ReadTimeout:     cfg.Server.ReadTimeout,
		WriteTimeout:    cfg.Server.WriteTimeout,
		ShutdownTimeout: cfg.Server.ShutdownTimeout,
		Handler:         router.Engine(),
		Logger:          logger,
	})

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server shutdown error", zap.Error(err))
	}

	logger.Info("server stopped")
}
