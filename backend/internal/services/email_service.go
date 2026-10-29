package services

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"job_application_analysis/internal/models"
	"job_application_analysis/internal/repositories"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/googleapi"
	"gorm.io/gorm"
)

// The poller depends on these persistence slices rather than the concrete
// repositories so deduplication, mailbox isolation and database-error handling can be
// unit-tested. The project builds with CGO_ENABLED=0, which makes the SQLite driver
// used for throwaway test databases unusable.
type emailEventStore interface {
	Create(event *models.EmailEvent) error
	FindByID(id uint) (*models.EmailEvent, error)
	FindPendingByUserID(userID uint) ([]models.EmailEvent, error)
	FindRecentByUserID(userID uint, limit int) ([]models.EmailEvent, error)
	ExistsByEmailID(userID uint, emailID string) (bool, error)
	Update(event *models.EmailEvent) error
}

type notificationStore interface {
	Create(notification *models.Notification) error
}

// gmailTokenStore is the whole persistence surface Gmail needs: read the live grant,
// write one back, and record that a grant is dead or removed.
type gmailTokenStore interface {
	// grantForUser returns the newest grant stored for this user.
	grantForUser(userID uint) (*models.GmailToken, error)
	// allGrants returns at most one grant per user, ignoring the older duplicate
	// rows that repeated reconnects created before storage became an upsert.
	allGrants() ([]models.GmailToken, error)
	// saveGrant upserts a user's live grant. An empty refresh token never overwrites
	// a stored one, because Google only sends it again when consent is re-granted.
	saveGrant(token *models.GmailToken) error
	markNeedsReconnect(userID uint, reason string) error
	clearGrant(userID uint) error
}

type dbGmailTokenStore struct {
	db *gorm.DB
}

func (s dbGmailTokenStore) grantForUser(userID uint) (*models.GmailToken, error) {
	var token models.GmailToken
	if err := s.db.Where("user_id = ?", userID).Order("id DESC").First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

func (s dbGmailTokenStore) allGrants() ([]models.GmailToken, error) {
	var rows []models.GmailToken
	if err := s.db.Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}

	latest := make(map[uint]models.GmailToken, len(rows))
	for _, row := range rows {
		kept, seen := latest[row.UserID]
		if !seen || row.ID > kept.ID {
			latest[row.UserID] = row
		}
	}

	grants := make([]models.GmailToken, 0, len(latest))
	for _, row := range latest {
		grants = append(grants, row)
	}
	sort.Slice(grants, func(i, j int) bool { return grants[i].UserID < grants[j].UserID })
	return grants, nil
}

// mergeRefreshToken keeps a stored refresh token when the incoming grant does not
// carry one. Google only returns a refresh token together with a new consent, so an
// empty value means "unchanged", never "revoke it".
func mergeRefreshToken(incoming, stored string) string {
	if incoming == "" {
		return stored
	}
	return incoming
}

func (s dbGmailTokenStore) saveGrant(token *models.GmailToken) error {
	existing, err := s.grantForUser(token.UserID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.db.Create(token).Error
	}
	if err != nil {
		return err
	}

	return s.db.Model(&models.GmailToken{}).
		Where("id = ?", existing.ID).
		Updates(map[string]any{
			"access_token":     token.AccessToken,
			"refresh_token":    mergeRefreshToken(token.RefreshToken, existing.RefreshToken),
			"expiry":           token.Expiry,
			"needs_reconnect":  false,
			"reconnect_reason": "",
		}).Error
}

func (s dbGmailTokenStore) markNeedsReconnect(userID uint, reason string) error {
	return s.db.Model(&models.GmailToken{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"needs_reconnect":  true,
			"reconnect_reason": firstN(reason, 300),
		}).Error
}

// clearGrant removes the credentials for a disconnected user. The row itself stays so
// the sync history it belongs to is not destroyed, and a reconnect overwrites it.
func (s dbGmailTokenStore) clearGrant(userID uint) error {
	return s.db.Model(&models.GmailToken{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"access_token":     "",
			"refresh_token":    "",
			"needs_reconnect":  false,
			"reconnect_reason": "",
		}).Error
}

