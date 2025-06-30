package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/OsGift/taskflow-api/internal/middleware"
	"github.com/OsGift/taskflow-api/internal/models"
	"github.com/OsGift/taskflow-api/internal/services"
	"github.com/OsGift/taskflow-api/internal/utils"
)

// TaskHandler handles task related HTTP requests
type TaskHandler struct {
	taskService *services.TaskService
	validator   *validator.Validate
}

// NewTaskHandler creates a new TaskHandler
func NewTaskHandler(ts *services.TaskService) *TaskHandler {
	return &TaskHandler{
		taskService: ts,
		validator:   validator.New(),
	}
}

// CreateTask handles creating a new task
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTaskRequest
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

	// Set default status if not provided
	if req.Status == "" {
		req.Status = string(models.StatusTodo)
	}

	task := &models.Task{
		Title:       req.Title,
		Description: req.Description,
		Status:      models.TaskStatus(req.Status),
		UserID:      authContext.UserID, // Assign task to the authenticated user
	}

	createdTask, err := h.taskService.CreateTask(task)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create task")
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, createdTask)
}

// GetTasks handles listing tasks with search, filter, and pagination
func (h *TaskHandler) GetTasks(w http.ResponseWriter, r *http.Request) {
	authContext, err := middleware.GetAuthContext(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

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

	// Filtering parameters
	statusFilter := r.URL.Query().Get("status")
	targetUserIDParam := r.URL.Query().Get("user_id") // For admins to filter by user

	filter := primitive.M{}

	// Determine if user has 'task:read_all' permission
	hasReadAllPermission := authContext.HasPermission("task:read_all")

	// If not admin, restrict to own tasks only
	if !hasReadAllPermission {
		filter["user_id"] = authContext.UserID
	} else {
		// If admin and a user_id query param is provided, filter by that user
		if targetUserIDParam != "" {
			objUserID, err := primitive.ObjectIDFromHex(targetUserIDParam)
			if err != nil {
				utils.RespondWithError(w, http.StatusBadRequest, "Invalid user_id filter format")
				return
			}
			filter["user_id"] = objUserID
		}
	}

	if statusFilter != "" {
		switch strings.ToLower(statusFilter) {
		case "todo", "in_progress", "done":
			filter["status"] = models.TaskStatus(strings.ToLower(statusFilter))
		default:
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid status filter. Must be 'todo', 'in_progress', or 'done'.")
			return
		}
	}

	// Search parameter
	searchQuery := r.URL.Query().Get("search")

	tasksResponse, err := h.taskService.ListTasks(filter, searchQuery, page, limit)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve tasks")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, tasksResponse)
}

// GetTaskByID handles retrieving a single task by ID
func (h *TaskHandler) GetTaskByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	authContext, err := middleware.GetAuthContext(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	task, err := h.taskService.GetTaskByID(taskID)
	if err != nil {
		if err.Error() == "task not found" {
			utils.RespondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve task")
		return
	}

	// Authorization check: 'task:read_all' or owner
	if !authContext.HasPermission("task:read_all") && task.UserID != authContext.UserID {
		utils.RespondWithError(w, http.StatusForbidden, "You do not have permission to view this task")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, task)
}

// UpdateTask handles updating an existing task
func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	var req models.UpdateTaskRequest
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

	task, err := h.taskService.GetTaskByID(taskID)
	if err != nil {
		if err.Error() == "task not found" {
			utils.RespondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve task for update")
		return
	}

	// Authorization check: 'task:update_all' or owner
	if !authContext.HasPermission("task:update_all") && task.UserID != authContext.UserID {
		utils.RespondWithError(w, http.StatusForbidden, "You do not have permission to update this task")
		return
	}

	updatedTask, err := h.taskService.UpdateTask(taskID, &req)
	if err != nil {
		if err.Error() == "task not found or no changes made" {
			utils.RespondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update task")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, updatedTask)
}

// DeleteTask handles deleting a task
func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	authContext, err := middleware.GetAuthContext(r)
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	task, err := h.taskService.GetTaskByID(taskID)
	if err != nil {
		if err.Error() == "task not found" {
			utils.RespondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve task for deletion check")
		return
	}

	// Authorization check: 'task:delete_all' or owner
	if !authContext.HasPermission("task:delete_all") && task.UserID != authContext.UserID {
		utils.RespondWithError(w, http.StatusForbidden, "You do not have permission to delete this task")
		return
	}

	err = h.taskService.DeleteTask(taskID)
	if err != nil {
		if err.Error() == "task not found" {
			utils.RespondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete task")
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204 No Content for successful deletion
}
