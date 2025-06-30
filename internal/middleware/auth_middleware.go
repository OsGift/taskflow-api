package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/OsGift/taskflow-api/internal/models"
	"github.com/OsGift/taskflow-api/internal/services"
	"github.com/OsGift/taskflow-api/internal/utils"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	ContextKeyAuthContext ContextKey = "authContext" // New context key for our AuthContext struct
)

// AuthMiddleware handles JWT authentication and sets user context
type AuthMiddleware struct {
	jwtSecret   []byte
	userService *services.UserService
	authService *services.AuthService // Added Auth service
}

// NewAuthMiddleware creates a new AuthMiddleware
// Changed constructor to accept AuthService
func NewAuthMiddleware(secret []byte, us *services.UserService, as *services.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret:   secret,
		userService: us,
		authService: as, // Assign auth service
	}
}

// JWTAuth middleware verifies the JWT token and populates AuthContext in request context
// requiredPermission is the minimum permission needed to pass this middleware.
// If it's an empty string (""), it means only authentication is required, no specific permission.
// If the handler needs more nuanced permission checks (e.g., resource ownership vs. global access),
// it should perform those using the AuthContext.HasPermission method.
func (m *AuthMiddleware) JWTAuth(next http.HandlerFunc, requiredPermission string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, "Missing authorization header")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			utils.RespondWithError(w, http.StatusUnauthorized, "Invalid authorization header format")
			return
		}

		tokenString := parts[1]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return m.jwtSecret, nil
		})

		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, "Invalid or expired token: "+err.Error())
			return
		}

		if !token.Valid {
			utils.RespondWithError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			utils.RespondWithError(w, http.StatusUnauthorized, "Invalid token claims")
			return
		}

		// Extract user and role ID from claims
		userIDHex, ok := claims["user_id"].(string)
		if !ok {
			utils.RespondWithError(w, http.StatusUnauthorized, "User ID claim missing or invalid")
			return
		}
		roleIDHex, ok := claims["role_id"].(string)
		if !ok {
			utils.RespondWithError(w, http.StatusUnauthorized, "Role ID claim missing or invalid")
			return
		}

		userID, err := primitive.ObjectIDFromHex(userIDHex)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Invalid user ID format in token")
			return
		}
		roleID, err := primitive.ObjectIDFromHex(roleIDHex)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Invalid role ID format in token")
			return
		}

		// Corrected: Use m.authService.AuthenticatedUserContext to get the AuthContext
		authContext, err := m.authService.AuthenticatedUserContext(userID, roleID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve user authentication context: "+err.Error())
			return
		}

		// Check if a specific permission is required for the route
		if requiredPermission != "" && !authContext.HasPermission(requiredPermission) {
			utils.RespondWithError(w, http.StatusForbidden, "You do not have sufficient permissions to access this resource")
			return
		}

		// Add AuthContext to the request context
		ctx := context.WithValue(r.Context(), ContextKeyAuthContext, authContext)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// GetAuthContext retrieves the AuthContext from the request's context
func GetAuthContext(r *http.Request) (*models.AuthContext, error) {
	val := r.Context().Value(ContextKeyAuthContext)
	authContext, ok := val.(*models.AuthContext)
	if !ok || authContext == nil {
		return nil, fmt.Errorf("authentication context not found or invalid in request")
	}
	return authContext, nil
}
