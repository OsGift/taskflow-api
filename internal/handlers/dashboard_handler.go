package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"

	"github.com/OsGift/taskflow-api/internal/models"
	"github.com/OsGift/taskflow-api/internal/services"
	"github.com/OsGift/taskflow-api/internal/utils"
)

// DashboardHandler handles dashboard related HTTP requests
type DashboardHandler struct {
	dashboardService *services.DashboardService
	validator        *validator.Validate
}

// NewDashboardHandler creates a new DashboardHandler
func NewDashboardHandler(ds *services.DashboardService) *DashboardHandler {
	return &DashboardHandler{
		dashboardService: ds,
		validator:        validator.New(),
	}
}

// GetDashboardMetrics handles fetching various dashboard metrics
func (h *DashboardHandler) GetDashboardMetrics(w http.ResponseWriter, r *http.Request) {
	// Permission 'dashboard:read_metrics' is checked by middleware

	periodStr := r.URL.Query().Get("period")
	if periodStr == "" {
		periodStr = string(models.PeriodMonthly) // Default to monthly if not specified
	}

	period := models.DashboardPeriod(strings.ToLower(periodStr))

	var startDate, endDate *time.Time
	if period == models.PeriodCustom {
		startStr := r.URL.Query().Get("start_date")
		endStr := r.URL.Query().Get("end_date")

		if startStr == "" || endStr == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "start_date and end_date are required for custom period")
			return
		}

		parsedStartDate, err := time.Parse("2006-01-02", startStr) // YYYY-MM-DD
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid start_date format. Use YYYY-MM-DD.")
			return
		}
		parsedEndDate, err := time.Parse("2006-01-02", endStr) // YYYY-MM-DD
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid end_date format. Use YYYY-MM-DD.")
			return
		}
		// Set end date to end of the day for proper range
		parsedEndDate = parsedEndDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

		startDate = &parsedStartDate
		endDate = &parsedEndDate

		if startDate.After(*endDate) {
			utils.RespondWithError(w, http.StatusBadRequest, "start_date cannot be after end_date")
			return
		}
	} else if period != models.PeriodDaily && period != models.PeriodWeekly && period != models.PeriodMonthly {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid period. Must be 'daily', 'weekly', 'monthly', or 'custom'.")
		return
	}

	metrics, err := h.dashboardService.GetDashboardMetrics(period, startDate, endDate)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve dashboard metrics")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, metrics)
}