type EmailService struct {
	db               *gorm.DB
	appRepo          *repositories.ApplicationRepository
	emailRepo        emailEventStore
	userRepo         *repositories.UserRepository
	notificationRepo notificationStore
	tokens           gmailTokenStore
	nlpURL           string
	oauthConfig      *oauth2.Config
	stateStore       *OAuthStateStore
	// pollMailbox reads one account's unread messages. It is a field so routing,
	// locking and reconnect handling can be tested without contacting Gmail.
	pollMailbox func(token *models.GmailToken) (int, error)
	// userLocks maps userID -> *sync.Mutex so one mailbox is only ever polled by
	// one worker at a time (background poller and manual sync).
	userLocks sync.Map
}

// Sentinel errors let the HTTP layer answer with the right status instead of leaking
// an internal error string: "not connected" and "reconnect needed" look identical to
// a user unless they are distinguished.
var (
	ErrGmailNotConnected = errors.New("gmail is not connected")
	ErrGmailReconnect    = errors.New("gmail needs to be reconnected")
	// ErrGmailMisconfigured means Google refused the app's own OAuth client or the
	// Gmail API is not enabled. Only the deployment can fix this, so the user's
	// stored grant is left untouched and polling may resume once config is correct.
	ErrGmailMisconfigured = errors.New("gmail is not configured for this deployment")
)

// frontendBaseURL is where the OAuth callback sends the browser back to. It is env
// driven so production deployments are not pinned to the dev server port.
func frontendBaseURL() string {
	base := strings.TrimRight(os.Getenv("FRONTEND_URL"), "/")
	if base == "" {
		base = "http://localhost:5173"
	}
	return base
}

// OAuthReturnURL builds the browser redirect used when the Gmail flow finishes, on the
// candidate dashboard, where the Gmail card reads the query and updates itself.
func OAuthReturnURL(connected bool, reason string) string {
	query := url.Values{}
	if connected {
		query.Set("gmail", "connected")
		return frontendBaseURL() + "/candidate?" + query.Encode()
	}
	query.Set("gmail", "error")
	if reason != "" {
		query.Set("reason", reason)
	}
	return frontendBaseURL() + "/candidate?" + query.Encode()
}

// OAuthFailureReason turns a Google callback error code into something a user can act
// on. The raw code stays in the server log; the description from Google is not shown
// because it often describes configuration the user cannot read.
func OAuthFailureReason(code string) string {
	switch code {
	case "access_denied":
		return "Gmail connection was cancelled. No emails are imported until you authorize."
	case "invalid_grant":
		return "Google refused the authorization code. Start the Gmail connection again."
	case "invalid_client", "unauthorized_client":
		return "This app is not allowed to request Gmail access (check the Google OAuth client)."
	case "redirect_uri_mismatch", "invalid_redirect_uri":
		return "The redirect URI is not registered in the Google OAuth client."
	case "insufficient_scope", "scope_changed":
		return "Gmail access was not granted. Reconnect and keep the Gmail permission checked."
	default:
		return "Gmail connection failed. Try again."
	}
}

func NewEmailService(
	db *gorm.DB,
	appRepo *repositories.ApplicationRepository,
	emailRepo *repositories.EmailEventRepository,
	userRepo *repositories.UserRepository,
	notificationRepo *repositories.NotificationRepository,
	nlpURL string,
) *EmailService {

	oauthConfig := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/gmail.modify",
		},
		Endpoint: google.Endpoint,
	}

	// Only configuration presence is logged: client id, secret, tokens and
	// authorization codes never reach the log.
	log.Println("📧 Gmail OAuth configuration:")
	log.Println("   client id configured:", oauthConfig.ClientID != "")
	log.Println("   client secret configured:", oauthConfig.ClientSecret != "")
	log.Println("   redirect url:", oauthConfig.RedirectURL)

	service := &EmailService{
		db:               db,
		appRepo:          appRepo,
		emailRepo:        emailRepo,
		userRepo:         userRepo,
		notificationRepo: notificationRepo,
		tokens:           dbGmailTokenStore{db: db},
		nlpURL:           nlpURL,
		oauthConfig:      oauthConfig,
		stateStore:       NewOAuthStateStore(oauthStateTTL),
	}
	service.pollMailbox = service.pollUser
	return service
}

