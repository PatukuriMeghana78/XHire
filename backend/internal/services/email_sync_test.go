package services

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"job_application_analysis/internal/models"

	"gorm.io/gorm"
)

type fakeTokenStore struct {
	tokensByUser  map[uint]*models.GmailToken
	lookupErr     error
	allErr        error
	saveErr       error
	requested     []uint
	saved         []*models.GmailToken
	reconnectMark map[uint]string
	cleared       []uint
}

func (f *fakeTokenStore) grantForUser(userID uint) (*models.GmailToken, error) {
	f.requested = append(f.requested, userID)
	if f.lookupErr != nil {
		return nil, f.lookupErr
	}
	if token, ok := f.tokensByUser[userID]; ok {
		return token, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeTokenStore) allGrants() ([]models.GmailToken, error) {
	if f.allErr != nil {
		return nil, f.allErr
	}
	tokens := make([]models.GmailToken, 0, len(f.tokensByUser))
	for _, token := range f.tokensByUser {
		tokens = append(tokens, *token)
	}
	return tokens, nil
}

func (f *fakeTokenStore) saveGrant(token *models.GmailToken) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = append(f.saved, token)
	stored := *token
	f.tokensByUser[token.UserID] = &stored
	return nil
}

func (f *fakeTokenStore) markNeedsReconnect(userID uint, reason string) error {
	if f.reconnectMark == nil {
		f.reconnectMark = map[uint]string{}
	}
	f.reconnectMark[userID] = reason
	if token, ok := f.tokensByUser[userID]; ok {
		token.NeedsReconnect = true
		token.ReconnectReason = reason
	}
	return nil
}

func (f *fakeTokenStore) clearGrant(userID uint) error {
	f.cleared = append(f.cleared, userID)
	if token, ok := f.tokensByUser[userID]; ok {
		token.AccessToken = ""
		token.RefreshToken = ""
		token.NeedsReconnect = false
	}
	return nil
}

type fakeEventStore struct {
	exists       bool
	existsErr    error
	createErr    error
	created      []*models.EmailEvent
	lookupCalls  int
	updateCalls  int
	pendingCalls int
	recentCalls  int
	recentLimit  int
}

func (f *fakeEventStore) Create(event *models.EmailEvent) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = append(f.created, event)
	return nil
}

func (f *fakeEventStore) FindByID(id uint) (*models.EmailEvent, error) {
	return &models.EmailEvent{Model: gorm.Model{ID: id}}, nil
}

func (f *fakeEventStore) FindPendingByUserID(userID uint) ([]models.EmailEvent, error) {
	f.pendingCalls++
	return []models.EmailEvent{}, nil
}

func (f *fakeEventStore) FindRecentByUserID(userID uint, limit int) ([]models.EmailEvent, error) {
	f.recentCalls++
	f.recentLimit = limit
	return []models.EmailEvent{}, nil
}

func (f *fakeEventStore) ExistsByEmailID(userID uint, emailID string) (bool, error) {
	f.lookupCalls++
	return f.exists, f.existsErr
}

func (f *fakeEventStore) Update(event *models.EmailEvent) error {
	f.updateCalls++
	return nil
}

type fakeNotificationStore struct {
	created []*models.Notification
}

func (f *fakeNotificationStore) Create(notification *models.Notification) error {
	f.created = append(f.created, notification)
	return nil
}

// newSyncTestService builds an EmailService with every external dependency faked: no
// database, no Gmail and no NLP service (classification therefore takes the
// documented heuristic fallback).
func newSyncTestService(t *testing.T) (*EmailService, *fakeTokenStore, *fakeEventStore, *fakeNotificationStore) {
	t.Helper()

	tokens := &fakeTokenStore{tokensByUser: map[uint]*models.GmailToken{}}
	events := &fakeEventStore{}
	notifications := &fakeNotificationStore{}

	unreachable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	unreachable.Close()

	svc := &EmailService{
		emailRepo:        events,
		notificationRepo: notifications,
		tokens:           tokens,
		nlpURL:           unreachable.URL,
	}
	return svc, tokens, events, notifications
}

// recordingPoller replaces the Gmail-facing poll step with a recorder and returns an
// accessor for what it saw.
func recordingPoller(svc *EmailService, result int) func() []*models.GmailToken {
	var seenMu sync.Mutex
	seen := make([]*models.GmailToken, 0, 4)
	svc.pollMailbox = func(token *models.GmailToken) (int, error) {
		seenMu.Lock()
		seen = append(seen, token)
		seenMu.Unlock()
		return result, nil
	}
	return func() []*models.GmailToken {
		seenMu.Lock()
		defer seenMu.Unlock()
		return append([]*models.GmailToken(nil), seen...)
	}
}

