package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"kiloforge/internal/adapter/tracing"
)

// TraceStore persists trace and span data to SQLite.
// It implements the same API as tracing.Store for drop-in replacement.
type TraceStore struct {
	db *sql.DB
}

// NewTraceStore creates a TraceStore backed by the given database.
func NewTraceStore(db *sql.DB) *TraceStore {
	return &TraceStore{db: db}
}

// Record adds a completed span to the store, persisting to SQLite.
func (s *TraceStore) Record(span sdktrace.ReadOnlySpan) error {
	attrs := make(map[string]string)
	for _, kv := range span.Attributes() {
		attrs[string(kv.Key)] = kv.Value.Emit()
	}

	var events []tracing.SpanEventInfo
	for _, ev := range span.Events() {
		evAttrs := make(map[string]string)
		for _, kv := range ev.Attributes {
			evAttrs[string(kv.Key)] = kv.Value.Emit()
		}
		events = append(events, tracing.SpanEventInfo{
			Name:       ev.Name,
			Timestamp:  ev.Time,
			Attributes: evAttrs,
		})
	}

	parentID := ""
	if span.Parent().IsValid() {
		parentID = span.Parent().SpanID().String()
	}

	status := "ok"
	if span.Status().Code.String() == "Error" {
		status = "error"
	}

	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()
	durationMs := span.EndTime().Sub(span.StartTime()).Milliseconds()

	attrsJSON, _ := json.Marshal(attrs)
	eventsJSON, _ := json.Marshal(events)

	// Upsert trace.
	if _, err := s.db.Exec(
		`INSERT INTO traces (trace_id, started_at, span_count)
		 VALUES (?, ?, 0)
		 ON CONFLICT(trace_id) DO NOTHING`,
		traceID, span.StartTime().Format(time.RFC3339Nano),
	); err != nil {
		return fmt.Errorf("trace store: upsert trace %s: %w", traceID, err)
	}

	// Update trace metadata.
	if parentID == "" {
		if _, err := s.db.Exec(
			"UPDATE traces SET root_span_name = ? WHERE trace_id = ?",
			span.Name(), traceID,
		); err != nil {
			return fmt.Errorf("trace store: update root span name: %w", err)
		}
	}
	if _, err := s.db.Exec(
		"UPDATE traces SET span_count = span_count + 1 WHERE trace_id = ?",
		traceID,
	); err != nil {
		return fmt.Errorf("trace store: update span count: %w", err)
	}

	// Update trace start/end.
	if _, err := s.db.Exec(
		`UPDATE traces SET started_at = MIN(started_at, ?), ended_at = MAX(COALESCE(ended_at, ?), ?)
		 WHERE trace_id = ?`,
		span.StartTime().Format(time.RFC3339Nano),
		span.EndTime().Format(time.RFC3339Nano),
		span.EndTime().Format(time.RFC3339Nano),
		traceID,
	); err != nil {
		return fmt.Errorf("trace store: update trace times: %w", err)
	}

	// Index by track.id and session.id.
	if trackID, ok := attrs["track.id"]; ok && trackID != "" {
		if _, err := s.db.Exec("UPDATE traces SET track_id = ? WHERE trace_id = ?", trackID, traceID); err != nil {
			return fmt.Errorf("trace store: update track_id: %w", err)
		}
	}
	if sessionID, ok := attrs["session.id"]; ok && sessionID != "" {
		if _, err := s.db.Exec("UPDATE traces SET session_id = ? WHERE trace_id = ?", sessionID, traceID); err != nil {
			return fmt.Errorf("trace store: update session_id: %w", err)
		}
	}

	// Insert span.
	if _, err := s.db.Exec(
		`INSERT OR REPLACE INTO spans
		 (span_id, trace_id, parent_id, name, start_time, end_time, duration_ms, status, attributes, events)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		spanID, traceID, parentID, span.Name(),
		span.StartTime().Format(time.RFC3339Nano),
		span.EndTime().Format(time.RFC3339Nano),
		durationMs, status, string(attrsJSON), string(eventsJSON),
	); err != nil {
		return fmt.Errorf("trace store: insert span %s: %w", spanID, err)
	}

	return nil
}

// SeedSpan inserts a pre-built SpanSummary directly (bypassing OTel).
// Used for E2E test seeding.
func (s *TraceStore) SeedSpan(span tracing.SpanSummary) error {
	attrsJSON, _ := json.Marshal(span.Attributes)
	eventsJSON, _ := json.Marshal(span.Events)

	// Upsert trace.
	if _, err := s.db.Exec(
		`INSERT INTO traces (trace_id, started_at, span_count)
		 VALUES (?, ?, 0)
		 ON CONFLICT(trace_id) DO NOTHING`,
		span.TraceID, span.StartTime.Format(time.RFC3339Nano),
	); err != nil {
		return fmt.Errorf("trace store: upsert trace %s: %w", span.TraceID, err)
	}

	// Update root span name.
	if span.ParentID == "" {
		if _, err := s.db.Exec(
			"UPDATE traces SET root_span_name = ? WHERE trace_id = ?",
			span.Name, span.TraceID,
		); err != nil {
			return fmt.Errorf("trace store: update root span name: %w", err)
		}
	}

	// Update span count.
	if _, err := s.db.Exec(
		"UPDATE traces SET span_count = span_count + 1 WHERE trace_id = ?",
		span.TraceID,
	); err != nil {
		return fmt.Errorf("trace store: update span count: %w", err)
	}

	// Update trace start/end.
	if _, err := s.db.Exec(
		`UPDATE traces SET started_at = MIN(started_at, ?), ended_at = MAX(COALESCE(ended_at, ?), ?)
		 WHERE trace_id = ?`,
		span.StartTime.Format(time.RFC3339Nano),
		span.EndTime.Format(time.RFC3339Nano),
		span.EndTime.Format(time.RFC3339Nano),
		span.TraceID,
	); err != nil {
		return fmt.Errorf("trace store: update trace times: %w", err)
	}

	// Index by track.id and session.id.
	if trackID, ok := span.Attributes["track.id"]; ok && trackID != "" {
		if _, err := s.db.Exec("UPDATE traces SET track_id = ? WHERE trace_id = ?", trackID, span.TraceID); err != nil {
			return fmt.Errorf("trace store: update track_id: %w", err)
		}
	}
	if sessionID, ok := span.Attributes["session.id"]; ok && sessionID != "" {
		if _, err := s.db.Exec("UPDATE traces SET session_id = ? WHERE trace_id = ?", sessionID, span.TraceID); err != nil {
			return fmt.Errorf("trace store: update session_id: %w", err)
		}
	}

	// Insert span.
	if _, err := s.db.Exec(
		`INSERT OR REPLACE INTO spans
		 (span_id, trace_id, parent_id, name, start_time, end_time, duration_ms, status, attributes, events)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		span.SpanID, span.TraceID, span.ParentID, span.Name,
		span.StartTime.Format(time.RFC3339Nano),
		span.EndTime.Format(time.RFC3339Nano),
		span.DurationMs, span.Status, string(attrsJSON), string(eventsJSON),
	); err != nil {
		return fmt.Errorf("trace store: insert span %s: %w", span.SpanID, err)
	}

	return nil
}

