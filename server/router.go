package server

import (
	"fmt"
	"os"
	"time"
	
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/techagentng/iweapp/websocket"
)

func (s *Server) setupRouter() *gin.Engine {
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "test" {
		r := gin.New()
		s.defineRoutes(r)
		return r
	}

	r := gin.New()

	// Logger middleware with custom format
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))
	r.Use(gin.Recovery())

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"https://api.iweapps.com",
			"https://iweapps.com",
			"https://www.iweapps.com",
			"http://localhost:3001",
			"http://localhost:5173",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type", "X-Client-State"},
		ExposeHeaders:    []string{"Content-Length", "X-Client-State"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	
	// Increase memory limit for multipart forms
	r.MaxMultipartMemory = 32 << 20
	s.defineRoutes(r)

	return r
}

func (s *Server) defineRoutes(router *gin.Engine) {
	apirouter := router.Group("/api/v1")
	
	// WebSocket endpoints
	// Unified endpoint supports both authenticated and public connections
	router.GET("/ws", s.OptionalAuthorize(), websocket.HandleWebSocketAuto(s.WSHub))
	// Dedicated authenticated endpoint remains available
	router.GET("/ws/auth", s.Authorize(), websocket.HandleWebSocketAuth(s.WSHub))
	// Health check for WebSocket service
	router.GET("/ws/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":       "ok",
			"connected":    s.WSHub.GetConnectedUsers(),
		})
	})
	
	// Public routes (no authentication required)
	apirouter.POST("/auth/signup", s.handleSignup())
	apirouter.POST("/auth/login", s.handleLogin())
	apirouter.POST("/google/user/login", s.handleGoogleLogin())

	// Protected routes (authentication required)
	authorized := apirouter.Group("/auth")
	authorized.Use(s.Authorize())
	{
		authorized.POST("/logout", s.handleLogout())
	}

	// File upload routes (authentication required)
	uploads := apirouter.Group("/upload")
	uploads.Use(s.Authorize())
	{
		uploads.POST("", s.handleFileUpload())
		uploads.GET("/status/:id", s.handleGetUploadStatus())
		uploads.GET("/my-uploads", s.handleGetUserUploads())
	}

	// AI routes (authentication required)
	aiRoutes := apirouter.Group("/ai")
	aiRoutes.Use(s.Authorize())
	{
		aiRoutes.POST("/analyze", s.handleAIAnalyze())
	}

	// Chat routes (authentication required)
	chatRoutes := apirouter.Group("/chat")
	chatRoutes.Use(s.Authorize())
	{
		chatRoutes.POST("/conversations", s.handleCreateConversation())
		chatRoutes.GET("/conversations", s.handleListConversations())
		chatRoutes.GET("/conversations/:id/messages", s.handleGetMessages())
		chatRoutes.POST("/messages", s.handlePostMessage())
	}
}
