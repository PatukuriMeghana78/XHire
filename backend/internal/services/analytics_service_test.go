package services

import (
	"errors"
	"testing"
	"time"

	"job_application_analysis/internal/models"

	"gorm.io/gorm"
)

type fakeApplicationReader struct {
	apps []models.Application
	err  error
	seen []uint
}

func (f *fakeApplicationReader) FindByUserID(userID uint) ([]models.Application, error) {
	f.seen = append(f.seen, userID)
	return f.apps, f.err
}

func TestGetTimeToResponse_UsesStatusHistory(t *testing.T) {
	applied := time.Now().UTC().AddDate(0, 0, -40)
	apps := []models.Application{
		{
			UserID:      1,
			AppliedDate: applied,
			Status:      "Rejected",
			StatusHistory: []models.StatusHistory{
				{Status: "Applied", ChangedAt: applied, Source: "Manual"},
				{Status: "Interview", ChangedAt: applied.AddDate(0, 0, 3), Source: "Manual"},
				{Status: "Rejected", ChangedAt: applied.AddDate(0, 0, 6), Source: "Email"},
			},
		},
	}
	svc := &AnalyticsService{appRepo: &fakeApplicationReader{apps: apps}}

	data, err := svc.GetTimeToResponse(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 1 {
		t.Fatalf("expected one bucket holding the first response only, got %+v", data)
	}
	// 3.0 days lands in 0-3; the later rejection must not be counted twice.
	if data[0].DaysRange != "0-3 days" || data[0].Count != 1 || data[0].AvgDays != 3.0 {
		t.Fatalf("unexpected bucket: %+v", data[0])
	}
}

func TestGetTimeToResponse_FallsBackToUpdateTimestamp(t *testing.T) {
	applied := time.Now().UTC().AddDate(0, 0, -30)
	apps := []models.Application{{
		UserID:      1,
		AppliedDate: applied,
		Status:      "Interview",
		LastUpdated: applied.AddDate(0, 0, 9),
	}}
	svc := &AnalyticsService{appRepo: &fakeApplicationReader{apps: apps}}

	data, err := svc.GetTimeToResponse(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 1 || data[0].DaysRange != "8-14 days" || data[0].AvgDays != 9.0 {
		t.Fatalf("expected a 9-day response in 8-14 days, got %+v", data)
	}
}

func TestGetTimeToResponse_SkipsPendingApplications(t *testing.T) {
	applied := time.Now().UTC().AddDate(0, 0, -12)
	apps := []models.Application{{
		UserID:        1,
		AppliedDate:   applied,
		Status:        "Applied",
		LastUpdated:   time.Now().UTC(),
		StatusHistory: []models.StatusHistory{{Status: "Applied", ChangedAt: applied}},
	}}
	svc := &AnalyticsService{appRepo: &fakeApplicationReader{apps: apps}}

	data, err := svc.GetTimeToResponse(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("an application with no employer reply must not be counted, got %+v", data)
	}
}

func TestGetTimeToResponse_EmptyDataset(t *testing.T) {
	svc := &AnalyticsService{appRepo: &fakeApplicationReader{}}

	data, err := svc.GetTimeToResponse(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data == nil {
		t.Fatal("expected an empty slice, not nil")
	}
	if len(data) != 0 {
		t.Fatalf("expected no fabricated buckets, got %+v", data)
	}
}

func TestGetTimeToResponse_PropagatesRepositoryError(t *testing.T) {
	svc := &AnalyticsService{appRepo: &fakeApplicationReader{err: errors.New("db down")}}

	data, err := svc.GetTimeToResponse(1)
	if err == nil {
		t.Fatal("expected the repository error to surface")
	}
	if data != nil {
		t.Fatalf("expected no data on error, got %+v", data)
	}
}

func TestGetTimeToResponse_QueriesRequestedUser(t *testing.T) {
	reader := &fakeApplicationReader{}
	svc := &AnalyticsService{appRepo: reader}

	if _, err := svc.GetTimeToResponse(42); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reader.seen) != 1 || reader.seen[0] != 42 {
		t.Fatalf("expected a lookup scoped to user 42, got %v", reader.seen)
	}
}

func TestGetTimeToResponse_AveragesWithinBucket(t *testing.T) {
	base := time.Now().UTC().AddDate(0, 0, -60)
	apps := []models.Application{
		{AppliedDate: base, Status: "Interview", StatusHistory: []models.StatusHistory{
			{Status: "Interview", ChangedAt: base.Add(24 * time.Hour)},
		}},
		{AppliedDate: base, Status: "Rejected", StatusHistory: []models.StatusHistory{
			{Status: "Rejected", ChangedAt: base.Add(48 * time.Hour)},
		}},
		{AppliedDate: base, Status: "Offer", StatusHistory: []models.StatusHistory{
			{Status: "Offer", ChangedAt: base.AddDate(0, 0, 20)},
		}},
	}
	svc := &AnalyticsService{appRepo: &fakeApplicationReader{apps: apps}}

	data, err := svc.GetTimeToResponse(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 2 {
		t.Fatalf("expected two populated buckets, got %+v", data)
	}
	if data[0].DaysRange != "0-3 days" || data[0].Count != 2 || data[0].AvgDays != 1.5 {
		t.Fatalf("unexpected first bucket: %+v", data[0])
	}
	if data[1].DaysRange != "15+ days" || data[1].Count != 1 || data[1].AvgDays != 20 {
		t.Fatalf("unexpected top bucket: %+v", data[1])
	}
}

func TestResponseDays_ExcludesUnmeasurableRows(t *testing.T) {
	base := time.Now().UTC()

	apps := []models.Application{
		// No applied date at all.
		{Status: "Interview", StatusHistory: []models.StatusHistory{{Status: "Interview", ChangedAt: base}}},
		// Applied date present, response never recorded.
		{AppliedDate: base.AddDate(0, 0, -5), Status: "Applied"},
		// Response recorded before the application date.
		{AppliedDate: base, Status: "Interview", StatusHistory: []models.StatusHistory{
			{Status: "Interview", ChangedAt: base.AddDate(0, 0, -2)},
		}},
		// Measurable: exactly 2.5 days.
		{AppliedDate: base, Status: "Interview", StatusHistory: []models.StatusHistory{
			{Status: "Interview", ChangedAt: base.Add(60 * time.Hour)},
		}},
	}

	got := responseDays(apps)
	if len(got) != 1 {
		t.Fatalf("expected only the measurable application, got %v", got)
	}
	if got[0] != 2.5 {
		t.Fatalf("expected 2.5 days, got %v", got[0])
	}
}

func TestResponseDays_FallsBackToCreatedAt(t *testing.T) {
	created := time.Now().UTC().AddDate(0, 0, -8)
	app := models.Application{Status: "Rejected", StatusHistory: []models.StatusHistory{
		{Status: "Rejected", ChangedAt: created.AddDate(0, 0, 2)},
	}}
	app.CreatedAt = created

	got := responseDays([]models.Application{app})
	if len(got) != 1 || got[0] != 2.0 {
		t.Fatalf("expected a 2-day measurement from CreatedAt, got %v", got)
	}
}

func TestIsResponseStatus(t *testing.T) {
	cases := map[string]bool{
		"Interview":    true,
		" offer ":      true,
		"REJECTED":     true,
		"hired":        true,
		"Applied":      false,
		"under-review": false,
		"":             false,
	}
	for status, want := range cases {
		if got := isResponseStatus(status); got != want {
			t.Fatalf("isResponseStatus(%q) = %v, want %v", status, got, want)
		}
	}
}

func TestBucketIndexFor(t *testing.T) {
	cases := []struct {
		day  float64
		want int
	}{
		{0, 0}, {2.5, 0}, {3.0, 0}, {3.1, 1}, {7.0, 1}, {7.5, 2}, {14.0, 2}, {14.1, 3}, {300, 3},
	}
	for _, c := range cases {
		if got := bucketIndexFor(c.day); got != c.want {
			t.Fatalf("bucketIndexFor(%v) = %d, want %d", c.day, got, c.want)
		}
	}
}

func TestBucketResponseDays_OmitsEmptyBuckets(t *testing.T) {
	data := bucketResponseDays([]float64{0.5, 3.9, 30.0})
	if len(data) != 3 {
		t.Fatalf("expected three populated buckets, got %+v", data)
	}
	if data[0].DaysRange != "0-3 days" || data[0].Count != 1 || data[0].AvgDays != 0.5 {
		t.Fatalf("unexpected first bucket: %+v", data[0])
	}
	if data[1].DaysRange != "4-7 days" || data[1].Count != 1 {
		t.Fatalf("expected 3.9 days in 4-7, got %+v", data[1])
	}
	if data[2].DaysRange != "15+ days" || data[2].AvgDays != 30 {
		t.Fatalf("unexpected top bucket: %+v", data[2])
	}

	if empty := bucketResponseDays(nil); len(empty) != 0 {
		t.Fatalf("expected no buckets without data, got %+v", empty)
	}
}

func TestGetDashboardStats_CountsBothStatusCasings(t *testing.T) {
	apps := []models.Application{
		{UserID: 1, Status: "interview"},
		{UserID: 1, Status: "Interview"},
		{UserID: 1, Status: "offer"},
		{UserID: 1, Status: "Rejected"},
		{UserID: 1, Status: "applied"},
	}
	svc := &AnalyticsService{appRepo: &fakeApplicationReader{apps: apps}}

	stats, err := svc.GetDashboardStats(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.TotalApplications != 5 {
		t.Fatalf("expected 5 applications, got %d", stats.TotalApplications)
	}
	if stats.Interviews != 2 {
		t.Fatalf("expected lowercase and capitalised interviews to both count, got %d", stats.Interviews)
	}
	if stats.Offers != 1 || stats.Rejected != 1 {
		t.Fatalf("unexpected offers/rejected counts: %+v", stats)
	}
	if stats.ConversionRate != 20 {
		t.Fatalf("expected 20%% conversion from 1 offer in 5 applications, got %v", stats.ConversionRate)
	}
}

func TestGetWeeklySummary_CountsBothStatusCasings(t *testing.T) {
	now := time.Now().UTC()
	apps := []models.Application{
		{
			UserID:      1,
			Status:      "interview",
			AppliedDate: now.AddDate(0, 0, -1),
			Model:       gorm.Model{UpdatedAt: now},
		},
		{
			UserID:      1,
			Status:      "applied",
			AppliedDate: now.AddDate(0, 0, -2),
		},
	}
	svc := &AnalyticsService{appRepo: &fakeApplicationReader{apps: apps}}

	summary, err := svc.GetWeeklySummary(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.ApplicationsThisWeek != 2 {
		t.Fatalf("expected 2 applications this week, got %d", summary.ApplicationsThisWeek)
	}
	if summary.InterviewsThisWeek != 1 {
		t.Fatalf("expected the lowercase interview status to be counted, got %d", summary.InterviewsThisWeek)
	}
	for _, action := range summary.SuggestedActions {
		if action == "Follow up on pending applications" {
			return
		}
	}
	t.Fatalf("expected a follow-up suggestion for the pending application, got %+v", summary.SuggestedActions)
}
