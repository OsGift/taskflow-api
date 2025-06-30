package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Permission represents a specific action a user can perform
type Permission struct {
	Action string `bson:"action" json:"action"` // e.g., "task:create", "task:read_all"
}

// Role represents a user role with a set of permissions
type Role struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name        string             `bson:"name" json:"name" validate:"required"` // e.g., "Admin", "User", "Manager"
	Permissions []Permission       `bson:"permissions" json:"permissions"`
}

// Define some default roles and their permissions (for seeding)
var DefaultRoles = []Role{
	{
		Name: "Admin",
		Permissions: []Permission{
			{Action: "task:create"}, {Action: "task:read_all"}, {Action: "task:update_all"}, {Action: "task:delete_all"},
			{Action: "user:read_all"}, {Action: "user:update_role"}, {Action: "user:update_profile"}, {Action: "user:verify_email"},
			{Action: "user:create_admin"}, // Permission for an Admin to add another Admin
			{Action: "dashboard:read_metrics"}, // Access to dashboard metrics
		},
	},
	{
		Name: "Manager",
		Permissions: []Permission{
			{Action: "task:create"}, {Action: "task:read_all"}, {Action: "task:update_all"}, {Action: "task:delete_all"},
			{Action: "user:update_profile"},
		},
	},
	{
		Name: "User",
		Permissions: []Permission{
			{Action: "task:create"}, {Action: "task:read_own"}, {Action: "task:update_own"}, {Action: "task:delete_own"},
			{Action: "user:update_profile"}, // Users can update their own profile
		},
	},
}