// DB getter
func (s *EmailService) DB() *gorm.DB {
	return s.db
}

// OAuth
//
// ApprovalForce is required as well as AccessTypeOffline: Google only includes a
// refresh token in the very first consent response, so without prompt=consent every
// reconnect returned an access token that expired an hour later and could never be
// refreshed.
func (s *EmailService) GetAuthURL(state string) string {
	return s.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// ConnectURL builds the browser-facing authorization URL for one user. The UI cannot
// send a bearer token in a top-level navigation, so it asks the API for this URL and
// then redirects the browser to it.
func (s *EmailService) ConnectURL(userID uint) (string, error) {
	state, err := s.stateStore.Issue(userID)
	if err != nil {
		return "", err
	}
	return s.GetAuthURL(state), nil
}

func (s *EmailService) ExchangeCode(code string) (*oauth2.Token, error) {
	ctx := context.Background()
	return s.oauthConfig.Exchange(ctx, code)
}

// PersistGmailGrant stores the result of a completed exchange as this user's live
// grant and marks the account connected.
func (s *EmailService) PersistGmailGrant(userID uint, token *oauth2.Token) error {
	grant := &models.GmailToken{
		UserID:       userID,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}
	if err := s.tokens.saveGrant(grant); err != nil {
		return fmt.Errorf("save gmail grant: %w", err)
	}
	return s.setConnectionFlag(userID, true)
}

// DisconnectGmail removes this user's stored credentials and clears the connected
// flag. The poller stops touching the mailbox because the grant is now empty.
func (s *EmailService) DisconnectGmail(userID uint) error {
	if err := s.tokens.clearGrant(userID); err != nil {
		return fmt.Errorf("clear gmail grant: %w", err)
	}
	return s.setConnectionFlag(userID, false)
}

// setConnectionFlag keeps the legacy users.gmail_connected column in sync with the
// stored grant so the older settings endpoint keeps reporting the truth.
func (s *EmailService) setConnectionFlag(userID uint, connected bool) error {
	if s.userRepo == nil {
		return nil
	}
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("load user for gmail flag: %w", err)
	}
	user.GmailConnected = connected
	if err := s.userRepo.Update(user); err != nil {
		return fmt.Errorf("update gmail connected flag: %w", err)
	}
	return nil
}

// withUserLock serializes all mailbox work for a single user.
func (s *EmailService) withUserLock(userID uint, fn func()) {
	value, _ := s.userLocks.LoadOrStore(userID, &sync.Mutex{})
	mutex := value.(*sync.Mutex)
	mutex.Lock()
	defer mutex.Unlock()
	fn()
}

// grantForUser loads the live Gmail grant for exactly one user.
func (s *EmailService) grantForUser(userID uint) (*models.GmailToken, error) {
	return s.tokens.grantForUser(userID)
}

// storedToken maps a persisted grant onto an oauth2 token.
func storedToken(grant *models.GmailToken) *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  grant.AccessToken,
		RefreshToken: grant.RefreshToken,
		Expiry:       grant.Expiry,
		TokenType:    "Bearer",
	}
}

// persistingTokenSource refreshes an access token through Google and writes the new
// token back to the store. Without the write-back, a rotated refresh token is dropped
// on the floor and the next process start has no usable grant at all.
type persistingTokenSource struct {
	base    oauth2.TokenSource
	userID  uint
	store   gmailTokenStore
	mu      sync.Mutex
	current models.GmailToken
}

func (p *persistingTokenSource) Token() (*oauth2.Token, error) {
	token, err := p.base.Token()
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	changed := token.AccessToken != p.current.AccessToken ||
		token.RefreshToken != p.current.RefreshToken ||
		!token.Expiry.Equal(p.current.Expiry)
	if !changed {
		return token, nil
	}

	grant := models.GmailToken{
		UserID:       p.userID,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}
	if err := p.store.saveGrant(&grant); err != nil {
		// The refreshed token is still usable for this request, so the mailbox read
		// proceeds; the next poll simply refreshes again.
		log.Println("⚠️ Failed to persist refreshed Gmail token for user", p.userID)
		return token, nil
	}
	p.current = grant
	return token, nil
}

