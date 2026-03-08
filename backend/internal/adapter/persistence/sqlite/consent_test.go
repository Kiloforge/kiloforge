package sqlite

import (
	"testing"
)

func TestConsentStore_RoundTrip(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewConsentStore(db)

	// Initially not consented.
	if store.HasAgentPermissionsConsent() {
		t.Fatal("expected no consent initially")
	}
	info := store.GetAgentPermissionsConsent()
	if info.Consented {
		t.Fatal("expected Consented=false initially")
	}
	if info.ConsentedAt != "" {
		t.Fatalf("expected empty ConsentedAt, got %q", info.ConsentedAt)
	}

	// Record consent.
	if err := store.RecordAgentPermissionsConsent(); err != nil {
		t.Fatalf("RecordAgentPermissionsConsent: %v", err)
	}

	// Now consented.
	if !store.HasAgentPermissionsConsent() {
		t.Fatal("expected consent after recording")
	}
	info = store.GetAgentPermissionsConsent()
	if !info.Consented {
		t.Fatal("expected Consented=true")
	}
	if info.ConsentedAt == "" {
		t.Fatal("expected non-empty ConsentedAt")
	}
}

func TestConsentStore_Idempotent(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewConsentStore(db)

	// Recording twice should not error.
	if err := store.RecordAgentPermissionsConsent(); err != nil {
		t.Fatalf("first record: %v", err)
	}
	if err := store.RecordAgentPermissionsConsent(); err != nil {
		t.Fatalf("second record: %v", err)
	}
	if !store.HasAgentPermissionsConsent() {
		t.Fatal("expected consent after double record")
	}
}
