package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/OsGift/taskflow-api/internal/middleware"
	"github.com/OsGift/taskflow-api/internal/models"
	"github.com/OsGift/taskflow-api/internal/services"
	"github.com/OsGift/taskflow-api/internal/utils"
)

// UserHandler handles user related HTTP requests
type UserHandler struct {
	userService *services.UserService
	authService *services.AuthService // Needed for admin creation to hash temp password
	validator   *validator.Validate
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(us *services.UserService, as *services.AuthService) *UserHandler {
	return &UserHandler{
		userService: us,
		authService: as,
		validator:   validator.New(),
	}
}

// CreateAdminUser handles creating a new admin user (requires 'user:create_admin' permission)
func (h *UserHandler) CreateAdminUser(w http.ResponseWriter, r *http.Request) {
	var req models.UserRegisterRequest // Using existing register request for email/password
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Generate a temporary password
	tempPassword := utils.GenerateRandomString(12) // You'll need to implement this in utils/helpers.go

	// Delegate to authService's register logic, but indicate it's an admin creation
	userResponse, err := h.authService.RegisterUser(req, true, tempPassword) // is_admin_creation = true
	if err != nil {
		if err.Error() == "email already registered" {
			utils.RespondWithError(w, http.StatusConflict, err.Error())
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create admin user")
		return
	}

	// Return partial response to avoid exposing temp password in API, it's sent via email
	response := map[string]interface{}{
		"message": "Admin user created successfully. Temporary password sent to email.",
		"user_id": userResponse.ID,
		"email":   userResponse.Email,
	}
	utils.RespondWithJSON(w, http.StatusCreated, response)
}

// GetUserByID retrieves a user profile by ID
func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetUserID := vars["id"] // ID of the user whose profile is being requested

	authContext, err := middleware.GetAuthContext(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Check if the authenticated user is requesting their own profile
	if authContext.UserID.Hex() == targetUserID {
		userResponse, err := h.userService.GetUserResponseByID(targetUserID)
		if err != nil {
			utils.RespondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		utils.RespondWithJSON(w, http.StatusOK, userResponse)
		return
	}

	// If not own profile, check for 'user:read_all' permission
	if !authContext.HasPermission("user:read_all") {
		utils.RespondWithError(w, http.StatusForbidden, "You do not have permission to view this user's profile")
		return
	}

	userResponse, err := h.userService.GetUserResponseByID(targetUserID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, userResponse)
}

// UpdateUserRole updates a user's role (Admin only)
func (h *UserHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetUserID := vars["id"] // ID of the user whose role is being updated

	var req models.UpdateUserRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	authContext, err := middleware.GetAuthContext(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// The permission check for "user:update_role" is done by the middleware before reaching here.
	// Additional check: Cannot change the role of a user to 'Admin' if not explicitly permitted
	// And Super Admin (the initial seeded Admin) role cannot be changed by another admin.
	targetUser, err := h.userService.GetUserByID(targetUserID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "Target user not found")
		return
	}

	targetRole, err := h.userService.GetRoleByID(targetUser.RoleID.Hex())
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Could not determine target user's current role")
		return
	}

	// Prevent changing role to/from Admin unless specific conditions met
	if targetRole.Name == "Admin" && authContext.RoleName == "Admin" && targetUserID != authContext.UserID.Hex() {
		// This is the check to prevent one Admin from changing another Admin's role
		utils.RespondWithError(w, http.StatusForbidden, "You cannot change the role of another Admin.")
		return
	}
	if req.RoleName == "Admin" && authContext.RoleName == "Admin" && targetUserID == authContext.UserID.Hex() {
		// Prevent an admin from demoting themselves (if they are the "Super Admin")
		// More robust Super Admin identification would be needed for production.
		utils.RespondWithError(w, http.StatusForbidden, "You cannot change your own role from Admin.")
		return
	}

	// Prevent changing role to Admin if not explicitly allowed by a more granular permission
	// (currently covered by 'user:update_role' which is for Admin role)
	// You might introduce a 'user:assign_admin_role' permission for this if needed.

	userResponse, err := h.userService.UpdateUserRole(targetUserID, req.RoleName)
	if err != nil {
		if err.Error() == "user not found or role not changed" || err.Error() == "new role not found" || err.Error() == "invalid user ID format" {
			utils.RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update user role")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, userResponse)
}

// UpdateUserProfile handles updating a user's first_name, last_name, and profile_picture_url
func (h *UserHandler) UpdateUserProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetUserID := vars["id"] // ID of the user whose profile is being updated

	var req models.UpdateUserProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	authContext, err := middleware.GetAuthContext(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// A user can update their own profile, or an admin with 'user:update_profile' permission can update any profile.
	if authContext.UserID.Hex() != targetUserID {
		if !authContext.HasPermission("user:update_profile") {
			utils.RespondWithError(w, http.StatusForbidden, "You do not have permission to update this user's profile")
			return
		}
	}

	userResponse, err := h.userService.UpdateUserProfile(targetUserID, &req)
	if err != nil {
		if err.Error() == "user not found or no changes made to profile" || err.Error() == "invalid user ID format" {
			utils.RespondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update user profile")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, userResponse)
}

// ListUsers handles listing all users for admins with pagination and filters
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	// Permission 'user:read_all' is checked by middleware

	// Pagination parameters
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page, err := strconv.ParseInt(pageStr, 10, 64)
	if err != nil || page < 1 {
		page = 1 // Default page
	}
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil || limit < 1 || limit > 100 { // Max 100 items per page
		limit = 10 // Default limit
	}

	// Build filter based on query parameters (e.g., by email, role, etc. if needed)
	filter := primitive.M{}
	// Example: filter by email fragment (case-insensitive)
	emailFilter := r.URL.Query().Get("email_like")
	if emailFilter != "" {
		filter["email"] = primitive.Regex{Pattern: emailFilter, Options: "i"}
	}
	// Example: filter by role name
	roleNameFilter := r.URL.Query().Get("role_name")
	if roleNameFilter != "" {
		role, err := h.userService.GetRoleByName(roleNameFilter)
		if err == nil {
			filter["role_id"] = role.ID
		} else {
			// If role name doesn't exist, return empty list or error
			utils.RespondWithJSON(w, http.StatusOK, models.UserListResponse{
				Users:      []models.UserResponse{},
				TotalCount: 0, Page: page, Limit: limit,
			})
			return
		}
	}

	usersResponse, err := h.userService.ListUsers(filter, page, limit)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve users")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, usersResponse)
}
