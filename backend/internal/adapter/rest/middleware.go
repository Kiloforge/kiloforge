package rest

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// RequestLogger returns middleware that logs every HTTP request using slog.
// It records method, path, status code, and duration.
// Long-lived connections (SSE) are excluded to avoid noisy logs.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip SSE and WebSocket paths — they are long-lived connections.
			if strings.HasSuffix(r.URL.Path, "/events") || strings.HasPrefix(r.URL.Path, "/ws/") {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			duration := time.Since(start)

			logger.Info("http",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", sw.status),
				slog.Duration("dur", duration),
			)
		})
	}
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (sw *statusWriter) WriteHeader(code int) {
	if !sw.wroteHeader {
		sw.status = code
		sw.wroteHeader = true
	}
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *statusWriter) Write(b []byte) (int, error) {
	if !sw.wroteHeader {
		sw.wroteHeader = true
	}
	return sw.ResponseWriter.Write(b)
}
