package main

import (
	"fmt"
	"log"
	"net/http"

	"pet-hotel/internal/config"
	"pet-hotel/internal/database"
	"pet-hotel/internal/handlers"
	"pet-hotel/internal/middleware"
	"pet-hotel/internal/models"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()

	// Initialize Database (Postgres with Neon/Supabase or local SQLite fallback)
	db := database.InitDB(cfg)

	// Set Gin mode
	gin.SetMode(cfg.Environment)
	r := gin.Default()

	// Global Security Headers Middleware
	r.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	})

	// Load HTML templates
	r.LoadHTMLGlob("templates/*")

	// Initialize Handlers
	authHandler := handlers.NewAuthHandler(db, cfg)
	petHandler := handlers.NewPetHandler(db)
	bookingHandler := handlers.NewBookingHandler(db)
	adminHandler := handlers.NewAdminHandler(db)

	// Public Routes
	r.GET("/", func(c *gin.Context) {
		// If already logged in, redirect
		if cookie, err := c.Cookie("auth_token"); err == nil && cookie != "" {
			c.Redirect(http.StatusSeeOther, "/dashboard")
			return
		}
		c.Redirect(http.StatusSeeOther, "/login")
	})

	// Auth Routes (protected by Rate Limiting: max 10 requests / min, burst 5)
	authGroup := r.Group("/")
	authGroup.Use(middleware.RateLimitLogin(10, 5))
	{
		authGroup.GET("/login", authHandler.ShowLoginPage)
		authGroup.POST("/login", authHandler.Login)
		authGroup.GET("/register", authHandler.ShowRegisterPage)
		authGroup.POST("/register", authHandler.Register)
	}
	r.GET("/logout", authHandler.Logout)

	// User Protected Routes (Requires valid session / JWT)
	userRoutes := r.Group("/")
	userRoutes.Use(middleware.AuthRequired(cfg))
	{
		userRoutes.GET("/dashboard", bookingHandler.ShowDashboard)

		// Form 1: Pet Registration & Management (Strict IDOR checks)
		userRoutes.GET("/pets/new", petHandler.ShowPetForm)
		userRoutes.POST("/pets/new", petHandler.CreatePet)
		userRoutes.GET("/pets/edit/:id", petHandler.ShowEditPetForm)
		userRoutes.POST("/pets/edit/:id", petHandler.UpdatePet)
		userRoutes.POST("/pets/delete/:id", petHandler.DeletePet)

		// Form 2: Room Booking & Management
		userRoutes.GET("/bookings/new", bookingHandler.ShowBookingForm)
		userRoutes.POST("/bookings/new", bookingHandler.CreateBooking)
		userRoutes.POST("/bookings/cancel/:id", bookingHandler.CancelBooking)
	}

	// Admin Protected Routes (Requires Auth + Admin Role)
	adminRoutes := r.Group("/admin")
	adminRoutes.Use(middleware.AuthRequired(cfg))
	adminRoutes.Use(middleware.RequireRole(models.RoleAdmin))
	{
		adminRoutes.GET("", adminHandler.ShowAdminDashboard)
		adminRoutes.POST("/bookings/:id/status", adminHandler.UpdateBookingStatus)
		adminRoutes.POST("/rooms/:id/toggle", adminHandler.ToggleRoomStatus)
	}

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("🚀 Pet Hotel Reservation System listening on http://localhost%s\n", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}