func TestSyncUserEmails_PollsOnlyRequestedMailbox(t *testing.T) {
	svc, tokens, _, _ := newSyncTestService(t)
	tokens.tokensByUser[1] = &models.GmailToken{UserID: 1, AccessToken: "token-user-1"}
	tokens.tokensByUser[2] = &models.GmailToken{UserID: 2, AccessToken: "token-user-2"}

	recorded := recordingPoller(svc, 1)

	count, err := svc.SyncUserEmails(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected the real processed count, got %d", count)
	}
	if len(tokens.requested) != 1 || tokens.requested[0] != 2 {
		t.Fatalf("expected a token lookup for user 2 only, got %v", tokens.requested)
	}

	seen := recorded()
	if len(seen) != 1 || seen[0].UserID != 2 || seen[0].AccessToken != "token-user-2" {
		t.Fatalf("expected only user 2's mailbox to be polled, got %+v", seen)
	}
}

func TestSyncUserEmails_RequiresConnectedAccount(t *testing.T) {
	svc, _, _, _ := newSyncTestService(t)
	polled := false
	svc.pollMailbox = func(token *models.GmailToken) (int, error) {
		polled = true
		return 0, nil
	}

	count, err := svc.SyncUserEmails(5)
	if !errors.Is(err, ErrGmailNotConnected) {
		t.Fatalf("expected the not-connected sentinel, got %v", err)
	}
	if count != 0 {
		t.Fatalf("expected zero processed on failure, got %d", count)
	}
	if polled {
		t.Fatal("must not poll any mailbox when the account is not connected")
	}
}

func TestSyncUserEmails_PropagatesDatabaseError(t *testing.T) {
	svc, tokens, _, _ := newSyncTestService(t)
	tokens.lookupErr = errors.New("connection reset")
	svc.pollMailbox = func(token *models.GmailToken) (int, error) { return 7, nil }

	count, err := svc.SyncUserEmails(1)
	if err == nil {
		t.Fatal("expected the database error to surface")
	}
	if count != 0 {
		t.Fatalf("expected zero processed on failure, got %d", count)
	}
}

func TestSyncUserEmails_RefusesToRetryADeadGrant(t *testing.T) {
	svc, tokens, _, _ := newSyncTestService(t)
	tokens.tokensByUser[3] = &models.GmailToken{
		UserID: 3, AccessToken: "expired-access", NeedsReconnect: true,
	}
	polled := false
	svc.pollMailbox = func(token *models.GmailToken) (int, error) {
		polled = true
		return 4, nil
	}

	count, err := svc.SyncUserEmails(3)
	if !errors.Is(err, ErrGmailReconnect) {
		t.Fatalf("expected the reconnect sentinel, got %v", err)
	}
	if count != 0 {
		t.Fatalf("expected zero processed for a dead grant, got %d", count)
	}
	if polled {
		t.Fatal("a grant Google already refused must not be retried")
	}
}

func TestSyncUserEmails_PropagatesPollError(t *testing.T) {
	svc, tokens, _, _ := newSyncTestService(t)
	tokens.tokensByUser[4] = &models.GmailToken{UserID: 4, AccessToken: "access", RefreshToken: "refresh"}
	failure := errors.New("gmail fetch: temporary failure")
	svc.pollMailbox = func(token *models.GmailToken) (int, error) { return 0, failure }

	count, err := svc.SyncUserEmails(4)
	if !errors.Is(err, failure) {
		t.Fatalf("expected the poll error to reach the caller, got %v", err)
	}
	if count != 0 {
		t.Fatalf("expected zero processed, got %d", count)
	}
}

func TestPollUnreadEmails_CoversEveryConnectedAccount(t *testing.T) {
	svc, tokens, _, _ := newSyncTestService(t)
	tokens.tokensByUser[1] = &models.GmailToken{UserID: 1, AccessToken: "a1"}
	tokens.tokensByUser[2] = &models.GmailToken{UserID: 2, AccessToken: "a2"}
	tokens.tokensByUser[3] = &models.GmailToken{UserID: 3, AccessToken: "a3"}

	recorded := recordingPoller(svc, 1)
	svc.PollUnreadEmails()

	if seen := recorded(); len(seen) != 3 {
		t.Fatalf("expected all three mailboxes polled, got %d", len(seen))
	}
}

func TestPollUnreadEmails_SkipsDeadAndEmptyGrants(t *testing.T) {
	svc, tokens, _, _ := newSyncTestService(t)
	tokens.tokensByUser[1] = &models.GmailToken{UserID: 1, AccessToken: "live"}
	tokens.tokensByUser[2] = &models.GmailToken{UserID: 2, AccessToken: "revoked", NeedsReconnect: true}
	tokens.tokensByUser[3] = &models.GmailToken{UserID: 3, AccessToken: ""}

	recorded := recordingPoller(svc, 0)
	svc.PollUnreadEmails()

	seen := recorded()
	if len(seen) != 1 || seen[0].UserID != 1 {
		t.Fatalf("expected only the live grant to be polled, got %+v", seen)
	}
}

