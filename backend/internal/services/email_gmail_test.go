package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"job_application_analysis/internal/models"

	"golang.org/x/oauth2"
	"google.golang.org/api/googleapi"
	"gorm.io/gorm"
)

// newGmailTestService builds an EmailService whose only external dependency is a
// local token endpoint, so the refresh and reconnect paths can be exercised without
// Gmail, Google or a database.
func newGmailTestService(t *testing.T, tokenURL string) (*EmailService, *fakeTokenStore) {
	t.Helper()

	tokens := &fakeTokenStore{tokensByUser: map[uint]*models.GmailToken{}}
	events := &fakeEventStore{}
	svc := &EmailService{
		emailRepo:        events,
		notificationRepo: &fakeNotificationStore{},
		tokens:           tokens,
		stateStore:       NewOAuthStateStore(oauthStateTTL),
		oauthConfig: &oauth2.Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "http://localhost:8080/api/email/oauth/callback",
			Scopes:       []string{"https://www.googleapis.com/auth/gmail.modify"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.google.com/o/oauth2/auth",
				TokenURL: tokenURL,
			},
		},
	}
	svc.pollMailbox = svc.pollUser
	return svc, tokens
}

func TestMergeRefreshToken(t *testing.T) {
	if got := mergeRefreshToken("", "stored-refresh"); got != "stored-refresh" {
		t.Fatalf("an empty incoming token must not erase the stored one, got %q", got)
	}
	if got := mergeRefreshToken("new-refresh", "stored-refresh"); got != "new-refresh" {
		t.Fatalf("expected the rotated token to win, got %q", got)
	}
}

func TestGetAuthURLRequestsOfflineConsent(t *testing.T) {
	svc, _ := newGmailTestService(t, "https://oauth2.example.test/token")

	url := svc.GetAuthURL("state-value")
	if !strings.Contains(url, "access_type=offline") {
		t.Fatal("offline access is required or Google never issues a refresh token")
	}
	if !strings.Contains(url, "prompt=consent") {
		t.Fatal("consent must be forced or the refresh token is only sent on first approval")
	}
	if !strings.Contains(url, "gmail.modify") {
		t.Fatalf("expected the gmail.modify scope in %q", url)
	}
	if strings.Contains(url, "test-client-secret") {
		t.Fatal("the client secret must never appear in an authorization URL")
	}
}

func TestConnectURLIssuesStateBoundToUser(t *testing.T) {
	svc, _ := newGmailTestService(t, "https://oauth2.example.test/token")

	authURL, err := svc.ConnectURL(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(authURL, "https://accounts.google.com/o/oauth2/auth") {
		t.Fatalf("unexpected auth url: %q", authURL)
	}

	state := strings.TrimPrefix(strings.Split(authURL, "state=")[1], "")
	if state == "" {
		t.Fatal("no state issued")
	}
	userID, ok := svc.ValidateOAuthState(state)
	if !ok || userID != 42 {
		t.Fatalf("state did not resolve to the starting user: %d %v", userID, ok)
	}
	if _, replay := svc.ValidateOAuthState(state); replay {
		t.Fatal("state must be single use")
	}
}

func TestClassifyGmailError(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		want     gmailFault
		wantText string
	}{
		{"nil", nil, gmailFaultTransient, ""},
		{"plain network error", errors.New("dial tcp: lookup failed"), gmailFaultTransient, ""},
		{
			"refresh token missing",
			errors.New("oauth2: token expired and refresh token is not set"),
			gmailFaultReconnect,
			"no refresh token",
		},
		{
			"the exact reported failure text",
			errors.New(`oauth2: "invalid_grant" "Bad Request"`),
			gmailFaultReconnect,
			"stored grant",
		},
		{
			"retrieve invalid grant",
			&oauth2.RetrieveError{ErrorCode: "invalid_grant", Response: &http.Response{StatusCode: 400}},
			gmailFaultReconnect,
			"stored grant",
		},
		{
			"a rejected client is configuration, not a dead grant",
			&oauth2.RetrieveError{ErrorCode: "invalid_client", Response: &http.Response{StatusCode: 401}},
			gmailFaultMisconfigured,
			"GOOGLE_CLIENT_ID",
		},
		{
			"gmail api not enabled",
			&googleapi.Error{Code: 403, Message: "Gmail API has not been used in project 123 before or it is disabled"},
			gmailFaultMisconfigured,
			"not enabled",
		},
		{
			"gmail rejects an access token",
			&googleapi.Error{Code: http.StatusUnauthorized, Message: "Unauthorized"},
			gmailFaultReconnect,
			"access token",
		},
	}

	for _, tc := range cases {
		fault, reason := classifyGmailError(tc.err)
		if fault != tc.want {
			t.Errorf("%s: fault=%v want %v (reason %q)", tc.name, fault, tc.want, reason)
		}
		if !strings.Contains(reason, tc.wantText) {
			t.Errorf("%s: reason %q does not mention %q", tc.name, reason, tc.wantText)
		}
	}
}

