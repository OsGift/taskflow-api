package services

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/OsGift/taskflow-api/internal/models"
)

// UserService provides methods for user and role related operations
type UserService struct {
	usersCollection *mongo.Collection
	rolesCollection *mongo.Collection
}

// NewUserService creates a new UserService
func NewUserService(db *mongo.Database) *UserService {
	return &UserService{
		usersCollection: db.Collection("users"),
		rolesCollection: db.Collection("roles"),
	}
}

// CreateUser creates a new user in the database
func (s *UserService) CreateUser(user *models.User) (*models.UserResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user.ID = primitive.NewObjectID()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	// Ensure default values are set for new fields if not already by handler/service
	if user.FirstName == "" {
		user.FirstName = "New"
	}
	if user.LastName == "" {
		user.LastName = "User"
	}
	if user.ProfilePictureURL == "" {
		user.ProfilePictureURL = "https://placehold.co/150x150/cccccc/ffffff?text=Avatar"
	} // Default avatar
	// IsEmailVerified and NeedsPasswordChange are set by the caller (AuthService)

	_, err := s.usersCollection.InsertOne(ctx, user)
	if err != nil {
		return nil, err
	}

	role, err := s.GetRoleByID(user.RoleID.Hex())
	if err != nil {
		return nil, errors.New("failed to retrieve role for new user")
	}

	return &models.UserResponse{
		ID:                  user.ID.Hex(),
		FirstName:           user.FirstName,
		LastName:            user.LastName,
		Email:               user.Email,
		RoleName:            role.Name,
		ProfilePictureURL:   user.ProfilePictureURL,
		IsEmailVerified:     user.IsEmailVerified,
		NeedsPasswordChange: user.NeedsPasswordChange,
		CreatedAt:           user.CreatedAt,
		UpdatedAt:           user.UpdatedAt,
	}, nil
}

// GetUserByID retrieves a user by their ID
func (s *UserService) GetUserByID(id string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid user ID format")
	}

	var user models.User
	err = s.usersCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by their email address
func (s *UserService) GetUserByEmail(email string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err := s.usersCollection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// GetRoleByName retrieves a role by its name
func (s *UserService) GetRoleByName(name string) (*models.Role, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var role models.Role
	err := s.rolesCollection.FindOne(ctx, bson.M{"name": name}).Decode(&role)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("role not found")
		}
		return nil, err
	}
	return &role, nil
}

// GetRoleByID retrieves a role by its ID
func (s *UserService) GetRoleByID(id string) (*models.Role, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid role ID format")
	}

	var role models.Role
	err = s.rolesCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&role)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("role not found")
		}
		return nil, err
	}
	return &role, nil
}

// UpdateUserPassword updates a user's password
func (s *UserService) UpdateUserPassword(userID primitive.ObjectID, hashedPassword string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": bson.M{
		"password":   hashedPassword,
		"updated_at": time.Now(),
	}}
	result, err := s.usersCollection.UpdateByID(ctx, userID, update)
	if err != nil {
		return err
	}
	if result.ModifiedCount == 0 {
		return errors.New("user not found or password not changed")
	}
	return nil
}

// UpdateUserPasswordAndNeedsChange updates a user's password and sets needs_password_change flag
func (s *UserService) UpdateUserPasswordAndNeedsChange(userID primitive.ObjectID, hashedPassword string, needsChange bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": bson.M{
		"password":              hashedPassword,
		"needs_password_change": needsChange,
		"updated_at":            time.Now(),
	}}
	result, err := s.usersCollection.UpdateByID(ctx, userID, update)
	if err != nil {
		return err
	}
	if result.ModifiedCount == 0 {
		return errors.New("user not found or password/needs_password_change not updated")
	}
	return nil
}

// UpdateUserRole updates a user's role
func (s *UserService) UpdateUserRole(userID string, newRoleName string) (*models.UserResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID format")
	}

	newRole, err := s.GetRoleByName(newRoleName)
	if err != nil {
		return nil, errors.New("new role not found")
	}

	update := bson.M{
		"$set": bson.M{
			"role_id":    newRole.ID,
			"updated_at": time.Now(),
		},
	}

	result, err := s.usersCollection.UpdateByID(ctx, objID, update)
	if err != nil {
		return nil, err
	}
	if result.ModifiedCount == 0 {
		return nil, errors.New("user not found or role not changed")
	}

	updatedUser, err := s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	return s.GetUserResponseByID(updatedUser.ID.Hex()) // Use the helper to build response
}

