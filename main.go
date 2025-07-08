package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/indrawanalghifary/whatsgo/internal/config"
	"github.com/indrawanalghifary/whatsgo/internal/handlers"
	"github.com/indrawanalghifary/whatsgo/internal/session"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize session manager
	sessionManager, err := session.NewManager(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize session manager: %v", err)
	}
	defer sessionManager.Close()

	// Setup Gin router
	r := gin.Default()

	// Initialize handlers
	apiHandler := handlers.NewAPIHandler(sessionManager, cfg)

	// Setup routes
	setupRoutes(r, apiHandler)

	// Setup server
	server := &http.Server{
		Addr:    cfg.Server.Host + ":" + cfg.Server.Port,
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}

func setupRoutes(r *gin.Engine, handler *handlers.APIHandler) {
	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API routes
	api := r.Group("/api/v1")
	{
		// Session management
		api.POST("/sessions", handler.CreateSession)
		api.GET("/sessions", handler.ListSessions)
		api.GET("/sessions/:sessionId", handler.GetSession)
		api.DELETE("/sessions/:sessionId", handler.DeleteSession)

		// Authentication
		api.GET("/sessions/:sessionId/qr", handler.GetQRCode)
		api.POST("/sessions/:sessionId/logout", handler.LogoutSession)

		// Messaging
		api.POST("/sessions/:sessionId/send", handler.SendMessage)
		api.GET("/sessions/:sessionId/chats", handler.GetChats)

		// Status
		api.GET("/sessions/:sessionId/status", handler.GetSessionStatus)
	}

	// Serve static files for QR codes
	r.Static("/qr", "./qr_codes")
}