// ListTraces returns all trace summaries.
func (s *TraceStore) ListTraces() []tracing.TraceSummary {
	rows, err := s.db.Query(
		`SELECT trace_id, root_span_name, span_count, started_at, COALESCE(ended_at, started_at)
		 FROM traces ORDER BY started_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var result []tracing.TraceSummary
	for rows.Next() {
		var t tracing.TraceSummary
		var startStr, endStr string
		var rootName *string
		if err := rows.Scan(&t.TraceID, &rootName, &t.SpanCount, &startStr, &endStr); err != nil {
			continue
		}
		if rootName != nil {
			t.RootName = *rootName
		}
		t.StartTime, _ = time.Parse(time.RFC3339Nano, startStr)
		t.EndTime, _ = time.Parse(time.RFC3339Nano, endStr)
		result = append(result, t)
	}
	return result
}

// GetTrace returns all spans for a given trace ID.
func (s *TraceStore) GetTrace(traceID string) []tracing.SpanSummary {
	rows, err := s.db.Query(
		`SELECT span_id, trace_id, parent_id, name, start_time, end_time,
		        duration_ms, status, attributes, events
		 FROM spans WHERE trace_id = ? ORDER BY start_time`, traceID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanSpans(rows)
}

// FindByTrackID returns trace summaries for a given track ID.
func (s *TraceStore) FindByTrackID(trackID string) []tracing.TraceSummary {
	rows, err := s.db.Query(
		`SELECT trace_id, root_span_name, span_count, started_at, COALESCE(ended_at, started_at)
		 FROM traces WHERE track_id = ? ORDER BY started_at DESC`, trackID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanTraceSummaries(rows)
}

// FindBySessionID returns trace summaries for a given session ID.
func (s *TraceStore) FindBySessionID(sessionID string) []tracing.TraceSummary {
	rows, err := s.db.Query(
		`SELECT trace_id, root_span_name, span_count, started_at, COALESCE(ended_at, started_at)
		 FROM traces WHERE session_id = ? ORDER BY started_at DESC`, sessionID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanTraceSummaries(rows)
}

func scanTraceSummaries(rows *sql.Rows) []tracing.TraceSummary {
	var result []tracing.TraceSummary
	for rows.Next() {
		var t tracing.TraceSummary
		var startStr, endStr string
		var rootName *string
		if err := rows.Scan(&t.TraceID, &rootName, &t.SpanCount, &startStr, &endStr); err != nil {
			continue
		}
		if rootName != nil {
			t.RootName = *rootName
		}
		t.StartTime, _ = time.Parse(time.RFC3339Nano, startStr)
		t.EndTime, _ = time.Parse(time.RFC3339Nano, endStr)
		result = append(result, t)
	}
	return result
}

func scanSpans(rows *sql.Rows) []tracing.SpanSummary {
	var result []tracing.SpanSummary
	for rows.Next() {
		var sp tracing.SpanSummary
		var startStr, endStr string
		var attrsJSON, eventsJSON string
		if err := rows.Scan(
			&sp.SpanID, &sp.TraceID, &sp.ParentID, &sp.Name,
			&startStr, &endStr, &sp.DurationMs, &sp.Status,
			&attrsJSON, &eventsJSON,
		); err != nil {
			continue
		}
		sp.StartTime, _ = time.Parse(time.RFC3339Nano, startStr)
		sp.EndTime, _ = time.Parse(time.RFC3339Nano, endStr)
		if attrsJSON != "" {
			_ = json.Unmarshal([]byte(attrsJSON), &sp.Attributes)
		}
		if eventsJSON != "" {
			_ = json.Unmarshal([]byte(eventsJSON), &sp.Events)
		}
		result = append(result, sp)
	}
	return result
}