func (s *EmailService) gmailServiceFor(token *models.GmailToken) (*gmail.Service, error) {
	source := s.oauthConfig.TokenSource(context.Background(), storedToken(token))
	client := &http.Client{
		Transport: &oauth2.Transport{
			Source: &persistingTokenSource{
				base:    source,
				userID:  token.UserID,
				store:   s.tokens,
				current: *token,
			},
		},
	}
	return gmail.New(client)
}

// Gmail Polling
func (s *EmailService) PollUnreadEmails() {
	grants, err := s.tokens.allGrants()
	if err != nil {
		log.Println("❌ Failed to load Gmail tokens")
		return
	}

	for i := range grants {
		grant := grants[i]
		// A blank or dead grant is deliberately skipped: retrying it every cycle is
		// what produced the endless invalid_grant log spam. Only a reconnect clears it.
		if grant.AccessToken == "" || grant.NeedsReconnect {
			continue
		}
		s.withUserLock(grant.UserID, func() { s.pollMailbox(&grant) })
	}
}

// SyncUserEmails polls only the requesting user's mailbox and returns the number of
// email events newly recorded by this call.
func (s *EmailService) SyncUserEmails(userID uint) (int, error) {
	token, err := s.grantForUser(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrGmailNotConnected
		}
		return 0, err
	}
	if token.AccessToken == "" {
		return 0, ErrGmailNotConnected
	}
	if token.NeedsReconnect {
		return 0, ErrGmailReconnect
	}

	var firstErr error
	count := 0
	s.withUserLock(userID, func() {
		count, firstErr = s.pollMailbox(token)
	})
	return count, firstErr
}

// pollUser processes one mailbox. Callers must already hold that user's lock.
func (s *EmailService) pollUser(token *models.GmailToken) (int, error) {
	gmailService, err := s.gmailServiceFor(token)
	if err != nil {
		return 0, fmt.Errorf("gmail client: %w", err)
	}

	resp, err := gmailService.Users.Messages.List("me").
		Q("is:unread newer_than:2d").
		MaxResults(5).
		Do()

	if err != nil {
		fault, reason := classifyGmailError(err)
		switch fault {
		case gmailFaultReconnect:
			// Persisted so neither this process nor a restarted one keeps hammering a
			// grant Google has already refused.
			if markErr := s.tokens.markNeedsReconnect(token.UserID, reason); markErr != nil {
				log.Println("⚠️ Failed to record that Gmail needs reconnecting for user", token.UserID)
			}
			log.Printf("⚠️ Gmail grant unusable for user %d: %s", token.UserID, reason)
			return 0, fmt.Errorf("%w: %s", ErrGmailReconnect, reason)
		case gmailFaultMisconfigured:
			// The user's grant is not at fault, so it must not be marked dead; the
			// operator has to fix the OAuth client.
			log.Printf("⚠️ Gmail is not configured for this deployment: %s", reason)
			return 0, fmt.Errorf("%w: %s", ErrGmailMisconfigured, reason)
		default:
			log.Println("❌ Gmail fetch error:", gmailErrorSummary(err))
			return 0, fmt.Errorf("gmail fetch: %w", err)
		}
	}

	log.Println("📩 Unread messages found:", len(resp.Messages))

	processed := 0
	for _, msg := range resp.Messages {
		fullMsg, err := gmailService.Users.Messages.Get("me", msg.Id).
			Format("full").
			Do()
		if err != nil {
			log.Println("❌ Message fetch error for", msg.Id)
			continue
		}

		created, err := s.recordEvent(token.UserID, msg.Id,
			extractEmailSubject(fullMsg), extractEmailBody(fullMsg))
		if err != nil {
			// Nothing was written, so the message is left unread and retried next poll.
			continue
		}
		if created {
			processed++
		}
		s.markRead(gmailService, msg.Id)
	}

	return processed, nil
}

// gmailFault distinguishes the two ways Gmail can refuse a request, because the fix
// is different: a dead grant needs the user to reauthorize, while a rejected OAuth
// client is a server configuration problem that reauthorizing cannot solve.
type gmailFault int

const (
	gmailFaultTransient gmailFault = iota
	gmailFaultReconnect
	gmailFaultMisconfigured
)

