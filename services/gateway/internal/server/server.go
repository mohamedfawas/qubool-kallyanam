package server

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/handlers"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/middleware"
)

// Server represents the HTTP server
type Server struct {
	router *gin.Engine
	config *config.Config
}

// NewServer creates a new Server instance
func NewServer(cfg *config.Config) (*Server, error) {
	// Set Gin mode based on environment
	if cfg.Service.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	// Add CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Add logger middleware
	router.Use(middleware.Logger(cfg))

	// Create server
	s := &Server{
		router: router,
		config: cfg,
	}

	// Setup routes
	s.setupRoutes()

	return s, nil
}

// setupRoutes sets up the routes for the server
func (s *Server) setupRoutes() {
	// Health check endpoint
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": s.config.Service.Name,
			"version": s.config.Service.Version,
		})
	})

	// Setup API routes - v1
	v1 := s.router.Group("/api/v1")

	// Auth routes - no auth required
	auth := v1.Group("/auth")
	{
		authHandler := handlers.NewAuthHandler()
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/verify", authHandler.Verify)
	}

	// Routes requiring authentication
	protected := v1.Group("/")
	protected.Use(middleware.AuthRequired())

	// User routes
	user := protected.Group("/users")
	{
		userHandler := handlers.NewUserHandler()
		user.GET("/profile", userHandler.GetProfile)
		user.PUT("/profile", userHandler.UpdateProfile)
		user.GET("/search", userHandler.Search)
	}

	// Chat routes
	chat := protected.Group("/chats")
	{
		chatHandler := handlers.NewChatHandler()
		chat.GET("/", chatHandler.GetChats)
		chat.GET("/:id", chatHandler.GetChat)
		chat.POST("/:id/messages", chatHandler.SendMessage)
	}

	// Admin routes
	admin := v1.Group("/admin")
	admin.Use(middleware.AdminRequired())
	{
		adminHandler := handlers.NewAdminHandler()
		admin.GET("/users", adminHandler.ListUsers)
		admin.PUT("/users/:id", adminHandler.UpdateUser)
	}
}

// Run starts the HTTP server
func (s *Server) Run(addr string) error {
	srv := &http.Server{
		Addr:           addr,
		Handler:        s.router,
		ReadTimeout:    time.Duration(s.config.Server.Timeout) * time.Second,
		WriteTimeout:   time.Duration(s.config.Server.Timeout) * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	return srv.ListenAndServe()
}