// UpdateUserProfile updates a user's profile details (first_name, last_name, profile_picture_url)
func (s *UserService) UpdateUserProfile(userID string, req *models.UpdateUserProfileRequest) (*models.UserResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID format")
	}

	updateDoc := bson.M{"$set": bson.M{"updated_at": time.Now()}}
	if req.FirstName != nil {
		updateDoc["$set"].(bson.M)["first_name"] = *req.FirstName
	}
	if req.LastName != nil {
		updateDoc["$set"].(bson.M)["last_name"] = *req.LastName
	}
	if req.ProfilePictureURL != nil {
		updateDoc["$set"].(bson.M)["profile_picture_url"] = *req.ProfilePictureURL
	}

	res, err := s.usersCollection.UpdateByID(ctx, objID, updateDoc)
	if err != nil {
		return nil, err
	}
	if res.ModifiedCount == 0 {
		return nil, errors.New("user not found or no changes made to profile")
	}

	return s.GetUserResponseByID(userID) // Use the helper to build response
}

// VerifyUserEmail sets a user's email_verified status to true
func (s *UserService) VerifyUserEmail(userID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": bson.M{
		"is_email_verified": true,
		"updated_at":        time.Now(),
	}}
	result, err := s.usersCollection.UpdateByID(ctx, userID, update)
	if err != nil {
		return err
	}
	if result.ModifiedCount == 0 {
		return errors.New("user not found or email already verified")
	}
	return nil
}

// GetUserResponseByID populates UserResponse with role name (used in handlers)
func (s *UserService) GetUserResponseByID(id string) (*models.UserResponse, error) {
	user, err := s.GetUserByID(id)
	if err != nil {
		return nil, err
	}

	role, err := s.GetRoleByID(user.RoleID.Hex())
	if err != nil {
		// If role not found, might imply corrupted data; handle gracefully
		return &models.UserResponse{
			ID:                  user.ID.Hex(),
			FirstName:           user.FirstName,
			LastName:            user.LastName,
			Email:               user.Email,
			RoleName:            "Unknown", // Default to unknown role
			ProfilePictureURL:   user.ProfilePictureURL,
			IsEmailVerified:     user.IsEmailVerified,
			NeedsPasswordChange: user.NeedsPasswordChange,
			CreatedAt:           user.CreatedAt,
			UpdatedAt:           user.UpdatedAt,
		}, nil
	}

	return &models.UserResponse{
		ID:                  user.ID.Hex(),
		FirstName:           user.FirstName,
		LastName:            user.LastName,
		Email:               user.Email,
		RoleName:            role.Name,
		ProfilePictureURL:   user.ProfilePictureURL,
		IsEmailVerified:     user.IsEmailVerified,
		NeedsPasswordChange: user.NeedsPasswordChange,
		CreatedAt:           user.CreatedAt,
		UpdatedAt:           user.UpdatedAt,
	}, nil
}

// ListUsers retrieves a list of users with optional filtering and pagination
func (s *UserService) ListUsers(
	filter primitive.M,
	page int64,
	limit int64,
) (*models.UserListResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Calculate skip for pagination
	skip := (page - 1) * limit
	if skip < 0 {
		skip = 0 // Ensure skip is not negative
	}

	findOptions := options.Find()
	findOptions.SetSkip(skip)
	findOptions.SetLimit(limit)
	findOptions.SetSort(bson.D{{"created_at", -1}}) // Sort by creation date descending

	cursor, err := s.usersCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, err
	}

	userResponses := make([]models.UserResponse, len(users))
	for i, user := range users {
		role, roleErr := s.GetRoleByID(user.RoleID.Hex())
		roleName := "Unknown"
		if roleErr == nil {
			roleName = role.Name
		}
		userResponses[i] = models.UserResponse{
			ID:                  user.ID.Hex(),
			FirstName:           user.FirstName,
			LastName:            user.LastName,
			Email:               user.Email,
			RoleName:            roleName,
			ProfilePictureURL:   user.ProfilePictureURL,
			IsEmailVerified:     user.IsEmailVerified,
			NeedsPasswordChange: user.NeedsPasswordChange,
			CreatedAt:           user.CreatedAt,
			UpdatedAt:           user.UpdatedAt,
		}
	}

	// Get total count for pagination metadata
	totalCount, err := s.usersCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &models.UserListResponse{
		Users:      userResponses,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
	}, nil
}

func (s *UserService) GetAuthContext(userID, roleID primitive.ObjectID) (*models.AuthContext, error) {
	user, err := s.GetUserByID(userID.Hex())
	if err != nil {
		return nil, err
	}

	role, err := s.GetRoleByID(user.RoleID.Hex())
	if err != nil {
		return nil, err
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