// classifyGmailError inspects the error codes Google returns, never any text that
// could contain a token.
func classifyGmailError(err error) (gmailFault, string) {
	if err == nil {
		return gmailFaultTransient, ""
	}

	var retrieve *oauth2.RetrieveError
	if errors.As(err, &retrieve) {
		switch retrieve.ErrorCode {
		case "invalid_grant":
			return gmailFaultReconnect, "Google no longer accepts the stored grant (it was revoked or expired)"
		case "invalid_client", "unauthorized_client":
			return gmailFaultMisconfigured, "Google rejected this app's OAuth client (check GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET)"
		case "invalid_scope":
			return gmailFaultMisconfigured, "the requested Gmail scope is not allowed for this app"
		}
		if retrieve.Response != nil {
			switch retrieve.Response.StatusCode {
			case http.StatusBadRequest:
				return gmailFaultReconnect, "Google rejected the stored grant"
			case http.StatusUnauthorized, http.StatusForbidden:
				return gmailFaultMisconfigured, "Google refused this app's OAuth credentials"
			}
		}
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		if isAccessNotConfigured(apiErr) {
			return gmailFaultMisconfigured, "the Gmail API is not enabled for this Google Cloud project"
		}
		switch apiErr.Code {
		case http.StatusUnauthorized:
			return gmailFaultReconnect, "Gmail rejected the access token"
		case http.StatusForbidden:
			return gmailFaultReconnect, "this Gmail account is not authorized for the Gmail API"
		}
	}

	msg := err.Error()
	switch {
	case strings.Contains(msg, "invalid_grant"):
		return gmailFaultReconnect, "Google no longer accepts the stored grant (it was revoked or expired)"
	case strings.Contains(msg, "refresh token is not set"):
		return gmailFaultReconnect, "the stored grant has no refresh token, so it expires after one hour"
	case strings.Contains(msg, "invalid_client"):
		return gmailFaultMisconfigured, "Google rejected this app's OAuth client (check GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET)"
	}

	return gmailFaultTransient, ""
}

// isAccessNotConfigured detects Google's "API not enabled for this project" error,
// which looks like a permission failure but is purely server-side configuration.
func isAccessNotConfigured(apiErr *googleapi.Error) bool {
	if apiErr == nil {
		return false
	}
	for _, detail := range apiErr.Errors {
		if detail.Reason == "accessNotConfigured" {
			return true
		}
	}

	msg := strings.ToLower(apiErr.Message)
	return strings.Contains(msg, "accessnotconfigured") ||
		strings.Contains(msg, "has not been used in project") && strings.Contains(msg, "or it is disabled")
}

// gmailErrorSummary keeps Gmail API failures in the log without ever writing a token,
// authorization code or message payload. OAuth error text can contain the raw token
// endpoint response, so known secret fields are redacted before anything is logged.
var credentialPattern = regexp.MustCompile(
	`(?i)(access_token|refresh_token|id_token|client_secret|code|token)[\"']?\s*[:=]\s*[\"']?[^\s,\"'}]+`)

func gmailErrorSummary(err error) string {
	if err == nil {
		return ""
	}

	var retrieve *oauth2.RetrieveError
	if errors.As(err, &retrieve) {
		status := 0
		if retrieve.Response != nil {
			status = retrieve.Response.StatusCode
		}
		code := retrieve.ErrorCode
		if code == "" {
			code = "unspecified"
		}
		return fmt.Sprintf("gmail oauth: token endpoint returned %d (%s)", status, code)
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		return fmt.Sprintf("googleapi: Error %d %s", apiErr.Code, apiErr.Message)
	}

	summary := credentialPattern.ReplaceAllString(err.Error(), "$1=[redacted]")
	return firstN(strings.ReplaceAll(summary, "\n", " "), 200)
}

// GmailConnection is the non-secret view of one user's Gmail state that the UI needs
// to decide between "connect", "reconnect" and "sync now".
type GmailConnection struct {
	Connected       bool
	NeedsReconnect  bool
	Reason          string
	HasRefreshToken bool
}

