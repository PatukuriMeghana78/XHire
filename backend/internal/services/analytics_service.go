package services

import (
	"math"
	"strings"
	"time"

	"job_application_analysis/internal/models"
	"job_application_analysis/internal/repositories"
)

// applicationReader is the slice of the application repository analytics needs.
// Declared as an interface so time calculations can be unit-tested; the project builds
// with CGO_ENABLED=0, so SQLite-backed throwaway databases are not available.
type applicationReader interface {
	FindByUserID(userID uint) ([]models.Application, error)
}

type AnalyticsService struct {
	appRepo applicationReader
}

func NewAnalyticsService(appRepo *repositories.ApplicationRepository) *AnalyticsService {
	return &AnalyticsService{appRepo: appRepo}
}

type DashboardStats struct {
	TotalApplications int     `json:"total_applications"`
	Interviews        int     `json:"interviews"`
	Offers            int     `json:"offers"`
	Rejected          int     `json:"rejected"`
	ConversionRate    float64 `json:"conversion_rate"`
}

type StatusDistribution struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type TimeToResponseData struct {
	DaysRange string  `json:"days_range"`
	Count     int     `json:"count"`
	AvgDays   float64 `json:"avg_days"`
}

type WeeklySummary struct {
	ApplicationsThisWeek int      `json:"applications_this_week"`
	InterviewsThisWeek   int      `json:"interviews_this_week"`
	SuggestedActions     []string `json:"suggested_actions"`
}

func (s *AnalyticsService) GetDashboardStats(userID uint) (*DashboardStats, error) {
	apps, err := s.appRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	stats := &DashboardStats{
		TotalApplications: len(apps),
	}

	for _, app := range apps {
		// Statuses reach the database from two generations of clients: the
		// legacy email flow writes "Interview"/"Offer", the XHire UI writes the
		// normalised lowercase form. Match case-insensitively so neither is
		// silently counted as zero.
		switch {
		case strings.EqualFold(app.Status, "Interview"):
			stats.Interviews++
		case strings.EqualFold(app.Status, "Offer"):
			stats.Offers++
		case strings.EqualFold(app.Status, "Rejected"):
			stats.Rejected++
		}
	}

	if stats.TotalApplications > 0 {
		stats.ConversionRate = float64(stats.Offers) / float64(stats.TotalApplications) * 100
	}

	return stats, nil
}

func (s *AnalyticsService) GetStatusDistribution(userID uint) ([]StatusDistribution, error) {
	apps, err := s.appRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	distribution := make(map[string]int)
	for _, app := range apps {
		distribution[app.Status]++
	}

	result := []StatusDistribution{}
	for status, count := range distribution {
		result = append(result, StatusDistribution{
			Status: status,
			Count:  count,
		})
	}

	return result, nil
}

func (s *AnalyticsService) GetTimeToResponse(userID uint) ([]TimeToResponseData, error) {
	apps, err := s.appRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	return bucketResponseDays(responseDays(apps)), nil
}

// responseStatuses are the statuses that represent a reply from the employer.
// "Applied" is the baseline the candidate created, so it is never a response.
var responseStatuses = map[string]bool{
	"interview": true,
	"offer":     true,
	"hired":     true,
	"rejected":  true,
}

func isResponseStatus(status string) bool {
	return responseStatuses[strings.ToLower(strings.TrimSpace(status))]
}

// responseDays returns the elapsed days between applying and the first recorded
// employer response, for every application where both timestamps actually exist.
// Applications still pending, with no recorded dates, or whose dates contradict each
// other are left out rather than estimated.
func responseDays(apps []models.Application) []float64 {
	out := make([]float64, 0, len(apps))

	for _, app := range apps {
		applied := app.AppliedDate
		if applied.IsZero() {
			applied = app.CreatedAt
		}
		response := firstResponseAt(app)
		if applied.IsZero() || response.IsZero() || !response.After(applied) {
			continue
		}
		out = append(out, response.Sub(applied).Hours()/24)
	}

	return out
}

// firstResponseAt prefers the earliest matching status-history entry, which is written
// on every status change. It falls back to the application's own update time because
// history rows were originally written best-effort, so older records may have none.
func firstResponseAt(app models.Application) time.Time {
	earliest := time.Time{}
	for _, history := range app.StatusHistory {
		if !isResponseStatus(history.Status) || history.ChangedAt.IsZero() {
			continue
		}
		if earliest.IsZero() || history.ChangedAt.Before(earliest) {
			earliest = history.ChangedAt
		}
	}
	if !earliest.IsZero() {
		return earliest
	}

	if isResponseStatus(app.Status) {
		if !app.LastUpdated.IsZero() {
			return app.LastUpdated
		}
		return app.UpdatedAt
	}

	return time.Time{}
}

// timeToResponseLabels are ordered low to high; a response is placed in the first
// bucket whose upper bound it does not exceed, so 3.0 days is "0-3 days" and
// 3.9 days is "4-7 days".
var timeToResponseLabels = []string{"0-3 days", "4-7 days", "8-14 days", "15+ days"}

func bucketIndexFor(day float64) int {
	switch {
	case day <= 3:
		return 0
	case day <= 7:
		return 1
	case day <= 14:
		return 2
	default:
		return 3
	}
}

func bucketResponseDays(days []float64) []TimeToResponseData {
	counts := make([]int, len(timeToResponseLabels))
	sums := make([]float64, len(timeToResponseLabels))

	for _, day := range days {
		index := bucketIndexFor(day)
		counts[index]++
		sums[index] += day
	}

	results := make([]TimeToResponseData, 0, len(timeToResponseLabels))
	for index, label := range timeToResponseLabels {
		if counts[index] == 0 {
			continue
		}
		results = append(results, TimeToResponseData{
			DaysRange: label,
			Count:     counts[index],
			AvgDays:   math.Round(sums[index]/float64(counts[index])*10) / 10,
		})
	}

	return results
}

func (s *AnalyticsService) GetWeeklySummary(userID uint) (*WeeklySummary, error) {
	apps, err := s.appRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	summary := &WeeklySummary{
		ApplicationsThisWeek: 0,
		InterviewsThisWeek:   0,
		SuggestedActions:     []string{},
	}

	// Calculate applications and interviews from the last 7 days
	now := time.Now()
	weekAgo := now.AddDate(0, 0, -7)

	for _, app := range apps {
		if app.AppliedDate.After(weekAgo) {
			summary.ApplicationsThisWeek++
		}
		if strings.EqualFold(app.Status, "Interview") && app.UpdatedAt.After(weekAgo) {
			summary.InterviewsThisWeek++
		}
	}

	// Generate suggested actions
	if summary.ApplicationsThisWeek == 0 {
		summary.SuggestedActions = append(summary.SuggestedActions, "Apply to at least 5 jobs this week")
	}
	if summary.ApplicationsThisWeek < 5 {
		summary.SuggestedActions = append(summary.SuggestedActions, "Increase your application rate to improve chances")
	}

	pendingCount := 0
	for _, app := range apps {
		if strings.EqualFold(app.Status, "Applied") {
			pendingCount++
		}
	}
	if pendingCount > 0 {
		summary.SuggestedActions = append(summary.SuggestedActions, "Follow up on pending applications")
	}

	if summary.InterviewsThisWeek > 0 {
		summary.SuggestedActions = append(summary.SuggestedActions, "Prepare for upcoming interviews")
	}

	if len(summary.SuggestedActions) == 0 {
		summary.SuggestedActions = append(summary.SuggestedActions, "Keep up the great work!")
	}

	return summary, nil
}
