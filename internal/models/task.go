package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	StatusTodo       TaskStatus = "todo"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
)

// Task represents a single task item
type Task struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Title       string             `bson:"title" json:"title" validate:"required,min=5"`
	Description string             `bson:"description" json:"description"`
	Status      TaskStatus         `bson:"status" json:"status" validate:"required,oneof=todo in_progress done"`
	UserID      primitive.ObjectID `bson:"user_id" json:"user_id"` // Owner of the task
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// CreateTaskRequest is for creating a new task
type CreateTaskRequest struct {
	Title       string `json:"title" validate:"required,min=5"`
	Description string `json:"description"`
	Status      string `json:"status" validate:"omitempty,oneof=todo in_progress done"`
}

// UpdateTaskRequest is for updating an existing task
type UpdateTaskRequest struct {
	Title       *string `json:"title,omitempty" validate:"omitempty,min=5"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty" validate:"omitempty,oneof=todo in_progress done"`
}

// TaskListResponse holds tasks and pagination metadata
type TaskListResponse struct {
	Tasks      []Task `json:"tasks"`
	TotalCount int64  `json:"total_count"`
	Page       int64  `json:"page"`
	Limit      int64  `json:"limit"`
}
