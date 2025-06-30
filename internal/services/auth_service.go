package services

import (
	"errors"
	"fmt"
	"sync" // For in-memory reset tokens
	"time"
	// For HTML email templates
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/OsGift/taskflow-api/internal/models"
	"github.com/OsGift/taskflow-api/internal/utils"
)

// In-memory store for password reset tokens.
// In a real application, this should be persisted (e.g., MongoDB, Redis)
// and more robustly handled (e.g., single-use tokens, rate limiting).
var (
	passwordResetTokens = make(map[string]primitive.ObjectID) // token -> user ID
	tokenMutex          sync.Mutex
)

// AuthService provides methods for user authentication and JWT operations
type AuthService struct {
	userService         *UserService
	jwtSecret           []byte
	passwordResetSecret []byte // New secret for password reset tokens
}

// NewAuthService creates a new AuthService
func NewAuthService(us *UserService, jwtSecret, passwordResetSecret []byte) *AuthService {
	return &AuthService{
		userService:         us,
		jwtSecret:           jwtSecret,
		passwordResetSecret: passwordResetSecret,
	}
}

// RegisterUser handles user registration. Can also register admins.
func (s *AuthService) RegisterUser(req models.UserRegisterRequest, isAdminCreation bool, tempPassword string) (*models.UserResponse, error) {
	// Check if user with this email already exists
	existingUser, _ := s.userService.GetUserByEmail(req.Email)
	if existingUser != nil {
		return nil, errors.New("email already registered")
	}

	var hashedPassword string
	var needsPasswordChange bool
	var role *models.Role
	var err error

	if isAdminCreation {
		hashedPassword, err = utils.HashPassword(tempPassword)
		if err != nil {
			return nil, errors.New("failed to hash temporary password")
		}
		needsPasswordChange = true
		role, err = s.userService.GetRoleByName("Admin")
		if err != nil {
			return nil, errors.New("admin role not found")
		}
	} else {
		hashedPassword, err = utils.HashPassword(req.Password)
		if err != nil {
			return nil, errors.New("failed to hash password")
		}
		needsPasswordChange = false
		role, err = s.userService.GetRoleByName("User")
		if err != nil {
			return nil, errors.New("default user role not found")
		}
	}

	if err != nil {
		return nil, err // From role lookup
	}

	newUser := &models.User{
		FirstName:           "New",  // Default for now, can be updated later
		LastName:            "User", // Default for now
		Email:               req.Email,
		Password:            hashedPassword,
		RoleID:              role.ID,
		ProfilePictureURL:   "https://placehold.co/150x150/cccccc/ffffff?text=Avatar", // Default avatar
		IsEmailVerified:     false,                                                    // Not verified initially
		NeedsPasswordChange: needsPasswordChange,                                      // Set based on admin creation
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	userResponse, err := s.userService.CreateUser(newUser)
	if err != nil {
		return nil, err
	}

	// Send email based on creation type
	if isAdminCreation {
		emailData := struct {
			FirstName         string
			TemporaryPassword string
			LoginLink         string
			Year              int
		}{
			FirstName:         userResponse.FirstName,
			TemporaryPassword: tempPassword,
			LoginLink:         "http://localhost:3000/login", // Frontend login URL
			Year:              time.Now().Year(),
		}
		go utils.SendEmail("admin_temp_password", "Your TaskFlow Admin Account Details", req.Email, emailData)
	} else {
		verificationToken, err := utils.GenerateVerificationToken(userResponse.ID, s.jwtSecret) // Pass hex string
		if err != nil {
			fmt.Printf("Warning: Failed to generate verification token for %s: %v\n", req.Email, err)
			// Proceed without sending verification email if token generation fails
		} else {
			emailData := struct {
				FirstName        string
				VerificationLink string
				Year             int
			}{
				FirstName:        userResponse.FirstName,
				VerificationLink: fmt.Sprintf("http://localhost:3000/verify-email?token=%s", verificationToken), // Frontend verify URL
				Year:             time.Now().Year(),
			}
			go utils.SendEmail("welcome", "Welcome to TaskFlow! Please verify your email.", req.Email, emailData)
		}
	}

	return userResponse, nil
}

// LoginUser handles user login and JWT generation
func (s *AuthService) LoginUser(req models.UserLoginRequest) (*models.LoginResponse, error) {
	user, err := s.userService.GetUserByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !utils.CheckPasswordHash(req.Password, user.Password) {
		return nil, errors.New("invalid credentials")
	}

	// Get user's role name
	role, err := s.userService.GetRoleByID(user.RoleID.Hex())
	if err != nil {
		return nil, errors.New("user role not found") // Should not happen if roles are seeded
	}

	// Generate JWT token
	tokenString, err := utils.GenerateToken(user.ID, user.Email, user.RoleID, s.jwtSecret)
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	return &models.LoginResponse{
		Message:             "Login successful",
		Token:               tokenString,
		UserID:              user.ID.Hex(),
		RoleName:            role.Name,
		NeedsPasswordChange: user.NeedsPasswordChange, // Pass this flag to frontend
	}, nil
}

// ValidateToken validates a JWT token string (used by middleware)
func (s *AuthService) ValidateToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

// ForgotPassword generates a password reset token and "sends" it to the user's email
func (s *AuthService) ForgotPassword(email string) error {
	user, err := s.userService.GetUserByEmail(email)
	if err != nil {
		// For security, don't reveal if email exists or not
		fmt.Printf("Attempted password reset for non-existent email: %s\n", email)
		return nil // Return nil to prevent leaking user existence
	}

	resetToken, err := utils.GeneratePasswordResetToken(user.ID, s.passwordResetSecret)
	if err != nil {
		return errors.New("failed to generate reset token")
	}

	// Store token in-memory with user ID. In production, this would be a DB/Redis entry
	tokenMutex.Lock()
	passwordResetTokens[resetToken] = user.ID
	tokenMutex.Unlock()

	// Simulate sending email with reset link
	emailData := struct {
		ResetLink string
		Year      int
	}{
		ResetLink: fmt.Sprintf("http://localhost:3000/reset-password?token=%s", resetToken), // Frontend reset password URL
		Year:      time.Now().Year(),
	}
	go utils.SendEmail("forgot_password", "Password Reset Request for TaskFlow", email, emailData)

	// Remove token after some time (e.g., 1 hour)
	go func(token string) {
		time.Sleep(1 * time.Hour)
		tokenMutex.Lock()
		delete(passwordResetTokens, token)
		tokenMutex.Unlock()
		fmt.Printf("Password reset token %s expired and removed.\n", token)
	}(resetToken)

	return nil
}

// ResetPassword validates the token and updates the user's password
func (s *AuthService) ResetPassword(tokenString, newPassword string) error {
	tokenMutex.Lock()
	userID, exists := passwordResetTokens[tokenString]
	tokenMutex.Unlock()

	if !exists {
		return errors.New("invalid or expired password reset token")
	}

	// Remove the token after use (important for security)
	tokenMutex.Lock()
	delete(passwordResetTokens, tokenString)
	tokenMutex.Unlock()

	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return errors.New("failed to hash new password")
	}

	err = s.userService.UpdateUserPassword(userID, hashedPassword)
	if err != nil {
		return errors.New("failed to update password in database")
	}

	return nil
}

