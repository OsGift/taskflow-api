package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"

	"github.com/OsGift/taskflow-api/internal/middleware"
	"github.com/OsGift/taskflow-api/internal/models"
	"github.com/OsGift/taskflow-api/internal/services"
	"github.com/OsGift/taskflow-api/internal/utils"
)

// AuthHandler handles authentication related HTTP requests
type AuthHandler struct {
	authService *services.AuthService
	userService *services.UserService // To get role name for login response
	validator   *validator.Validate
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(as *services.AuthService, us *services.UserService) *AuthHandler {
	return &AuthHandler{
		authService: as,
		userService: us,
		validator:   validator.New(),
	}
}

// RegisterUser handles user registration via POST /register
func (h *AuthHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var req models.UserRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// This endpoint is for regular user registration. Admin creation is a separate process.
	userResponse, err := h.authService.RegisterUser(req, false, "") // not admin creation, no temp password
	if err != nil {
		if err.Error() == "email already registered" {
			utils.RespondWithError(w, http.StatusConflict, err.Error())
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to register user")
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, userResponse)
}

// LoginUser handles user login via POST /login
func (h *AuthHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
	var req models.UserLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	loginResponse, err := h.authService.LoginUser(req)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, loginResponse)
}

// ForgotPassword handles initiating the password reset process
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req models.ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// It's important NOT to reveal if the email exists or not for security reasons.
	// Always return a success message if the email format is valid.
	err := h.authService.ForgotPassword(req.Email)
	if err != nil {
		// Log internal error but return generic success to client
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to initiate password reset")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "If an account with that email exists, a password reset link has been sent."})
}

// ResetPassword handles completing the password reset process
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req models.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	err := h.authService.ResetPassword(req.Token, req.NewPassword)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error()) // Specific errors are OK here
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Password has been reset successfully."})
}

// ChangeTemporaryPassword handles admin's first login password change
func (h *AuthHandler) ChangeTemporaryPassword(w http.ResponseWriter, r *http.Request) {
	var req models.ChangeTemporaryPasswordRequest
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

	// Ensure this is only for users who actually need a password change
	if !authContext.NeedsPasswordChange {
		utils.RespondWithError(w, http.StatusForbidden, "Password change not required for this account.")
		return
	}

	err = h.authService.ChangeTemporaryPassword(authContext.UserID, req.OldPassword, req.NewPassword)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Password updated successfully. You can now access the dashboard."})
}

// VerifyEmail handles setting a user's email as verified.
// This endpoint expects a verification token in the query params.
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	tokenString := r.URL.Query().Get("token")
	if tokenString == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Missing verification token")
		return
	}

	// In a real app, this verification token would contain the user ID and be validated.
	// For this simplified example, we'll assume the token is just a placeholder
	// and we get the user ID from the JWT of the *current* authenticated user.
	// A more robust system would validate the token itself to get the user ID.

	authContext, err := middleware.GetAuthContext(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Check if user is already verified
	if authContext.IsEmailVerified {
		utils.RespondWithError(w, http.StatusBadRequest, "Email already verified")
		return
	}

	err = h.userService.VerifyUserEmail(authContext.UserID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Email verified successfully."})
}
