package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a user in the system
type User struct {
	ID                  primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	FirstName           string             `bson:"first_name" json:"first_name" validate:"required,min=2,max=50"`
	LastName            string             `bson:"last_name" json:"last_name" validate:"required,min=2,max=50"`
	Email               string             `bson:"email" json:"email" validate:"required,email"`
	Password            string             `bson:"password" json:"-"` // Exclude from JSON output
	RoleID              primitive.ObjectID `bson:"role_id" json:"role_id"`
	ProfilePictureURL   string             `bson:"profile_picture_url,omitempty" json:"profile_picture_url,omitempty"`
	IsEmailVerified     bool               `bson:"is_email_verified" json:"is_email_verified"`
	NeedsPasswordChange bool               `bson:"needs_password_change" json:"needs_password_change"` // New field
	CreatedAt           time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt           time.Time          `bson:"updated_at" json:"updated_at"`
}

// UserLoginRequest is used for login requests (email and password only)
type UserLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// UserRegisterRequest is used for registration requests (email and password only)
type UserRegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// UserResponse is used for user data returned to client
type UserResponse struct {
	ID                  string    `json:"id"`
	FirstName           string    `json:"first_name"`
	LastName            string    `json:"last_name"`
	Email               string    `json:"email"`
	RoleName            string    `json:"role_name"` // Populated from Role collection
	ProfilePictureURL   string    `json:"profile_picture_url,omitempty"`
	IsEmailVerified     bool      `json:"is_email_verified"`
	NeedsPasswordChange bool      `json:"needs_password_change"` // New field
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// LoginResponse is the response body for a successful login
type LoginResponse struct {
	Message             string `json:"message"`
	Token               string `json:"token"`
	UserID              string `json:"user_id"`
	RoleName            string `json:"role_name"`
	NeedsPasswordChange bool   `json:"needs_password_change"` // Added for frontend redirection
}

// UpdateUserRoleRequest for changing user roles
type UpdateUserRoleRequest struct {
	RoleName string `json:"role_name" validate:"required"`
}

// UpdateUserProfileRequest for updating user profile details
type UpdateUserProfileRequest struct {
	FirstName         *string `json:"first_name,omitempty" validate:"omitempty,min=2,max=50"`
	LastName          *string `json:"last_name,omitempty" validate:"omitempty,min=2,max=50"`
	ProfilePictureURL *string `json:"profile_picture_url,omitempty" validate:"omitempty,url"`
}

// ForgotPasswordRequest for initiating password reset
type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ResetPasswordRequest for completing password reset
type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

// ChangeTemporaryPasswordRequest for an admin's first login password change
type ChangeTemporaryPasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

// AuthContext holds authenticated user details to be stored in request context
type AuthContext struct {
	UserID              primitive.ObjectID
	RoleID              primitive.ObjectID
	RoleName            string
	Permissions         []Permission
	IsEmailVerified     bool
	NeedsPasswordChange bool
}

// HasPermission checks if the AuthContext has a specific permission
func (ac *AuthContext) HasPermission(permission string) bool {
	for _, p := range ac.Permissions {
		if p.Action == permission {
			return true
		}
	}
	return false
}

// UserListResponse holds a list of users and pagination metadata
type UserListResponse struct {
	Users      []UserResponse `json:"users"`
	TotalCount int64          `json:"total_count"`
	Page       int64          `json:"page"`
	Limit      int64          `json:"limit"`
}