// ChangeTemporaryPassword allows a logged-in user with needs_password_change to set a new password
func (s *AuthService) ChangeTemporaryPassword(userID primitive.ObjectID, oldPassword, newPassword string) error {
	user, err := s.userService.GetUserByID(userID.Hex())
	if err != nil {
		return errors.New("user not found")
	}

	if !user.NeedsPasswordChange {
		return errors.New("password change not required for this account")
	}

	// Verify old password (even if temporary)
	if !utils.CheckPasswordHash(oldPassword, user.Password) {
		return errors.New("invalid old password")
	}

	hashedNewPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return errors.New("failed to hash new password")
	}

	err = s.userService.UpdateUserPasswordAndNeedsChange(userID, hashedNewPassword, false)
	if err != nil {
		return errors.New("failed to update password")
	}
	return nil
}

// AuthenticatedUserContext fetches the full AuthContext for a given user ID and role ID.
// This is used by the middleware to prepare the context.
func (s *AuthService) AuthenticatedUserContext(userID primitive.ObjectID, roleID primitive.ObjectID) (*models.AuthContext, error) {
	user, err := s.userService.GetUserByID(userID.Hex())
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	role, err := s.userService.GetRoleByID(roleID.Hex())
	if err != nil {
		return nil, fmt.Errorf("user role not found: %w", err)
	}

	return &models.AuthContext{
		UserID:              user.ID,
		RoleID:              role.ID,
		RoleName:            role.Name,
		Permissions:         role.Permissions,
		IsEmailVerified:     user.IsEmailVerified,
		NeedsPasswordChange: user.NeedsPasswordChange,
	}, nil
}
