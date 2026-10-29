package services

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"job_application_analysis/internal/models"
)

// newTestEmailService builds an EmailService pointed at the given NLP base URL.
// Only nlpURL is needed for classification; no DB is touched.
func newTestEmailService(nlpURL string) *EmailService {
	return &EmailService{nlpURL: nlpURL}
}

func TestClassifyEmail_Success(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"company":"Acme Corp","role":"Backend Engineer","status":"Interview","confidence":0.82,"reason":"Detected keywords: interview"}`))
	}))
	defer ts.Close()

	svc := newTestEmailService(ts.URL)
	cls, ok := svc.classifyEmail("Subject: Interview\nHello, we would like to schedule an interview.")
	if !ok {
		t.Fatal("expected ok=true for a valid NLP response")
	}
	if gotPath != "/classify" {
		t.Fatalf("expected POST to /classify, got %q", gotPath)
	}
	if _, hasText := gotBody["text"]; !hasText {
		t.Fatalf("expected request body to include 'text', got %v", gotBody)
	}
	if cls.Company != "Acme Corp" || cls.Role != "Backend Engineer" || cls.Status != "Interview" || cls.Confidence != 82 {
		t.Fatalf("unexpected classification: %+v", cls)
	}
}

func TestClassifyEmail_Non200(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	if _, ok := newTestEmailService(ts.URL).classifyEmail("some text"); ok {
		t.Fatal("expected ok=false on non-200 response")
	}
}

func TestClassifyEmail_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("this is not json"))
	}))
	defer ts.Close()

	if _, ok := newTestEmailService(ts.URL).classifyEmail("some text"); ok {
		t.Fatal("expected ok=false on invalid JSON")
	}
}

func TestClassifyEmail_MissingStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"company":"Acme","role":"Dev","status":"","confidence":0.9}`))
	}))
	defer ts.Close()

	if _, ok := newTestEmailService(ts.URL).classifyEmail("some text"); ok {
		t.Fatal("expected ok=false when status is missing/empty")
	}
}

func TestClassifyEmail_EmptyText(t *testing.T) {
	called := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer ts.Close()

	if _, ok := newTestEmailService(ts.URL).classifyEmail("   "); ok {
		t.Fatal("expected ok=false for empty text")
	}
	if called {
		t.Fatal("NLP service should not be called for empty text")
	}
}

func TestClassifyEmail_Timeout(t *testing.T) {
	orig := nlpTimeout
	nlpTimeout = 30 * time.Millisecond
	defer func() { nlpTimeout = orig }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"Interview","confidence":0.8}`))
	}))
	defer ts.Close()

	if _, ok := newTestEmailService(ts.URL).classifyEmail("some text"); ok {
		t.Fatal("expected ok=false when the NLP call times out")
	}
}

func TestClassifyEmail_Unavailable(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close() // URL now refuses connections

	if _, ok := newTestEmailService(ts.URL).classifyEmail("some text"); ok {
		t.Fatal("expected ok=false when the NLP service is unreachable")
	}
}

func TestClassifyEmail_ConfidenceClamped(t *testing.T) {
	high := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"Offer","confidence":3.0}`))
	}))
	defer high.Close()
	cls, ok := newTestEmailService(high.URL).classifyEmail("text")
	if !ok || cls.Confidence != 100 {
		t.Fatalf("expected confidence clamped to 100, got %+v ok=%v", cls, ok)
	}

	low := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"Applied","confidence":-1.0}`))
	}))
	defer low.Close()
	cls, ok = newTestEmailService(low.URL).classifyEmail("text")
	if !ok || cls.Confidence != 0 {
		t.Fatalf("expected confidence clamped to 0, got %+v ok=%v", cls, ok)
	}

	// Verify the 0-1 float is scaled to the 0-100 int we persist.
	normal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"Applied","confidence":0.4,"reason":"Defaulted to Applied."}`))
	}))
	defer normal.Close()
	cls, ok = newTestEmailService(normal.URL).classifyEmail("text")
	if !ok || cls.Confidence != 40 {
		t.Fatalf("expected 0.4 scaled to 40, got %+v ok=%v", cls, ok)
	}
}

func TestPopulateClassification_NLP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"company":"Acme","role":"SRE","status":"Rejected","confidence":0.7}`))
	}))
	defer ts.Close()

	event := &models.EmailEvent{}
	newTestEmailService(ts.URL).populateClassification(event, "Subject: Update", "some email text")

	if event.ClassificationSource != "nlp" {
		t.Fatalf("expected source=nlp, got %q", event.ClassificationSource)
	}
	if event.SuggestedStatus != "Rejected" || event.DetectedRole != "SRE" || event.DetectedCompany != "Acme" || event.Confidence != 70 {
		t.Fatalf("NLP fields not mapped: %+v", event)
	}
}

func TestPopulateClassification_Fallback(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close() // unreachable -> heuristic fallback

	event := &models.EmailEvent{}
	newTestEmailService(ts.URL).populateClassification(event, "Your application at Acme", "body text")

	if event.ClassificationSource != "heuristic" {
		t.Fatalf("expected source=heuristic, got %q", event.ClassificationSource)
	}
	// Fallback must NOT invent a status or confidence.
	if event.SuggestedStatus != "" {
		t.Fatalf("fallback must not set a status, got %q", event.SuggestedStatus)
	}
	if event.Confidence != 0 {
		t.Fatalf("fallback must not set a confidence, got %d", event.Confidence)
	}
	if event.DetectedRole != "" {
		t.Fatalf("fallback must not set a role, got %q", event.DetectedRole)
	}
}

func TestEmailNotificationMessage(t *testing.T) {
	full := &models.EmailEvent{DetectedCompany: "Acme", DetectedRole: "Dev", SuggestedStatus: "Interview"}
	if msg := emailNotificationMessage(full); msg == "" {
		t.Fatal("expected a non-empty message")
	}

	statusOnly := &models.EmailEvent{DetectedCompany: "Acme", SuggestedStatus: "Offer"}
	if msg := emailNotificationMessage(statusOnly); msg == "" {
		t.Fatal("expected a non-empty message")
	}

	unclassified := &models.EmailEvent{}
	msg := emailNotificationMessage(unclassified)
	if msg == "" {
		t.Fatal("expected a non-empty message for an unclassified event")
	}
	// Must not assert a status that was never detected.
	for _, bad := range []string{"Interview", "Rejected", "Offer", "Applied"} {
		if strings.Contains(msg, bad) {
			t.Fatalf("unclassified message should not claim status %q: %s", bad, msg)
		}
	}
}

// TestClassifyEmail_LiveService exercises a real HTTP round-trip against a running
// NLP service. It is skipped unless XHIRE_NLP_LIVE_URL points at one, so the
// offline suite stays hermetic.
func TestClassifyEmail_LiveService(t *testing.T) {
	base := strings.TrimSpace(os.Getenv("XHIRE_NLP_LIVE_URL"))
	if base == "" {
		t.Skip("XHIRE_NLP_LIVE_URL not set; skipping live NLP integration test")
	}

	cls, ok := newTestEmailService(base).classifyEmail(
		"Subject: Interview invitation\nWe would like to schedule a technical round for the Backend Engineer role at Acme Corp.")
	if !ok {
		t.Fatal("expected the live NLP service to return a usable classification")
	}
	if cls.Status == "" {
		t.Fatalf("live service returned an empty status: %+v", cls)
	}
	if cls.Confidence < 0 || cls.Confidence > 100 {
		t.Fatalf("confidence out of range: %+v", cls)
	}
	t.Logf("live NLP result: %+v", cls)
}
