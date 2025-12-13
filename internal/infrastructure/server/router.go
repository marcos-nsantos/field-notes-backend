package server

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	"github.com/marcos-nsantos/field-notes-backend/internal/adapter/handler"
	"github.com/marcos-nsantos/field-notes-backend/internal/infrastructure/middleware"
)

type Router struct {
	engine         *gin.Engine
	authHandler    *handler.AuthHandler
	noteHandler    *handler.NoteHandler
	syncHandler    *handler.SyncHandler
	uploadHandler  *handler.UploadHandler
	authMiddleware *middleware.AuthMiddleware
	logger         *zap.Logger
}

type RouterConfig struct {
	AuthHandler    *handler.AuthHandler
	NoteHandler    *handler.NoteHandler
	SyncHandler    *handler.SyncHandler
	UploadHandler  *handler.UploadHandler
	AuthMiddleware *middleware.AuthMiddleware
	Logger         *zap.Logger
	Environment    string
}

func NewRouter(cfg RouterConfig) *Router {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	r := &Router{
		engine:         engine,
		authHandler:    cfg.AuthHandler,
		noteHandler:    cfg.NoteHandler,
		syncHandler:    cfg.SyncHandler,
		uploadHandler:  cfg.UploadHandler,
		authMiddleware: cfg.AuthMiddleware,
		logger:         cfg.Logger,
	}

	r.setupMiddleware()
	r.setupRoutes()

	return r
}

func (r *Router) setupMiddleware() {
	r.engine.Use(middleware.Recovery(r.logger))
	r.engine.Use(middleware.RequestID())
	r.engine.Use(middleware.Logger(r.logger))
	r.engine.Use(middleware.CORS())
}

func (r *Router) setupRoutes() {
	r.engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Swagger documentation
	r.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.engine.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", r.authHandler.Register)
			auth.POST("/login", r.authHandler.Login)
			auth.POST("/refresh", r.authHandler.Refresh)
			auth.POST("/logout", r.authMiddleware.RequireAuth(), r.authHandler.Logout)
		}

		notes := api.Group("/notes")
		notes.Use(r.authMiddleware.RequireAuth())
		{
			notes.POST("", r.noteHandler.Create)
			notes.GET("", r.noteHandler.List)
			notes.GET("/:id", r.noteHandler.Get)
			notes.PUT("/:id", r.noteHandler.Update)
			notes.DELETE("/:id", r.noteHandler.Delete)
		}

		sync := api.Group("/sync")
		sync.Use(r.authMiddleware.RequireAuth())
		{
			sync.POST("", r.syncHandler.Sync)
		}

		upload := api.Group("/upload")
		upload.Use(r.authMiddleware.RequireAuth())
		{
			upload.POST("/:note_id", r.uploadHandler.Upload)
		}

		photos := api.Group("/photos")
		photos.Use(r.authMiddleware.RequireAuth())
		{
			photos.DELETE("/:id", r.uploadHandler.Delete)
		}
	}
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}
