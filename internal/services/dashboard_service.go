package services

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/OsGift/taskflow-api/internal/models"
)

// DashboardService provides methods for fetching application-wide metrics
type DashboardService struct {
	usersCollection *mongo.Collection
	tasksCollection *mongo.Collection
	rolesCollection *mongo.Collection
}

// NewDashboardService creates a new DashboardService
func NewDashboardService(db *mongo.Database) *DashboardService {
	return &DashboardService{
		usersCollection: db.Collection("users"),
		tasksCollection: db.Collection("tasks"),
		rolesCollection: db.Collection("roles"),
	}
}

// GetDashboardMetrics fetches various metrics based on the specified time period or custom range
func (s *DashboardService) GetDashboardMetrics(
	period models.DashboardPeriod,
	startDate, endDate *time.Time,
) (*models.DashboardMetricsResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	metrics := &models.DashboardMetricsResponse{
		Period: period,
	}

	// 1. Get total counts (always relevant)
	totalUsers, err := s.usersCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	metrics.TotalUsers = totalUsers

	totalTasks, err := s.tasksCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	metrics.TotalTasks = totalTasks

	// 2. Get counts by role
	adminRole, _ := s.rolesCollection.FindOne(ctx, bson.M{"name": "Admin"}).DecodeBytes()
	if adminRole != nil {
		metrics.AdminsCount, _ = s.usersCollection.CountDocuments(ctx, bson.M{"role_id": adminRole.Lookup("_id").ObjectID()})
	}
	managerRole, _ := s.rolesCollection.FindOne(ctx, bson.M{"name": "Manager"}).DecodeBytes()
	if managerRole != nil {
		metrics.ManagersCount, _ = s.usersCollection.CountDocuments(ctx, bson.M{"role_id": managerRole.Lookup("_id").ObjectID()})
	}
	userRole, _ := s.rolesCollection.FindOne(ctx, bson.M{"name": "User"}).DecodeBytes()
	if userRole != nil {
		metrics.RegularUsersCount, _ = s.usersCollection.CountDocuments(ctx, bson.M{"role_id": userRole.Lookup("_id").ObjectID()})
	}

	// 3. Define date range for "new" counts and filtering
	var periodFilter bson.M
	if period == models.PeriodCustom && startDate != nil && endDate != nil {
		periodFilter = bson.M{
			"created_at": bson.M{
				"$gte": *startDate,
				"$lte": *endDate,
			},
		}
		metrics.StartDate = startDate
		metrics.EndDate = endDate
	} else if period != models.PeriodCustom {
		// Calculate dynamic start/end dates based on period
		now := time.Now()
		var start time.Time
		switch period {
		case models.PeriodDaily:
			start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			endDate = &now // End date is now
		case models.PeriodWeekly:
			weekday := time.Duration(now.Weekday())
			if weekday == 0 { // Sunday
				weekday = 7
			}
			start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Add(-((weekday - 1) * 24 * time.Hour))
			endDate = &now
		case models.PeriodMonthly:
			start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
			endDate = &now
		}
		periodFilter = bson.M{
			"created_at": bson.M{
				"$gte": start,
				"$lte": *endDate,
			},
		}
		metrics.StartDate = &start
		metrics.EndDate = endDate
	}

	// 4. Get new users/tasks within the specified period
	if periodFilter != nil {
		newUsers, err := s.usersCollection.CountDocuments(ctx, periodFilter)
		if err != nil {
			return nil, err
		}
		metrics.NewUsers = newUsers

		newTasks, err := s.tasksCollection.CountDocuments(ctx, periodFilter)
		if err != nil {
			return nil, err
		}
		metrics.NewTasks = newTasks
	} else {
		// If no periodFilter (e.g., initial load without specific date filters), new users/tasks count is 0
		metrics.NewUsers = 0
		metrics.NewTasks = 0
	}

	// 5. Get tasks by status
	// Use aggregation pipeline for tasks by status
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$status"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "status", Value: "$_id"},
			{Key: "count", Value: 1},
			{Key: "_id", Value: 0},
		}}},
	}

	// Add period filter to aggregation if applicable
	if periodFilter != nil {
		pipeline = append(mongo.Pipeline{bson.D{{Key: "$match", Value: periodFilter}}}, pipeline...)
	}

	cursor, err := s.tasksCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var taskStatusCounts []models.TaskStatusCount
	if err = cursor.All(ctx, &taskStatusCounts); err != nil {
		return nil, err
	}
	metrics.TasksByStatus = taskStatusCounts

	return metrics, nil
}
