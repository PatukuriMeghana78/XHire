package services

import (
	"testing"
	"time"
)

func TestOAuthState_Valid(t *testing.T) {
	store := NewOAuthStateStore(time.Minute)
	state, err := store.Issue(42)
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}
	if len(state) != 64 { // 32 bytes hex-encoded
		t.Fatalf("unexpected state length: %d", len(state))
	}
	userID, ok := store.Validate(state)
	if !ok || userID != 42 {
		t.Fatalf("Validate = (%d, %v); want (42, true)", userID, ok)
	}
}

func TestOAuthState_Unique(t *testing.T) {
	store := NewOAuthStateStore(time.Minute)
	a, _ := store.Issue(1)
	b, _ := store.Issue(1)
	if a == b {
		t.Fatal("Issue returned duplicate states; expected cryptographically random values")
	}
}

func TestOAuthState_Invalid(t *testing.T) {
	store := NewOAuthStateStore(time.Minute)
	if _, ok := store.Validate("not-a-real-state"); ok {
		t.Fatal("Validate accepted an unknown state")
	}
}

func TestOAuthState_Empty(t *testing.T) {
	store := NewOAuthStateStore(time.Minute)
	if _, ok := store.Validate(""); ok {
		t.Fatal("Validate accepted an empty state")
	}
}

func TestOAuthState_Expiry(t *testing.T) {
	store := NewOAuthStateStore(5 * time.Millisecond)
	state, _ := store.Issue(7)
	time.Sleep(20 * time.Millisecond)
	if _, ok := store.Validate(state); ok {
		t.Fatal("Validate accepted an expired state")
	}
}

func TestOAuthState_Replay(t *testing.T) {
	store := NewOAuthStateStore(time.Minute)
	state, _ := store.Issue(9)
	if _, ok := store.Validate(state); !ok {
		t.Fatal("first Validate should succeed")
	}
	if _, ok := store.Validate(state); ok {
		t.Fatal("Validate accepted a replayed (already consumed) state")
	}
}

func TestOAuthState_ExpiredNotReplayable(t *testing.T) {
	// An expired state must be consumed too, so it can never validate later.
	store := NewOAuthStateStore(1 * time.Millisecond)
	state, _ := store.Issue(3)
	time.Sleep(10 * time.Millisecond)
	if _, ok := store.Validate(state); ok {
		t.Fatal("expected expired state to be rejected")
	}
}