func TestGmailErrorSummaryDoesNotLeakTokens(t *testing.T) {
	err := errors.New("oauth2: cannot fetch token: 400 response: {\"error\":\"invalid_grant\",\"access_token\":\"secret-token\"}")
	summary := gmailErrorSummary(err)
	if strings.Contains(summary, "secret-token") {
		t.Fatalf("token material leaked into the error summary: %q", summary)
	}
}

func TestPersistingTokenSourceWritesBackRefreshedGrant(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "fresh-access",
			"token_type":    "Bearer",
			"expires_in":    3599,
			"refresh_token": "rotated-refresh",
		})
	}))
	defer server.Close()

	svc, tokens := newGmailTestService(t, server.URL)
	granted := time.Now().Add(-time.Hour)
	tokens.tokensByUser[9] = &models.GmailToken{
		UserID: 9, AccessToken: "stale-access", RefreshToken: "old-refresh", Expiry: granted,
	}

	source := &persistingTokenSource{
		base:    svc.oauthConfig.TokenSource(context.Background(), storedToken(tokens.tokensByUser[9])),
		userID:  9,
		store:   tokens,
		current: *tokens.tokensByUser[9],
	}

	token, err := source.Token()
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if token.AccessToken != "fresh-access" {
		t.Fatalf("expected the refreshed access token, got %q", token.AccessToken)
	}
	if len(tokens.saved) != 1 {
		t.Fatalf("expected exactly one write-back, got %d", len(tokens.saved))
	}
	stored := tokens.tokensByUser[9]
	if stored.AccessToken != "fresh-access" || stored.RefreshToken != "rotated-refresh" {
		t.Fatalf("refreshed grant was not persisted: %+v", stored)
	}
	if time.Now().Add(30 * time.Minute).After(stored.Expiry) {
		t.Fatalf("expected the new expiry to be persisted, got %v", stored.Expiry)
	}
}

func TestPersistingTokenSourceKeepsRefreshTokenWhenGoogleOmitsIt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "fresh-access",
			"token_type":   "Bearer",
			"expires_in":   3599,
		})
	}))
	defer server.Close()

	svc, tokens := newGmailTestService(t, server.URL)
	tokens.tokensByUser[9] = &models.GmailToken{
		UserID: 9, AccessToken: "stale-access", RefreshToken: "keep-me",
		Expiry: time.Now().Add(-time.Hour),
	}

	source := &persistingTokenSource{
		base:    svc.oauthConfig.TokenSource(context.Background(), storedToken(tokens.tokensByUser[9])),
		userID:  9,
		store:   tokens,
		current: *tokens.tokensByUser[9],
	}
	if _, err := source.Token(); err != nil {
		t.Fatalf("refresh failed: %v", err)
	}

	// oauth2 keeps the previous refresh token when Google's response omits it, and the
	// write-back must carry that value forward rather than storing a blank.
	if len(tokens.saved) != 1 {
		t.Fatalf("expected exactly one write-back, got %d", len(tokens.saved))
	}
	if tokens.saved[0].RefreshToken != "keep-me" {
		t.Fatalf("expected the refresh token to survive the write-back, got %q", tokens.saved[0].RefreshToken)
	}
	if merged := mergeRefreshToken(tokens.saved[0].RefreshToken, "keep-me"); merged != "keep-me" {
		t.Fatalf("the stored refresh token would be lost: %q", merged)
	}
}

func TestPersistingTokenSourceSkipsUnchangedToken(t *testing.T) {
	_, tokens := newGmailTestService(t, "https://oauth2.example.test/token")
	live := &models.GmailToken{
		UserID: 9, AccessToken: "access", RefreshToken: "refresh",
		Expiry: time.Now().Add(time.Hour),
	}
	tokens.tokensByUser[9] = live

	source := &persistingTokenSource{
		base:    oauth2.StaticTokenSource(storedToken(live)),
		userID:  9,
		store:   tokens,
		current: *live,
	}
	if _, err := source.Token(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens.saved) != 0 {
		t.Fatalf("a token that did not change must not be written back, got %d writes", len(tokens.saved))
	}
}

