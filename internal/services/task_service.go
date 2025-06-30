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

// TaskService provides methods for task-related operations
type TaskService struct {
	tasksCollection *mongo.Collection
}

// NewTaskService creates a new TaskService
func NewTaskService(db *mongo.Database) *TaskService {
	return &TaskService{
		tasksCollection: db.Collection("tasks"),
	}
}

// CreateTask creates a new task
func (s *TaskService) CreateTask(task *models.Task) (*models.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	task.ID = primitive.NewObjectID()
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

	_, err := s.tasksCollection.InsertOne(ctx, task)
	if err != nil {
		return nil, err
	}
	return task, nil
}

// GetTaskByID retrieves a task by its ID
func (s *TaskService) GetTaskByID(id string) (*models.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid task ID format")
	}

	var task models.Task
	err = s.tasksCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&task)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("task not found")
		}
		return nil, err
	}
	return &task, nil
}

// ListTasks retrieves a list of tasks with optional filtering, search, and pagination
func (s *TaskService) ListTasks(
	filter primitive.M,
	searchQuery string,
	page int64,
	limit int64,
) (*models.TaskListResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Build the query filter
	query := bson.M{}
	for k, v := range filter {
		query[k] = v
	}

	// Add search query if provided (case-insensitive regex on title and description)
	if searchQuery != "" {
		searchPattern := primitive.Regex{Pattern: searchQuery, Options: "i"} // "i" for case-insensitive
		query["$or"] = []bson.M{
			{"title": searchPattern},
			{"description": searchPattern},
		}
	}

	// Calculate skip for pagination
	skip := (page - 1) * limit
	if skip < 0 {
		skip = 0 // Ensure skip is not negative
	}

	findOptions := options.Find()
	findOptions.SetSkip(skip)
	findOptions.SetLimit(limit)
	findOptions.SetSort(bson.D{{"created_at", -1}}) // Sort by creation date descending

	cursor, err := s.tasksCollection.Find(ctx, query, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tasks []models.Task
	if err = cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}

	// Get total count for pagination metadata
	totalCount, err := s.tasksCollection.CountDocuments(ctx, query)
	if err != nil {
		return nil, err
	}

	return &models.TaskListResponse{
		Tasks:      tasks,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
	}, nil
}

// UpdateTask updates an existing task
func (s *TaskService) UpdateTask(id string, update *models.UpdateTaskRequest) (*models.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid task ID format")
	}

	updateDoc := bson.M{"$set": bson.M{"updated_at": time.Now()}}
	if update.Title != nil {
		updateDoc["$set"].(bson.M)["title"] = *update.Title
	}
	if update.Description != nil {
		updateDoc["$set"].(bson.M)["description"] = *update.Description
	}
	if update.Status != nil {
		updateDoc["$set"].(bson.M)["status"] = models.TaskStatus(*update.Status)
	}

	res, err := s.tasksCollection.UpdateByID(ctx, objID, updateDoc)
	if err != nil {
		return nil, err
	}
	if res.ModifiedCount == 0 {
		return nil, errors.New("task not found or no changes made")
	}

	updatedTask, err := s.GetTaskByID(id)
	if err != nil {
		return nil, err // Task should exist, this would be an unexpected error
	}
	return updatedTask, nil
}

// DeleteTask deletes a task by its ID
func (s *TaskService) DeleteTask(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid task ID format")
	}

	res, err := s.tasksCollection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return errors.New("task not found")
	}
	return nil
}
