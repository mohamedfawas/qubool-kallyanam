package server

import (
	"github.com/gin-gonic/gin"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/auth/jwt"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	authHandler "github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/handlers/v1/auth"
	chatHandler "github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/handlers/v1/chat"
	paymentHandler "github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/handlers/v1/payment"
	userHandler "github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/handlers/v1/user"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/middleware"
)

// SetupRouter configures all routes and middleware for the server.
func SetupRouter(
	r *gin.Engine,
	authH *authHandler.Handler,
	userH *userHandler.Handler,
	chatH *chatHandler.Handler,
	paymentH *paymentHandler.Handler,
	auth *middleware.Auth,
	logger logging.Logger,
) {
	// Global middleware
	r.Use(
		gin.Recovery(),                   // recover from panics
		middleware.SetupCORS(),           // CORS support for Razorpay
		middleware.SecurityHeaders(),     // Security headers
		middleware.RequestLogger(logger), // request logging
		middleware.ErrorHandler(),        // unified error handling
	)

	// API v1 base group
	v1 := r.Group("/api/v1")

	// Register auth and user routes
	registerAuthRoutes(v1.Group("/auth"), authH, auth)
	registerUserRoutes(v1.Group("/user"), userH, auth)
	registerChatRoutes(v1.Group("/chat"), chatH, auth)
	registerPaymentRoutes(v1.Group("/payment"), paymentH, auth)
	// Health check endpoint
	r.GET("/health", healthHandler)
}

// registerAuthRoutes sets up authentication-related endpoints.
func registerAuthRoutes(rg *gin.RouterGroup, h *authHandler.Handler, auth *middleware.Auth) {
	rg.POST("/register", h.Register)
	rg.POST("/verify", h.Verify)
	rg.POST("/login", h.Login)

	// Protected user-auth routes
	protected := rg.Group("/")
	protected.Use(auth.Authenticate())
	{
		protected.POST("/logout", h.Logout)
		protected.POST("/refresh", h.RefreshToken)
		protected.DELETE("/delete", h.DeleteAccount)
	}

	// Admin-specific routes
	admin := rg.Group("/admin")
	{
		admin.POST("/login", h.AdminLogin)
		adminProtected := admin.Group("/")
		adminProtected.Use(
			auth.Authenticate(),
			auth.RequireRole(jwt.RoleAdmin),
		)
		{
			adminProtected.POST("/logout", h.AdminLogout)
		}
	}
}

// registerUserRoutes sets up user-related endpoints, all protected.
func registerUserRoutes(rg *gin.RouterGroup, h *userHandler.Handler, auth *middleware.Auth) {
	// Regular users can access their profiles
	rg.Use(
		auth.Authenticate(),
		auth.ForwardToken(), // Forward token to downstream services
	)
	{
		rg.GET("/profile", h.GetProfile)
		rg.POST("/profile", h.UpdateProfile)
		rg.PATCH("/profile", h.PatchProfile)
		rg.POST("/profile/photo", h.UploadProfilePhoto)
		rg.DELETE("/profile/photo", h.DeleteProfilePhoto)
		rg.POST("/photos", h.UploadUserPhoto)
		rg.GET("/photos", h.GetUserPhotos)
		rg.DELETE("/photos/:order", h.DeleteUserPhoto)
		rg.POST("/video", h.UploadUserVideo)
		rg.GET("/video", h.GetUserVideo)
		rg.DELETE("/video", h.DeleteUserVideo)
		rg.POST("/partner-preferences", h.UpdatePartnerPreferences)
		rg.PATCH("/partner-preferences", h.PatchPartnerPreferences)
		rg.GET("/partner-preferences", h.GetPartnerPreferences)
		rg.GET("/matches/recommended", h.GetRecommendedMatches)
		rg.POST("/matches/action", h.RecordMatchAction)
		rg.PATCH("/matches/action", h.UpdateMatchAction)
		rg.GET("/matches/history", h.GetMatchHistory)
		rg.GET("/matches/mutual", h.GetMutualMatches)
	}
}

// registerChatRoutes sets up chat-related endpoints.
func registerChatRoutes(rg *gin.RouterGroup, h *chatHandler.Handler, auth *middleware.Auth) {
	// Protected routes (require authentication)
	protected := rg.Group("/")
	protected.Use(
		auth.Authenticate(),
		auth.RequireRole(jwt.RolePremiumUser), // Only premium users can create conversations
	)
	{
		protected.POST("/conversations", h.CreateConversation)
		protected.GET("/conversations", h.GetConversations)
		protected.POST("/conversations/:id/messages", h.SendMessage)
		protected.GET("/conversations/:id/messages", h.GetMessages)
		protected.GET("/conversations/:id/online", h.GetOnlineStatus)
		protected.GET("/ws", h.HandleWebSocket)
	}

	// Status endpoint (no auth required)
	rg.GET("/status", h.ChatStatus)
}

func registerPaymentRoutes(rg *gin.RouterGroup, h *paymentHandler.Handler, auth *middleware.Auth) {
	// API routes
	rg.GET("/status", h.PaymentStatus)

	protected := rg.Group("/")
	protected.Use(
		auth.Authenticate(),
		auth.ForwardToken(),
	)
	{
		protected.POST("/create-order", h.CreateOrder)
		protected.POST("/verify", h.VerifyPayment)
		protected.GET("/subscription", h.GetSubscription)
		protected.GET("/history", h.GetPaymentHistory)

		// HTML routes
		protected.GET("/plans", h.ShowPlans)
		protected.GET("/premium", h.ShowPaymentPage)
	}

	// Public HTML routes (no auth required)
	rg.GET("/success", h.ShowSuccessPage)
	rg.GET("/failed", h.ShowFailedPage)
}

// healthHandler handles the health check endpoint.
func healthHandler(c *gin.Context) {
	c.JSON(200, gin.H{"status": "UP"})
}