func TestPollUnreadEmails_PropagatesDatabaseError(t *testing.T) {
	svc, tokens, _, _ := newSyncTestService(t)
	tokens.allErr = errors.New("connection reset")
	polled := false
	svc.pollMailbox = func(token *models.GmailToken) (int, error) {
		polled = true
		return 0, nil
	}

	svc.PollUnreadEmails()
	if polled {
		t.Fatal("expected no polling after a token listing failure")
	}
}

func TestManualSyncAndPoller_DoNotOverlapForOneUser(t *testing.T) {
	svc, tokens, _, _ := newSyncTestService(t)
	tokens.tokensByUser[7] = &models.GmailToken{UserID: 7, AccessToken: "access"}

	var mu sync.Mutex
	running := 0
	maxRunning := 0
	done := make(chan struct{}, 2)

	svc.pollMailbox = func(token *models.GmailToken) (int, error) {
		mu.Lock()
		running++
		if running > maxRunning {
			maxRunning = running
		}
		mu.Unlock()

		time.Sleep(20 * time.Millisecond)

		mu.Lock()
		running--
		mu.Unlock()
		done <- struct{}{}
		return 1, nil
	}

	go func() { svc.PollUnreadEmails() }()
	go func() { _, _ = svc.SyncUserEmails(7) }()

	<-done
	<-done

	if maxRunning != 1 {
		t.Fatalf("manual sync and background polling overlapped on one mailbox (max %d)", maxRunning)
	}
}

func TestRecordEvent_StoresOnceAndSkipsDuplicates(t *testing.T) {
	svc, _, events, notifications := newSyncTestService(t)

	created, err := svc.recordEvent(1, "msg-1", "Your application at Acme", "we have received your resume")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Fatal("expected the first record to report created=true")
	}

	events.exists = true
	created, err = svc.recordEvent(1, "msg-1", "Your application at Acme", "we have received your resume")
	if err != nil {
		t.Fatalf("unexpected error on duplicate: %v", err)
	}
	if created {
		t.Fatal("expected a re-polled message to report created=false")
	}

	if len(events.created) != 1 {
		t.Fatalf("expected exactly one stored event, got %d", len(events.created))
	}
	if len(notifications.created) != 1 {
		t.Fatalf("expected exactly one notification, got %d", len(notifications.created))
	}
	if events.created[0].UserID != 1 || events.created[0].EmailID != "msg-1" {
		t.Fatalf("event not scoped to the owning user: %+v", events.created[0])
	}
	if events.created[0].ClassificationSource != "heuristic" {
		t.Fatalf("expected the fallback to be marked, got %q", events.created[0].ClassificationSource)
	}
}

func TestRecordEvent_LookupErrorWritesNothing(t *testing.T) {
	svc, _, events, notifications := newSyncTestService(t)
	events.existsErr = errors.New("connection reset")

	created, err := svc.recordEvent(1, "msg-9", "Acme update", "body")
	if err == nil {
		t.Fatal("expected the lookup error to propagate so the message is retried")
	}
	if created {
		t.Fatal("expected created=false when the duplicate check failed")
	}
	if events.lookupCalls != 1 {
		t.Fatalf("expected a single duplicate check, got %d", events.lookupCalls)
	}
	if len(events.created) != 0 {
		t.Fatal("a failed duplicate check must not insert an event")
	}
	if len(notifications.created) != 0 {
		t.Fatal("a failed duplicate check must not notify")
	}
}

func TestRecordEvent_StoreFailureWritesNothing(t *testing.T) {
	svc, _, events, notifications := newSyncTestService(t)
	events.createErr = errors.New("insert failed")

	created, err := svc.recordEvent(1, "msg-3", "Acme update", "body")
	if err == nil {
		t.Fatal("expected the insert error to propagate")
	}
	if created {
		t.Fatal("expected created=false when the insert failed")
	}
	if len(notifications.created) != 0 {
		t.Fatal("expected no notification for an unwritten event")
	}
}

func TestWithUserLock_SerializesSameUser(t *testing.T) {
	svc, _, _, _ := newSyncTestService(t)

	var counterMu sync.Mutex
	inside := 0
	maxInside := 0

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc.withUserLock(7, func() {
				counterMu.Lock()
				inside++
				if inside > maxInside {
					maxInside = inside
				}
				counterMu.Unlock()

				time.Sleep(5 * time.Millisecond)

				counterMu.Lock()
				inside--
				counterMu.Unlock()
			})
		}()
	}
	wg.Wait()

	if maxInside != 1 {
		t.Fatalf("expected the same user's mailbox work to serialize, saw %d concurrent runs", maxInside)
	}
}

func TestWithUserLock_DifferentUsersDoNotBlock(t *testing.T) {
	svc, _, _, _ := newSyncTestService(t)

	entered := make(chan struct{})
	release := make(chan struct{})
	go func() {
		svc.withUserLock(1, func() {
			close(entered)
			<-release
		})
	}()
	<-entered

	done := make(chan struct{})
	go func() {
		svc.withUserLock(2, func() { close(done) })
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("user 2 was blocked by user 1's lock")
	}
	close(release)
}
