package service

import (
	"os"
	"path/filepath"
	"testing"
)

// stubConsentStore is a test double for ImplementConsentStore.
type stubConsentStore struct {
	hasConsent bool
	recorded   bool
}

func (s *stubConsentStore) HasAgentPermissionsConsent() bool { return s.hasConsent }
func (s *stubConsentStore) RecordAgentPermissionsConsent() error {
	s.recorded = true
	s.hasConsent = true
	return nil
}

func TestImplementService_ValidateTrack(t *testing.T) {
	t.Parallel()

	// Create a temp project dir with kf tracks.yaml.
	projectDir := t.TempDir()
	kfDir := filepath.Join(projectDir, ".agent", "kf")
	if err := os.MkdirAll(kfDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tracksContent := `pending-track_123Z: {"title":"Pending Track","status":"pending","type":"feature","created":"2026-03-10","updated":"2026-03-10"}
complete-track_456Z: {"title":"Complete Track","status":"completed","type":"feature","created":"2026-03-10","updated":"2026-03-10"}
inprogress-track_789Z: {"title":"In Progress Track","status":"in-progress","type":"feature","created":"2026-03-10","updated":"2026-03-10"}
`
	if err := os.WriteFile(filepath.Join(kfDir, "tracks.yaml"), []byte(tracksContent), 0o644); err != nil {
		t.Fatal(err)
	}

	consent := &stubConsentStore{}
	svc := NewImplementService(consent, nil, t.TempDir(), "")

	t.Run("pending track is valid", func(t *testing.T) {
		entry, err := svc.ValidateTrack(projectDir, "pending-track_123Z")
		if err != nil {
			t.Fatalf("ValidateTrack: %v", err)
		}
		if entry.ID != "pending-track_123Z" {
			t.Errorf("ID = %q, want %q", entry.ID, "pending-track_123Z")
		}
		if entry.Title != "Pending Track" {
			t.Errorf("Title = %q, want %q", entry.Title, "Pending Track")
		}
	})

	t.Run("complete track returns error", func(t *testing.T) {
		_, err := svc.ValidateTrack(projectDir, "complete-track_456Z")
		if err == nil {
			t.Fatal("expected error for complete track")
		}
		var e *TrackAlreadyCompleteError
		if !isErrorType[TrackAlreadyCompleteError](err) {
			t.Errorf("got %T, want *TrackAlreadyCompleteError", err)
		}
		_ = e
	})

	t.Run("in-progress track returns error", func(t *testing.T) {
		_, err := svc.ValidateTrack(projectDir, "inprogress-track_789Z")
		if err == nil {
			t.Fatal("expected error for in-progress track")
		}
		if !isErrorType[TrackInProgressError](err) {
			t.Errorf("got %T, want *TrackInProgressError", err)
		}
	})

	t.Run("nonexistent track returns error", func(t *testing.T) {
		_, err := svc.ValidateTrack(projectDir, "nonexistent-track")
		if err == nil {
			t.Fatal("expected error for missing track")
		}
		if !isErrorType[TrackNotFoundError](err) {
			t.Errorf("got %T, want *TrackNotFoundError", err)
		}
	})
}

func TestImplementService_Consent(t *testing.T) {
	t.Parallel()

	consent := &stubConsentStore{hasConsent: false}
	svc := NewImplementService(consent, nil, t.TempDir(), "")

	if svc.HasConsent() {
		t.Error("HasConsent should be false initially")
	}

	if err := svc.RecordConsent(); err != nil {
		t.Fatalf("RecordConsent: %v", err)
	}

	if !svc.HasConsent() {
		t.Error("HasConsent should be true after recording")
	}
	if !consent.recorded {
		t.Error("consent store should have recorded consent")
	}
}

func TestImplementService_ListPendingTracks(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	kfDir := filepath.Join(projectDir, ".agent", "kf")
	if err := os.MkdirAll(kfDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tracksContent := `complete-1_456Z: {"title":"Complete One","status":"completed","type":"feature","created":"2026-03-10","updated":"2026-03-10"}
pending-1_123Z: {"title":"First Pending","status":"pending","type":"feature","created":"2026-03-10","updated":"2026-03-10"}
pending-2_789Z: {"title":"Second Pending","status":"pending","type":"feature","created":"2026-03-10","updated":"2026-03-10"}
`
	if err := os.WriteFile(filepath.Join(kfDir, "tracks.yaml"), []byte(tracksContent), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := NewImplementService(&stubConsentStore{}, nil, t.TempDir(), "")

	pending, err := svc.ListPendingTracks(projectDir)
	if err != nil {
		t.Fatalf("ListPendingTracks: %v", err)
	}
	if len(pending) != 2 {
		t.Fatalf("got %d pending tracks, want 2", len(pending))
	}
	if pending[0].ID != "pending-1_123Z" {
		t.Errorf("first pending ID = %q, want %q", pending[0].ID, "pending-1_123Z")
	}
}

func TestImplementService_LogDir(t *testing.T) {
	t.Parallel()

	svc := NewImplementService(nil, nil, "/data", "")
	got := svc.LogDir("myproject")
	want := filepath.Join("/data", "projects", "myproject", "logs")
	if got != want {
		t.Errorf("LogDir = %q, want %q", got, want)
	}
}

// isErrorType checks if an error is of type *T using errors.As.
func isErrorType[T any](err error) bool {
	var target *T
	return asError(err, &target)
}

// asError wraps errors.As to avoid import in each test.
func asError(err error, target interface{}) bool {
	// Manual type assertion since we can't import errors in a generic helper easily.
	switch t := target.(type) {
	case **TrackNotFoundError:
		e, ok := err.(*TrackNotFoundError)
		if ok {
			*t = e
		}
		return ok
	case **TrackAlreadyCompleteError:
		e, ok := err.(*TrackAlreadyCompleteError)
		if ok {
			*t = e
		}
		return ok
	case **TrackInProgressError:
		e, ok := err.(*TrackInProgressError)
		if ok {
			*t = e
		}
		return ok
	}
	return false
}
