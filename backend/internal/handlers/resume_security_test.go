package handlers

import (
	"path/filepath"
	"strings"
	"testing"

	"job_application_analysis/internal/dto"
	"job_application_analysis/internal/models"
)

func TestSafeResumePath_Inside(t *testing.T) {
	stored := filepath.Join(resumeUploadDir, "1_123.pdf")
	abs, ok := safeResumePath(stored)
	if !ok {
		t.Fatalf("expected a valid path inside the upload dir, got ok=false")
	}
	if !filepath.IsAbs(abs) {
		t.Fatalf("expected absolute path, got %q", abs)
	}
}

func TestSafeResumePath_Traversal(t *testing.T) {
	bad := []string{
		"../../etc/passwd",
		filepath.Join(resumeUploadDir, "..", "..", "secret.txt"),
		"/etc/passwd",
		"",
	}
	for _, p := range bad {
		if abs, ok := safeResumePath(p); ok {
			t.Fatalf("safeResumePath(%q) should be rejected, got (%q, true)", p, abs)
		}
	}
}

func TestSanitizeDownloadName(t *testing.T) {
	cases := map[string]string{
		"resume.pdf":        "resume.pdf",
		"../../evil.pdf":    "evil.pdf",
		`bad"name.pdf`:      "badname.pdf",
		"injected\r\nX.pdf": "injectedX.pdf",
		"":                  "resume",
		".":                 "resume",
	}
	for in, want := range cases {
		if got := sanitizeDownloadName(in); got != want {
			t.Fatalf("sanitizeDownloadName(%q) = %q; want %q", in, got, want)
		}
		if strings.ContainsAny(sanitizeDownloadName(in), "\r\n\"") {
			t.Fatalf("sanitizeDownloadName(%q) left header-injection characters", in)
		}
	}
}

func TestResumeOwnedBy(t *testing.T) {
	resume := &models.Resume{CandidateID: 7}
	if !resumeOwnedBy(resume, 7) {
		t.Fatalf("expected the owning candidate to be allowed")
	}
	if resumeOwnedBy(resume, 8) {
		t.Fatalf("expected a different candidate to be refused")
	}
	if resumeOwnedBy(resume, 0) {
		t.Fatalf("expected an empty user id to be refused")
	}
	if resumeOwnedBy(nil, 7) {
		t.Fatalf("expected a missing resume to be refused")
	}
}

func TestValidStatusUpdate(t *testing.T) {
	valid := []string{
		"applied", "Applied", "under-review", "under_review", "review",
		"shortlisted", "interview", "Interview", "rejected", "hired", "Offer",
	}
	for _, s := range valid {
		if !dto.ValidStatusUpdate(s) {
			t.Fatalf("ValidStatusUpdate(%q) = false; want true", s)
		}
	}
	invalid := []string{"", "   ", "Bogus", "delete", "hired;drop", "pending", "unknown-status"}
	for _, s := range invalid {
		if dto.ValidStatusUpdate(s) {
			t.Fatalf("ValidStatusUpdate(%q) = true; want false", s)
		}
	}
}
