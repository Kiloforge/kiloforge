package lock

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func setupTestHandler() (*Handler, *http.ServeMux) {
	m := New("")
	ctx, cancel := context.WithCancel(context.Background())
	m.StartReaper(ctx)
	_ = cancel // tests are short-lived
	h := NewHandler(m)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return h, mux
}

func postJSON(mux http.Handler, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func deleteJSON(mux http.Handler, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodDelete, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func getJSON(mux http.Handler, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestHandler_AcquireSuccess(t *testing.T) {
	_, mux := setupTestHandler()

	w := postJSON(mux, "/-/api/locks/merge/acquire", acquireRequest{
		Holder:     "dev-1",
		TTLSeconds: 60,
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp lockResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Scope != "merge" || resp.Holder != "dev-1" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestHandler_AcquireTimeout(t *testing.T) {
	_, mux := setupTestHandler()

	// Acquire first.
	postJSON(mux, "/-/api/locks/merge/acquire", acquireRequest{
		Holder:     "dev-1",
		TTLSeconds: 60,
	})

	// Second acquire with very short timeout.
	w := postJSON(mux, "/-/api/locks/merge/acquire", acquireRequest{
		Holder:         "dev-2",
		TTLSeconds:     60,
		TimeoutSeconds: 1,
	})

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_AcquireMissingHolder(t *testing.T) {
	_, mux := setupTestHandler()

	w := postJSON(mux, "/-/api/locks/merge/acquire", acquireRequest{TTLSeconds: 60})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandler_ReleaseByHolder(t *testing.T) {
	_, mux := setupTestHandler()

	postJSON(mux, "/-/api/locks/merge/acquire", acquireRequest{
		Holder:     "dev-1",
		TTLSeconds: 60,
	})

	w := deleteJSON(mux, "/-/api/locks/merge", releaseRequest{Holder: "dev-1"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_ReleaseByNonHolder(t *testing.T) {
	_, mux := setupTestHandler()

	postJSON(mux, "/-/api/locks/merge/acquire", acquireRequest{
		Holder:     "dev-1",
		TTLSeconds: 60,
	})

	w := deleteJSON(mux, "/-/api/locks/merge", releaseRequest{Holder: "dev-2"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_Heartbeat(t *testing.T) {
	_, mux := setupTestHandler()

	postJSON(mux, "/-/api/locks/merge/acquire", acquireRequest{
		Holder:     "dev-1",
		TTLSeconds: 5,
	})

	w := postJSON(mux, "/-/api/locks/merge/heartbeat", heartbeatRequest{
		Holder:     "dev-1",
		TTLSeconds: 60,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp lockResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.TTLRemainingSeconds < 50 {
		t.Errorf("heartbeat did not extend TTL: remaining=%.1f", resp.TTLRemainingSeconds)
	}
}

func TestHandler_HeartbeatNonHolder(t *testing.T) {
	_, mux := setupTestHandler()

	postJSON(mux, "/-/api/locks/merge/acquire", acquireRequest{
		Holder:     "dev-1",
		TTLSeconds: 60,
	})

	w := postJSON(mux, "/-/api/locks/merge/heartbeat", heartbeatRequest{
		Holder:     "dev-2",
		TTLSeconds: 60,
	})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandler_ListEmpty(t *testing.T) {
	_, mux := setupTestHandler()

	w := getJSON(mux, "/-/api/locks")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp []lockResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 0 {
		t.Errorf("expected empty list, got %d", len(resp))
	}
}

func TestHandler_ListMultiple(t *testing.T) {
	_, mux := setupTestHandler()

	postJSON(mux, "/-/api/locks/merge/acquire", acquireRequest{Holder: "dev-1", TTLSeconds: 60})
	postJSON(mux, "/-/api/locks/deploy/acquire", acquireRequest{Holder: "dev-2", TTLSeconds: 60})

	w := getJSON(mux, "/-/api/locks")
	var resp []lockResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 2 {
		t.Errorf("expected 2 locks, got %d", len(resp))
	}
}

func TestHandler_AcquireAfterRelease(t *testing.T) {
	_, mux := setupTestHandler()

	postJSON(mux, "/-/api/locks/merge/acquire", acquireRequest{Holder: "dev-1", TTLSeconds: 60})
	deleteJSON(mux, "/-/api/locks/merge", releaseRequest{Holder: "dev-1"})

	w := postJSON(mux, "/-/api/locks/merge/acquire", acquireRequest{Holder: "dev-2", TTLSeconds: 60})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 after release, got %d", w.Code)
	}

	var resp lockResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Holder != "dev-2" {
		t.Errorf("expected holder dev-2, got %s", resp.Holder)
	}
}

func TestHandler_TTLExpiryViaHTTP(t *testing.T) {
	m := New("")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.StartReaper(ctx)
	h := NewHandler(m)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Acquire with very short TTL.
	postJSON(mux, "/-/api/locks/merge/acquire", acquireRequest{
		Holder:     "dev-1",
		TTLSeconds: 1,
	})

	// Wait for expiry.
	time.Sleep(1500 * time.Millisecond)

	// Should be free now.
	w := postJSON(mux, "/-/api/locks/merge/acquire", acquireRequest{
		Holder:     "dev-2",
		TTLSeconds: 60,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 after TTL expiry, got %d: %s", w.Code, w.Body.String())
	}
}
