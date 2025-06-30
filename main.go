package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"github.com/OsGift/taskflow-api/api"
	"github.com/OsGift/taskflow-api/internal/config"
	"github.com/OsGift/taskflow-api/internal/database"
	"github.com/OsGift/taskflow-api/internal/handlers"
	"github.com/OsGift/taskflow-api/internal/middleware"
	"github.com/OsGift/taskflow-api/internal/services"
	"github.com/OsGift/taskflow-api/internal/utils" // Import utils for mailer initialization
)

func main() {
	// 1. Load configuration
	cfg, err := config.LoadConfig(".env")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// 2. Initialize Mailer
	if err := utils.InitMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword); err != nil {
		log.Fatalf("Error initializing mailer: %v", err)
	}

	// 3. Connect to MongoDB
	client, err := database.ConnectMongoDB(cfg.MongoURI, cfg.DBName)
	if err != nil {
		log.Fatalf("Error connecting to MongoDB: %v", err)
	}
	defer func() {
		if err = client.Disconnect(context.Background()); err != nil {
			log.Printf("Error disconnecting from MongoDB: %v", err)
		}
	}()

	// 4. Initialize services
	userService := services.NewUserService(client.Database(cfg.DBName))
	taskService := services.NewTaskService(client.Database(cfg.DBName))
	authService := services.NewAuthService(userService, []byte(cfg.JWTSecret), []byte(cfg.PasswordResetSecret))
	dashboardService := services.NewDashboardService(client.Database(cfg.DBName))
	uploadService := services.NewUploadService(cfg.CloudinaryCloudName, cfg.CloudinaryAPIKey, cfg.CloudinaryAPISecret)

	// 5. Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, userService)
	userHandler := handlers.NewUserHandler(userService, authService)
	taskHandler := handlers.NewTaskHandler(taskService)
	dashboardHandler := handlers.NewDashboardHandler(dashboardService)
	uploadHandler := handlers.NewUploadHandler(uploadService)

	// 6. Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware([]byte(cfg.JWTSecret), userService, authService)

	// 7. Seed default roles if they don't exist
	if err := database.SeedDefaultRoles(client.Database(cfg.DBName)); err != nil {
		log.Fatalf("Error seeding default roles: %v", err)
	}

	// 8. Setup router
	router := mux.NewRouter()
	api.SetupRoutes(router, authMiddleware, authHandler, userHandler, taskHandler, dashboardHandler, uploadHandler)

	// --- CORS: Allow All Origins ---
	c := cors.AllowAll()
	handlerWithCORS := c.Handler(router)

	// 9. Start HTTP server
	log.Printf("Server starting on port %s", cfg.Port)
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handlerWithCORS,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on %s: %v\n", cfg.Port, err)
	}
}