func TestPollUserMarksReconnectInsteadOfRetrying(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid_grant"})
	}))
	defer server.Close()

	svc, tokens := newGmailTestService(t, server.URL)
	grant := &models.GmailToken{
		UserID: 11, AccessToken: "expired-access", RefreshToken: "revoked-refresh",
		Expiry: time.Now().Add(-time.Hour),
	}
	tokens.tokensByUser[11] = grant

	processed, err := svc.pollUser(grant)
	if !errors.Is(err, ErrGmailReconnect) {
		t.Fatalf("expected the reconnect sentinel, got %v", err)
	}
	if processed != 0 {
		t.Fatalf("expected nothing processed, got %d", processed)
	}
	if !grant.NeedsReconnect {
		t.Fatal("the dead grant must be persisted so the poller stops retrying it")
	}
	if len(tokens.reconnectMark) != 1 || tokens.reconnectMark[11] == "" {
		t.Fatalf("expected a persisted reconnect reason, got %v", tokens.reconnectMark)
	}
}

func TestPollUserReportsMissingRefreshTokenAsReconnect(t *testing.T) {
	svc, tokens := newGmailTestService(t, "https://oauth2.example.test/token")
	grant := &models.GmailToken{
		UserID: 12, AccessToken: "expired-access", Expiry: time.Now().Add(-time.Minute),
	}
	tokens.tokensByUser[12] = grant

	_, err := svc.pollUser(grant)
	if !errors.Is(err, ErrGmailReconnect) {
		t.Fatalf("expected the reconnect sentinel, got %v", err)
	}
	if reason, ok := tokens.reconnectMark[12]; !ok || !strings.Contains(reason, "refresh token") {
		t.Fatalf("expected the missing refresh token to be recorded, got %q", reason)
	}
}

// A rejected OAuth client is a deployment mistake, not a dead grant: the stored
// authorization must survive untouched so the poller works again once the server
// configuration is fixed, and the user must not be told to reauthorize.
func TestPollUserSeparatesMisconfiguredClientFromDeadGrant(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": "invalid_client", "error_description": "Unauthorized client or client secret incorrect",
		})
	}))
	defer server.Close()

	svc, tokens := newGmailTestService(t, server.URL)
	grant := &models.GmailToken{
		UserID: 14, AccessToken: "expired-access", RefreshToken: "still-good-refresh",
		Expiry: time.Now().Add(-time.Minute),
	}
	tokens.tokensByUser[14] = grant

	_, err := svc.pollUser(grant)
	if !errors.Is(err, ErrGmailMisconfigured) {
		t.Fatalf("expected the misconfigured sentinel, got %v", err)
	}
	if grant.NeedsReconnect {
		t.Fatal("a server configuration error must not disable the user's grant")
	}
	if len(tokens.reconnectMark) != 0 {
		t.Fatalf("expected no persisted reconnect state, got %v", tokens.reconnectMark)
	}
	if strings.Contains(err.Error(), "still-good-refresh") || strings.Contains(err.Error(), "expired-access") {
		t.Fatalf("token material leaked into the error: %v", err)
	}
}

func TestPollUserKeepsTransientErrorsRetryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"temporarily_unavailable"}`))
	}))
	defer server.Close()

	svc, tokens := newGmailTestService(t, server.URL)
	grant := &models.GmailToken{
		UserID: 13, AccessToken: "expired-access", RefreshToken: "good-refresh",
		Expiry: time.Now().Add(-time.Minute),
	}
	tokens.tokensByUser[13] = grant

	_, err := svc.pollUser(grant)
	if err == nil {
		t.Fatal("expected the failing token endpoint to surface")
	}
	if errors.Is(err, ErrGmailReconnect) {
		t.Fatalf("a transient failure must not be reported as a reconnect: %v", err)
	}
	if len(tokens.reconnectMark) != 0 {
		t.Fatalf("transient failures must not persist reconnect state: %v", tokens.reconnectMark)
	}
	if grant.NeedsReconnect {
		t.Fatal("the grant must stay pollable after a transient failure")
	}
}

func TestConnectionStatus(t *testing.T) {
	cases := []struct {
		name             string
		grant            *models.GmailToken
		lookupErr        error
		wantConnected    bool
		wantReconnect    bool
		wantReasonHas    string
		wantHasRefreshed bool
	}{
		{
			name:          "never connected",
			lookupErr:     gorm.ErrRecordNotFound,
			wantReasonHas: "not connected",
		},
		{
			name:             "healthy grant",
			grant:            &models.GmailToken{UserID: 1, AccessToken: "a", RefreshToken: "r", Expiry: time.Now().Add(time.Hour)},
			wantConnected:    true,
			wantHasRefreshed: true,
		},
		{
			name:             "revoked grant",
			grant:            &models.GmailToken{UserID: 1, AccessToken: "a", RefreshToken: "r", NeedsReconnect: true, ReconnectReason: "Google refused the grant"},
			wantReconnect:    true,
			wantReasonHas:    "refused",
			wantHasRefreshed: true,
		},
		{
			name:          "grant without refresh token",
			grant:         &models.GmailToken{UserID: 1, AccessToken: "a"},
			wantReconnect: true,
			wantReasonHas: "refresh token",
		},
		{
			name:          "cleared grant",
			grant:         &models.GmailToken{UserID: 1, AccessToken: ""},
			wantReasonHas: "not connected",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, tokens := newGmailTestService(t, "https://oauth2.example.test/token")
			tokens.lookupErr = tc.lookupErr
			if tc.grant != nil {
				tokens.tokensByUser[tc.grant.UserID] = tc.grant
			}

			status, err := svc.ConnectionStatus(1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if status.Connected != tc.wantConnected {
				t.Errorf("connected=%v want %v", status.Connected, tc.wantConnected)
			}
			if status.NeedsReconnect != tc.wantReconnect {
				t.Errorf("needsReconnect=%v want %v", status.NeedsReconnect, tc.wantReconnect)
			}
			if status.HasRefreshToken != tc.wantHasRefreshed {
				t.Errorf("hasRefreshToken=%v want %v", status.HasRefreshToken, tc.wantHasRefreshed)
			}
			if tc.wantReasonHas != "" && !strings.Contains(status.Reason, tc.wantReasonHas) {
				t.Errorf("reason %q does not mention %q", status.Reason, tc.wantReasonHas)
			}
		})
	}
}

func TestConnectionStatusPropagatesDatabaseError(t *testing.T) {
	svc, tokens := newGmailTestService(t, "https://oauth2.example.test/token")
	tokens.lookupErr = errors.New("connection reset")

	if _, err := svc.ConnectionStatus(1); err == nil {
		t.Fatal("expected the database error to surface")
	}
}

func TestPersistGmailGrantAndDisconnect(t *testing.T) {
	svc, tokens := newGmailTestService(t, "https://oauth2.example.test/token")

	err := svc.PersistGmailGrant(5, &oauth2.Token{
		AccessToken:  "access-5",
		RefreshToken: "refresh-5",
		Expiry:       time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens.tokensByUser[5] == nil || tokens.tokensByUser[5].AccessToken != "access-5" {
		t.Fatalf("grant was not stored for the right user: %+v", tokens.tokensByUser[5])
	}

	status, err := svc.ConnectionStatus(5)
	if err != nil || !status.Connected {
		t.Fatalf("expected a connected account, got %+v err=%v", status, err)
	}

	if err := svc.DisconnectGmail(5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens.cleared) != 1 || tokens.cleared[0] != 5 {
		t.Fatalf("expected user 5's grant to be cleared, got %v", tokens.cleared)
	}
	status, err = svc.ConnectionStatus(5)
	if err != nil || status.Connected {
		t.Fatalf("a disconnected account must not report connected: %+v err=%v", status, err)
	}
}

func TestRecentEmailsClampsLimit(t *testing.T) {
	svc, _, events, _ := newSyncTestService(t)

	if _, err := svc.RecentEmails(1, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if events.recentLimit != 10 {
		t.Fatalf("expected the default limit, got %d", events.recentLimit)
	}

	if _, err := svc.RecentEmails(1, 500); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if events.recentLimit != 50 {
		t.Fatalf("expected the capped limit, got %d", events.recentLimit)
	}
	if events.recentCalls != 2 {
		t.Fatalf("expected two listing calls, got %d", events.recentCalls)
	}
}

func TestOAuthReturnURL(t *testing.T) {
	t.Setenv("FRONTEND_URL", "http://localhost:5173/")

	connected := OAuthReturnURL(true, "")
	if !strings.HasPrefix(connected, "http://localhost:5173/candidate?gmail=connected") {
		t.Fatalf("unexpected redirect: %q", connected)
	}

	failed := OAuthReturnURL(false, "Gmail connection was cancelled.")
	if !strings.Contains(failed, "gmail=error") || !strings.Contains(failed, "reason=") {
		t.Fatalf("unexpected failure redirect: %q", failed)
	}
	if strings.ContainsAny(failed, "\n\r") {
		t.Fatal("redirect target must not contain header separators")
	}
}

func TestOAuthFailureReason(t *testing.T) {
	if got := OAuthFailureReason("access_denied"); !strings.Contains(got, "cancelled") {
		t.Fatalf("unexpected message: %q", got)
	}
	if got := OAuthFailureReason("redirect_uri_mismatch"); !strings.Contains(got, "redirect URI") {
		t.Fatalf("unexpected message: %q", got)
	}
	if got := OAuthFailureReason("whatever"); got == "" {
		t.Fatal("unknown codes still need a message")
	}
}
