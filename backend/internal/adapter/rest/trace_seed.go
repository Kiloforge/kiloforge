package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"kiloforge/internal/adapter/tracing"
)

// seedTraceRequest is the JSON body for POST /api/traces.
type seedTraceRequest struct {
	TraceID string         `json:"trace_id"`
	Spans   []seedSpanInfo `json:"spans"`
}

type seedSpanInfo struct {
	SpanID     string            `json:"span_id"`
	ParentID   string            `json:"parent_id,omitempty"`
	Name       string            `json:"name"`
	StartTime  string            `json:"start_time"`
	EndTime    string            `json:"end_time"`
	DurationMs int64             `json:"duration_ms"`
	Status     string            `json:"status"`
	Attributes map[string]string `json:"attributes,omitempty"`
	Events     []seedEventInfo   `json:"events,omitempty"`
}

type seedEventInfo struct {
	Name       string            `json:"name"`
	Timestamp  string            `json:"timestamp"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// handleSeedTrace returns an HTTP handler that seeds trace data for E2E tests.
func handleSeedTrace(writer tracing.TraceWriter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req seedTraceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
			return
		}

		if req.TraceID == "" || len(req.Spans) == 0 {
			http.Error(w, `{"error":"trace_id and spans required"}`, http.StatusBadRequest)
			return
		}

		for _, sp := range req.Spans {
			startTime, _ := time.Parse(time.RFC3339Nano, sp.StartTime)
			endTime, _ := time.Parse(time.RFC3339Nano, sp.EndTime)

			var events []tracing.SpanEventInfo
			for _, ev := range sp.Events {
				ts, _ := time.Parse(time.RFC3339Nano, ev.Timestamp)
				events = append(events, tracing.SpanEventInfo{
					Name:       ev.Name,
					Timestamp:  ts,
					Attributes: ev.Attributes,
				})
			}

			summary := tracing.SpanSummary{
				TraceID:    req.TraceID,
				SpanID:     sp.SpanID,
				ParentID:   sp.ParentID,
				Name:       sp.Name,
				StartTime:  startTime,
				EndTime:    endTime,
				DurationMs: sp.DurationMs,
				Status:     sp.Status,
				Attributes: sp.Attributes,
				Events:     events,
			}

			if err := writer.SeedSpan(summary); err != nil {
				http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"trace_id":   req.TraceID,
			"span_count": len(req.Spans),
		})
	}
}
