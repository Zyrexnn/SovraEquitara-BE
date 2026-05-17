package main

import (
	"log"

	"sovraequitara-be/internal/config"
	"sovraequitara-be/internal/handler"
	"sovraequitara-be/internal/middleware"
	"sovraequitara-be/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load Configuration
	cfg := config.LoadConfig()

	// Initialize Database — Native PostgreSQL (no Supabase)
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  cfg.DatabaseURL,
		PreferSimpleProtocol: true, // Disable prepared statements for pgbouncer compatibility
	}), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Database connected successfully!")

	// Initialize Repository and Handler
	repo := repository.NewRepository(db)
	h := handler.NewHandler(repo, cfg)

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Global Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, HEAD, PUT, DELETE, PATCH",
	}))

	// Serve static files for image uploads
	app.Static("/uploads", "./uploads")

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
	})

	// API Group
	api := app.Group("/api")

	// Public Routes (Auth)
	authGroup := api.Group("/auth")
	authGroup.Post("/register", h.AuthRegister)
	authGroup.Post("/login", h.AuthLogin)
	authGroup.Post("/verify", h.AuthVerify)
	authGroup.Post("/admin-login", h.AdminLogin)
	authGroup.Post("/forgot-password", h.ForgotPassword)
	authGroup.Post("/reset-password", h.ResetPassword)

	// Public Routes (Data)
	api.Get("/categories", h.GetCategories)
	api.Get("/public-reports", h.GetPublicReports)
	api.Get("/reports/:id/comments", h.GetComments)
	api.Get("/leaderboard", h.GetLeaderboard)

	// Profile Listing (Public for profile popup)
	api.Get("/profiles", h.GetAllProfiles)
	api.Get("/profiles/:id", h.GetProfileByID)

	// Protected Routes (Requires valid JWT)
	protected := api.Group("/", middleware.Protected(cfg))

	// User Routes
	protected.Get("/my-reports", h.GetMyReports)
	protected.Post("/reports", h.CreateReport)
	protected.Delete("/reports/:id", h.DeleteReport)
	protected.Post("/reports/:id/comments", h.AddComment)
	protected.Post("/reports/:id/vote", h.VoteReport)
	protected.Get("/reports/stats", h.GetReportStats)
	protected.Get("/profile", h.GetProfile)
	protected.Put("/profile", h.UpdateProfile)
	protected.Post("/profile/avatar", h.UploadAvatar)

	// Chat Routes (any authenticated user can chat)
	protected.Post("/chat/send", h.SendChatMessage)
	protected.Get("/chat/messages", h.GetMyMessages)

	// Admin Routes (Requires 'admin' role)
	adminGroup := protected.Group("/admin", middleware.AdminOnly(repo))
	adminGroup.Get("/reports", h.GetAllReports)
	adminGroup.Patch("/reports/:id/verify", h.VerifyReport)
	adminGroup.Patch("/reports/:id/resolve", h.ResolveReport)
	adminGroup.Post("/ai-assistant", h.AIAssistant)

	// Admin Chat Inbox
	adminGroup.Get("/chat/conversations", h.GetAllConversations)
	adminGroup.Get("/chat/conversations/:id/messages", h.GetConversationMessages)
	adminGroup.Post("/chat/conversations/:id/reply", h.ReplyChatMessage)

	// Super Admin Routes (Requires 'super_admin' role)
	superAdminGroup := protected.Group("/superadmin", middleware.SuperAdminOnly(repo))
	
	// Super Admin can cancel reports
	superAdminGroup.Patch("/reports/:id/cancel", h.CancelReport)
	
	// User Stats (Super Admin)
	superAdminGroup.Get("/profiles/:id/stats", h.GetUserStats)
	
	// CRUD Admin
	superAdminGroup.Get("/admins", h.GetAdmins)
	superAdminGroup.Post("/admins", h.CreateAdmin)
	superAdminGroup.Put("/admins/:id", h.UpdateAdmin)
	superAdminGroup.Delete("/admins/:id", h.DeleteAdmin)

	// Start server
	log.Printf("Server starting on port %s", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
