package rest

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestLogger(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	handler := RequestLogger(logger)(inner)

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	logOutput := buf.String()
	for _, want := range []string{"GET", "/health", "200"} {
		if !containsCI(logOutput, want) {
			t.Errorf("log output missing %q: %s", want, logOutput)
		}
	}
}

func TestRequestLogger_StatusCapture(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	handler := RequestLogger(logger)(inner)

	req := httptest.NewRequest("POST", "/api/agents", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}

	logOutput := buf.String()
	if !containsCI(logOutput, "404") {
		t.Errorf("log output missing status 404: %s", logOutput)
	}
}

func TestRequestLogger_SkipsSSE(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RequestLogger(logger)(inner)

	req := httptest.NewRequest("GET", "/-/events", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	logOutput := buf.String()
	if logOutput != "" {
		t.Errorf("SSE path should be skipped, got log: %s", logOutput)
	}
}
