package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"job_application_analysis/internal/models"

	"google.golang.org/api/gmail/v1"
)

// nlpTimeout bounds every call to the NLP service so a slow or hung service can
// never block the polling worker. It is a var so tests can shorten it.
var nlpTimeout = 5 * time.Second

// maxNLPTextChars caps how much email text we send to the NLP service.
const maxNLPTextChars = 20000

// nlpClassifyResponse mirrors the Python POST /classify response schema:
// {company: str|null, role: str|null, status: str, confidence: float(0-1), reason: str}.
// Reason is intentionally unused: it can quote email-derived text, so it is never
// persisted or logged.
type nlpClassifyResponse struct {
	Company    string  `json:"company"`
	Role       string  `json:"role"`
	Status     string  `json:"status"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

// emailClassification is the normalized, validated classification used to fill
// an EmailEvent.
type emailClassification struct {
	Company    string
	Role       string
	Status     string
	Confidence int
}

// nlpBaseURL resolves the NLP service base URL, reusing NLP_SERVICE_URL and the
// shared default (http://localhost:8000).
func (s *EmailService) nlpBaseURL() string {
	if s.nlpURL != "" {
		return s.nlpURL
	}
	return nlpServiceURL()
}

// classifyEmail calls the NLP /classify endpoint. It returns ok=false whenever the
// service is unreachable, returns a non-2xx status, returns invalid JSON, or omits
// a usable status — in all of which the caller falls back to the heuristic. It
// never panics and never logs email content, tokens, or raw upstream errors.
func (s *EmailService) classifyEmail(text string) (emailClassification, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return emailClassification{}, false
	}
	text = firstN(text, maxNLPTextChars)

	payload, err := json.Marshal(map[string]interface{}{"text": text, "verbose": false})
	if err != nil {
		return emailClassification{}, false
	}

	base := strings.TrimRight(s.nlpBaseURL(), "/")
	client := &http.Client{Timeout: nlpTimeout}
	resp, err := client.Post(base+"/classify", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return emailClassification{}, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return emailClassification{}, false
	}

	var out nlpClassifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return emailClassification{}, false
	}

	status := strings.TrimSpace(out.Status)
	if status == "" {
		// No usable classification — do not guess a status.
		return emailClassification{}, false
	}

	// NLP returns confidence as a 0-1 float; EmailEvent stores an int 0-100.
	conf := int(math.Round(out.Confidence * 100))
	if conf < 0 {
		conf = 0
	}
	if conf > 100 {
		conf = 100
	}

	return emailClassification{
		Company:    strings.TrimSpace(out.Company),
		Role:       strings.TrimSpace(out.Role),
		Status:     status,
		Confidence: conf,
	}, true
}

// populateClassification fills the event's detection fields, preferring the NLP
// result and falling back to the offline subject heuristic when NLP is unusable.
// The fallback deliberately does NOT invent a status or confidence: those are left
// empty/zero and the source is marked "heuristic".
func (s *EmailService) populateClassification(event *models.EmailEvent, subject, text string) {
	if cls, ok := s.classifyEmail(text); ok {
		event.DetectedCompany = cls.Company
		event.DetectedRole = cls.Role
		event.SuggestedStatus = cls.Status
		event.Confidence = cls.Confidence
		event.ClassificationSource = "nlp"
		return
	}

	event.DetectedCompany = extractCompanyFromSubject(subject)
	event.DetectedRole = ""
	event.SuggestedStatus = ""
	event.Confidence = 0
	event.ClassificationSource = "heuristic"
}

// emailNotificationMessage builds a user-facing message from only the data we
// actually have, without asserting a status or role that was not detected.
func emailNotificationMessage(event *models.EmailEvent) string {
	company := strings.TrimSpace(event.DetectedCompany)
	if company == "" {
		company = "an unknown sender"
	}
	role := strings.TrimSpace(event.DetectedRole)
	status := strings.TrimSpace(event.SuggestedStatus)

	switch {
	case status != "" && role != "":
		return fmt.Sprintf("Email from %s about the %s position (%s)", company, role, status)
	case status != "":
		return fmt.Sprintf("Email from %s detected as %s", company, status)
	default:
		return fmt.Sprintf("New job-related email detected from %s (not auto-classified)", company)
	}
}

// markRead removes the UNREAD label so a message is not re-fetched every cycle.
// Failures are logged without message content and do not abort processing.
func (s *EmailService) markRead(svc *gmail.Service, id string) {
	if id == "" {
		return
	}
	_, err := svc.Users.Messages.Modify("me", id, &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"UNREAD"},
	}).Do()
	if err != nil {
		log.Println("❌ Failed to mark email as read:", id)
	}
}