// ConnectionStatus reads the stored grant for one user. It never returns token
// material, and the answer comes from the grant itself rather than from the legacy
// users.gmail_connected flag, so a row written by the old insert-only code cannot
// report a connection that no longer works.
func (s *EmailService) ConnectionStatus(userID uint) (GmailConnection, error) {
	status := GmailConnection{}

	grant, err := s.grantForUser(userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		status.Reason = "Gmail is not connected to this account"
		return status, nil
	}
	if err != nil {
		return status, err
	}

	status.NeedsReconnect = grant.NeedsReconnect
	status.Reason = grant.ReconnectReason
	status.HasRefreshToken = grant.RefreshToken != ""
	status.Connected = grant.AccessToken != "" && !grant.NeedsReconnect

	switch {
	case grant.AccessToken == "":
		status.Reason = "Gmail is not connected to this account"
	case grant.NeedsReconnect:
		if status.Reason == "" {
			status.Reason = "Gmail authorization was revoked or expired"
		}
	case grant.RefreshToken == "":
		// Reported so the UI can ask for a reconnect before the hour is up. The grant
		// itself stays unmarked: it still works until it expires, and the first failed
		// refresh records it.
		status.Reason = "This connection has no refresh token, so it stops working after one hour. Reconnect Gmail."
		status.NeedsReconnect = true
		status.Connected = false
	}

	return status, nil
}

// RecentEmails lists the recruitment emails detected for one user, newest first. The
// limit is bounded so a caller cannot ask for the whole mailbox history at once.
func (s *EmailService) RecentEmails(userID uint, limit int) ([]models.EmailEvent, error) {
	switch {
	case limit <= 0:
		limit = 10
	case limit > 50:
		limit = 50
	}
	return s.emailRepo.FindRecentByUserID(userID, limit)
}

// recordEvent deduplicates, stores and notifies for one Gmail message. It reports
// created=false when the message was already recorded, and a non-nil error when
// nothing was written and the message should be retried later. Subject and body are
// used for classification only and are never logged or stored verbatim.
func (s *EmailService) recordEvent(userID uint, emailID, subject, body string) (bool, error) {
	exists, err := s.emailRepo.ExistsByEmailID(userID, emailID)
	if err != nil {
		// A failed lookup cannot prove the message is new, so skipping it is safer
		// than inserting a possible duplicate.
		log.Println("❌ Duplicate check failed for", emailID)
		return false, err
	}
	if exists {
		return false, nil
	}

	text := strings.TrimSpace(subject + "\n" + body)
	emailEvent := &models.EmailEvent{
		UserID:      userID,
		EmailID:     emailID,
		ProcessedAt: time.Now(),
		Confirmed:   false,
	}
	// Prefer the NLP service; fall back to the subject heuristic (without inventing
	// a status or confidence) when NLP is unusable.
	s.populateClassification(emailEvent, subject, text)

	if err := s.emailRepo.Create(emailEvent); err != nil {
		log.Println("❌ Failed to save email event for", emailID)
		return false, err
	}

	notification := &models.Notification{
		UserID:    userID,
		Type:      "email",
		Title:     "New Job Email Detected",
		Message:   emailNotificationMessage(emailEvent),
		Read:      false,
		ActionURL: "/emails",
	}
	if err := s.notificationRepo.Create(notification); err != nil {
		log.Println("❌ Failed to create notification for", emailID)
	}

	log.Printf("✅ Processed email %s status=%q confidence=%d source=%s",
		emailID, emailEvent.SuggestedStatus, emailEvent.Confidence, emailEvent.ClassificationSource)

	return true, nil
}

// Extract email body
func extractEmailBody(msg *gmail.Message) string {

	if msg.Payload == nil {
		return ""
	}

	if msg.Payload.Body != nil && msg.Payload.Body.Data != "" {
		data, _ := base64.URLEncoding.DecodeString(msg.Payload.Body.Data)
		return string(data)
	}

	for _, part := range msg.Payload.Parts {
		if part.Body != nil && part.Body.Data != "" {
			data, _ := base64.URLEncoding.DecodeString(part.Body.Data)
			return string(data)
		}
	}

	return ""
}

