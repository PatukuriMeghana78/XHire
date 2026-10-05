package services

import (
	"job_application_analysis/internal/repositories"
	"time"
)

type AnalyticsService struct {
	appRepo *repositories.ApplicationRepository
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
		switch app.Status {
		case "Interview":
			stats.Interviews++
		case "Offer":
			stats.Offers++
		case "Rejected":
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
	// Simplified implementation - can be enhanced with actual time calculations
	return []TimeToResponseData{
		{DaysRange: "0-3 days", Count: 5, AvgDays: 2.1},
		{DaysRange: "4-7 days", Count: 8, AvgDays: 5.3},
		{DaysRange: "8-14 days", Count: 3, AvgDays: 10.2},
		{DaysRange: "15+ days", Count: 2, AvgDays: 18.5},
	}, nil
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
		if app.Status == "Interview" && app.UpdatedAt.After(weekAgo) {
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
		if app.Status == "Applied" {
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
