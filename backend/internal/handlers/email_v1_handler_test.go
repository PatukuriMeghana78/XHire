package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"job_application_analysis/internal/dto"
	"job_application_analysis/internal/models"
	"job_application_analysis/internal/services"
)

func TestWriteGmailErrorMapsSentinels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"not connected", services.ErrGmailNotConnected, http.StatusConflict, "gmail_not_connected"},
		{"reconnect", services.ErrGmailReconnect, http.StatusConflict, "gmail_reconnect_required"},
		{"wrapped reconnect", fmt.Errorf("%w: google refused", services.ErrGmailReconnect), http.StatusConflict, "gmail_reconnect_required"},
		{
			"misconfigured server, not the user's fault",
			fmt.Errorf("%w: check GOOGLE_CLIENT_ID", services.ErrGmailMisconfigured),
			http.StatusServiceUnavailable,
			"gmail_misconfigured",
		},
		{"transient", errors.New("gmail fetch: dial tcp: lookup failed"), http.StatusBadGateway, "gmail_sync_failed"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)

			writeGmailError(c, tc.err)

			if recorder.Code != tc.wantStatus {
				t.Fatalf("status = %d want %d", recorder.Code, tc.wantStatus)
			}

			var body map[string]string
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("invalid json: %v", err)
			}
			if body["error"] != tc.wantCode {
				t.Fatalf("code = %q want %q", body["error"], tc.wantCode)
			}
			if body["message"] == "" {
				t.Fatal("the response must carry a message the UI can show")
			}
			if strings.Contains(body["message"], "dial tcp") {
				t.Fatalf("internal error text leaked to the client: %q", body["message"])
			}
		})
	}
}

func TestGmailTokenModelNeverSerializesCredentials(t *testing.T) {
	grant := models.GmailToken{
		UserID:       3,
		AccessToken:  "access-secret",
		RefreshToken: "refresh-secret",
	}

	payload, err := json.Marshal(grant)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	body := string(payload)
	if strings.Contains(body, "access-secret") || strings.Contains(body, "refresh-secret") {
		t.Fatalf("stored credentials are serializable: %s", body)
	}
}

func TestGmailStatusResponseCarriesNoCredentials(t *testing.T) {
	payload, err := json.Marshal(dto.GmailStatusResponse{
		Connected: true, HasRefreshToken: true, Reason: "ok",
	})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	for _, forbidden := range []string{"access_token", "refresh_token", "expiry"} {
		if strings.Contains(string(payload), forbidden) {
			t.Fatalf("status response exposes %q: %s", forbidden, payload)
		}
	}
}

func TestEmailEventFromModelShapes(t *testing.T) {
	applicationID := uint(7)
	event := models.EmailEvent{
		SuggestedStatus:      "Under Review",
		DetectedCompany:      "Acme",
		DetectedRole:         "Backend Engineer",
		Confidence:           82,
		ClassificationSource: "nlp",
		ApplicationID:        &applicationID,
	}

	out := dto.EmailEventFromModel(&event)
	if out.SuggestedStatus != "under-review" {
		t.Fatalf("status not normalized: %q", out.SuggestedStatus)
	}
	if out.ApplicationID != "7" {
		t.Fatalf("expected the linked application id, got %q", out.ApplicationID)
	}
	if out.DetectedAt == "" {
		t.Fatal("expected a detected timestamp")
	}

	payload, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	// The poller deliberately stores no message content, so nothing can leak here.
	if strings.Contains(string(payload), "body") || strings.Contains(string(payload), "subject") {
		t.Fatalf("email payload must not carry message content: %s", payload)
	}
}