// Extract email subject
func extractEmailSubject(msg *gmail.Message) string {
	if msg.Payload == nil {
		return ""
	}
	for _, header := range msg.Payload.Headers {
		if header.Name == "Subject" {
			return header.Value
		}
	}
	return "No Subject"
}

// Extract company name from subject (simple heuristic)
func extractCompanyFromSubject(subject string) string {
	// Common patterns in job emails
	// "Interview with Google", "Your application at Microsoft", "Amazon - Next Steps"

	// Try to extract company name from common patterns
	subject = strings.ToLower(subject)

	// Pattern: "from [Company]"
	if strings.Contains(subject, "from ") {
		parts := strings.Split(subject, "from ")
		if len(parts) > 1 {
			company := strings.TrimSpace(parts[1])
			if len(company) > 0 && len(company) < 50 {
				return strings.Title(strings.Split(company, " ")[0])
			}
		}
	}

	// Pattern: "at [Company]"
	if strings.Contains(subject, " at ") {
		parts := strings.Split(subject, " at ")
		if len(parts) > 1 {
			company := strings.TrimSpace(parts[1])
			if len(company) > 0 && len(company) < 50 {
				return strings.Title(strings.Split(company, " ")[0])
			}
		}
	}

	// Pattern: "[Company] -" or "[Company]:"
	if strings.Contains(subject, " - ") {
		parts := strings.Split(subject, " - ")
		if len(parts) > 0 {
			company := strings.TrimSpace(parts[0])
			if len(company) > 0 && len(company) < 50 {
				return strings.Title(company)
			}
		}
	}

	// Default: return first 30 chars of subject
	if len(subject) > 30 {
		return strings.Title(subject[:30]) + "..."
	}
	return strings.Title(subject)
}

func firstN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// Existing suggestion logic
func (s *EmailService) GetPendingSuggestions(userID uint) ([]models.EmailEvent, error) {
	return s.emailRepo.FindPendingByUserID(userID)
}

func (s *EmailService) ConfirmSuggestion(
	userID,
	emailEventID uint,
	applicationID *uint,
	status string,
) (*models.Application, error) {

	var resultApp *models.Application

	err := s.db.Transaction(func(tx *gorm.DB) error {

		event, err := s.emailRepo.FindByID(emailEventID)
		if err != nil {
			return err
		}

		if event.UserID != userID {
			return errors.New("unauthorized")
		}

		var app *models.Application

		if applicationID != nil {

			app, err = s.appRepo.FindByID(*applicationID)
			if err != nil {
				return err
			}

			app.Status = status
			app.LastUpdated = time.Now()

			if err := tx.Save(app).Error; err != nil {
				return err
			}

		} else {

			app = &models.Application{
				UserID:      userID,
				Company:     event.DetectedCompany,
				Role:        event.DetectedRole,
				AppliedDate: time.Now(),
				Status:      status,
				LastUpdated: time.Now(),
				Source:      "Email",
			}

			if err := tx.Create(app).Error; err != nil {
				return err
			}
		}

		event.Confirmed = true
		event.ApplicationID = &app.ID

		if err := tx.Save(event).Error; err != nil {
			return err
		}

		// Create notification for confirmed email
		action := "updated"
		if applicationID == nil {
			action = "created"
		}
		notification := &models.Notification{
			UserID:    userID,
			Type:      "application",
			Title:     "Email Confirmed",
			Message:   "Application for " + app.Role + " at " + app.Company + " " + action + " from email",
			Read:      false,
			ActionURL: "/applications",
		}
		if err := s.notificationRepo.Create(notification); err != nil {
			log.Println("⚠️  Failed to create notification:", err)
		}

		resultApp = app
		return nil
	})

	if err != nil {
		return nil, err
	}

	return resultApp, nil
}

func (s *EmailService) IgnoreSuggestion(userID, emailEventID uint) error {

	event, err := s.emailRepo.FindByID(emailEventID)
	if err != nil {
		return err
	}

	if event.UserID != userID {
		return errors.New("unauthorized")
	}

	event.Confirmed = true
	return s.emailRepo.Update(event)
}

// CreateConnectionNotification creates a notification when Gmail is connected
func (s *EmailService) CreateConnectionNotification(notification *models.Notification) error {
	return s.notificationRepo.Create(notification)
}
