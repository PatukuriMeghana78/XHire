package services

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// oauthStateTTL is how long an issued OAuth state remains valid.
const oauthStateTTL = 10 * time.Minute

// oauthStateEntry binds a random state token to the user who initiated the flow.
type oauthStateEntry struct {
	userID    uint
	expiresAt time.Time
}

// OAuthStateStore is an in-memory, single-use, expiring store for OAuth `state`
// values. It replaces the previous predictable user-ID-only state and provides
// CSRF protection for the Gmail authorization callback. It holds no secrets:
// states are random tokens, and only the owning user's ID is stored.
type OAuthStateStore struct {
	mu      sync.Mutex
	entries map[string]oauthStateEntry
	ttl     time.Duration
}

// NewOAuthStateStore creates a store with the given time-to-live for states.
func NewOAuthStateStore(ttl time.Duration) *OAuthStateStore {
	return &OAuthStateStore{
		entries: make(map[string]oauthStateEntry),
		ttl:     ttl,
	}
}

// Issue generates a cryptographically random state bound to userID and stores it.
// Expired entries are pruned opportunistically to bound memory.
func (s *OAuthStateStore) Issue(userID uint) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	state := hex.EncodeToString(buf)

	now := time.Now()
	s.mu.Lock()
	for k, v := range s.entries {
		if now.After(v.expiresAt) {
			delete(s.entries, k)
		}
	}
	s.entries[state] = oauthStateEntry{userID: userID, expiresAt: now.Add(s.ttl)}
	s.mu.Unlock()

	return state, nil
}

// Validate consumes the state if it is present and unexpired, returning the bound
// userID. It is single-use: the entry is always removed on lookup so a state can
// never be replayed, and expired or unknown states return ok=false.
func (s *OAuthStateStore) Validate(state string) (uint, bool) {
	if state == "" {
		return 0, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.entries[state]
	if !ok {
		return 0, false
	}
	delete(s.entries, state)
	if time.Now().After(entry.expiresAt) {
		return 0, false
	}
	return entry.userID, true
}

// IssueOAuthState creates a single-use state bound to the authenticated user.
func (s *EmailService) IssueOAuthState(userID uint) (string, error) {
	return s.stateStore.Issue(userID)
}

// ValidateOAuthState consumes and validates a callback state, returning the bound
// userID. Returns ok=false for missing, unknown, expired, or replayed states.
func (s *EmailService) ValidateOAuthState(state string) (uint, bool) {
	return s.stateStore.Validate(state)
}
