package api

import (
	"github.com/gorilla/mux"

	"github.com/OsGift/taskflow-api/internal/handlers"
	"github.com/OsGift/taskflow-api/internal/middleware"
)

// SetupRoutes configures all API routes
func SetupRoutes(
	router *mux.Router,
	authMiddleware *middleware.AuthMiddleware,
	authHandler *handlers.AuthHandler,
	userHandler *handlers.UserHandler,
	taskHandler *handlers.TaskHandler,
	dashboardHandler *handlers.DashboardHandler, // New
	uploadHandler *handlers.UploadHandler, // New
) {
	v1 := router.PathPrefix("/api/v1").Subrouter()

	// Authentication routes (public)
	v1.HandleFunc("/auth/register", authHandler.RegisterUser).Methods("POST")
	v1.HandleFunc("/auth/login", authHandler.LoginUser).Methods("POST")
	v1.HandleFunc("/auth/forgot_password", authHandler.ForgotPassword).Methods("POST")
	v1.HandleFunc("/auth/reset_password", authHandler.ResetPassword).Methods("POST")
	// This endpoint is for logged-in users to verify their email, using a token from email
	v1.HandleFunc("/auth/verify_email", authMiddleware.JWTAuth(authHandler.VerifyEmail, "")).Methods("POST")
	// For admins who log in with a temporary password to set a permanent one
	v1.HandleFunc("/auth/change_temp_password", authMiddleware.JWTAuth(authHandler.ChangeTemporaryPassword, "")).Methods("POST")

	// User routes (protected)
	// Admin can create another admin user
	v1.HandleFunc("/users/admin", authMiddleware.JWTAuth(userHandler.CreateAdminUser, "user:create_admin")).Methods("POST")
	// Get user by ID (own profile or any if admin)
	v1.HandleFunc("/users/{id}", authMiddleware.JWTAuth(userHandler.GetUserByID, "user:read_own")).Methods("GET")
	// Update user role (admin only)
	v1.HandleFunc("/users/{id}/role", authMiddleware.JWTAuth(userHandler.UpdateUserRole, "user:update_role")).Methods("PUT")
	// Update user profile (own profile or any if admin with permission)
	v1.HandleFunc("/users/{id}/profile", authMiddleware.JWTAuth(userHandler.UpdateUserProfile, "user:update_profile")).Methods("PUT")
	// List all users (admin only, with pagination/filters)
	v1.HandleFunc("/users", authMiddleware.JWTAuth(userHandler.ListUsers, "user:read_all")).Methods("GET")

	// Task routes (protected)
	v1.HandleFunc("/tasks", authMiddleware.JWTAuth(taskHandler.CreateTask, "task:create")).Methods("POST")
	v1.HandleFunc("/tasks", authMiddleware.JWTAuth(taskHandler.GetTasks, "task:read_own")).Methods("GET")
	v1.HandleFunc("/tasks/{id}", authMiddleware.JWTAuth(taskHandler.GetTaskByID, "task:read_own")).Methods("GET")
	v1.HandleFunc("/tasks/{id}", authMiddleware.JWTAuth(taskHandler.UpdateTask, "task:update_own")).Methods("PUT")
	v1.HandleFunc("/tasks/{id}", authMiddleware.JWTAuth(taskHandler.DeleteTask, "task:delete_own")).Methods("DELETE")

	// Dashboard routes (protected, typically admin/manager access)
	v1.HandleFunc("/dashboard/metrics", authMiddleware.JWTAuth(dashboardHandler.GetDashboardMetrics, "dashboard:read_metrics")).Methods("GET")

	// File Uploads (protected)
	v1.HandleFunc("/upload", authMiddleware.JWTAuth(uploadHandler.UploadFile, "user:update_profile")).Methods("POST") // Example: only users who can update profiles can upload
}
