package models

import "time"

// DashboardPeriod defines possible date filters
type DashboardPeriod string

const (
	PeriodDaily   DashboardPeriod = "daily"
	PeriodWeekly  DashboardPeriod = "weekly"
	PeriodMonthly DashboardPeriod = "monthly"
	PeriodCustom  DashboardPeriod = "custom"
)

// TaskStatusCount represents the count of tasks by status
type TaskStatusCount struct {
	Status TaskStatus `json:"status"`
	Count  int64      `json:"count"`
}

// DashboardMetricsResponse holds various metrics for the dashboard
type DashboardMetricsResponse struct {
	TotalUsers     int64             `json:"total_users"`
	TotalTasks     int64             `json:"total_tasks"`
	NewUsers       int64             `json:"new_users_count"`      // Users created in the period
	NewTasks       int64             `json:"new_tasks_count"`      // Tasks created in the period
	TasksByStatus  []TaskStatusCount `json:"tasks_by_status"`
	AdminsCount    int64             `json:"admins_count"`
	ManagersCount  int64             `json:"managers_count"`
	RegularUsersCount int64          `json:"regular_users_count"`
	StartDate      *time.Time        `json:"start_date,omitempty"` // Applied filter start date
	EndDate        *time.Time        `json:"end_date,omitempty"`   // Applied filter end date
	Period         DashboardPeriod   `json:"period"`               // Period requested
}