package tracing

import (
	"sync"
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// SpanSummary is a lightweight representation of a completed span for API queries.
type SpanSummary struct {
	TraceID    string            `json:"trace_id"`
	SpanID     string            `json:"span_id"`
	ParentID   string            `json:"parent_id,omitempty"`
	Name       string            `json:"name"`
	StartTime  time.Time         `json:"start_time"`
	EndTime    time.Time         `json:"end_time"`
	DurationMs int64             `json:"duration_ms"`
	Status     string            `json:"status"`
	Attributes map[string]string `json:"attributes,omitempty"`
	Events     []SpanEventInfo   `json:"events,omitempty"`
}

// SpanEventInfo is a lightweight span event.
type SpanEventInfo struct {
	Name       string            `json:"name"`
	Timestamp  time.Time         `json:"timestamp"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// TraceSummary groups spans by trace ID with aggregate info.
type TraceSummary struct {
	TraceID   string    `json:"trace_id"`
	RootName  string    `json:"root_name"`
	SpanCount int       `json:"span_count"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// Store holds completed spans in memory for API queries.
type Store struct {
	mu    sync.RWMutex
	spans []SpanSummary

	// Secondary indexes: attribute value → set of trace IDs.
	trackIndex   map[string]map[string]struct{} // track.id → trace IDs
	sessionIndex map[string]map[string]struct{} // session.id → trace IDs
}

// NewStore creates a new in-memory trace store.
func NewStore() *Store {
	return &Store{
		trackIndex:   make(map[string]map[string]struct{}),
		sessionIndex: make(map[string]map[string]struct{}),
	}
}

// Record adds a completed span to the store.
func (s *Store) Record(span sdktrace.ReadOnlySpan) error {
	attrs := make(map[string]string)
	for _, kv := range span.Attributes() {
		attrs[string(kv.Key)] = kv.Value.Emit()
	}

	var events []SpanEventInfo
	for _, ev := range span.Events() {
		evAttrs := make(map[string]string)
		for _, kv := range ev.Attributes {
			evAttrs[string(kv.Key)] = kv.Value.Emit()
		}
		events = append(events, SpanEventInfo{
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

	summary := SpanSummary{
		TraceID:    span.SpanContext().TraceID().String(),
		SpanID:     span.SpanContext().SpanID().String(),
		ParentID:   parentID,
		Name:       span.Name(),
		StartTime:  span.StartTime(),
		EndTime:    span.EndTime(),
		DurationMs: span.EndTime().Sub(span.StartTime()).Milliseconds(),
		Status:     status,
		Attributes: attrs,
		Events:     events,
	}

	s.mu.Lock()
	s.spans = append(s.spans, summary)
	s.indexSpan(summary)
	s.mu.Unlock()
	return nil
}

// indexSpan populates secondary indexes from span attributes. Must be called with mu held.
func (s *Store) indexSpan(sp SpanSummary) {
	if trackID, ok := sp.Attributes["track.id"]; ok && trackID != "" {
		if s.trackIndex[trackID] == nil {
			s.trackIndex[trackID] = make(map[string]struct{})
		}
		s.trackIndex[trackID][sp.TraceID] = struct{}{}
	}
	if sessionID, ok := sp.Attributes["session.id"]; ok && sessionID != "" {
		if s.sessionIndex[sessionID] == nil {
			s.sessionIndex[sessionID] = make(map[string]struct{})
		}
		s.sessionIndex[sessionID][sp.TraceID] = struct{}{}
	}
}

// ListTraces returns unique trace summaries.
func (s *Store) ListTraces() []TraceSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	traces := make(map[string]*TraceSummary)
	for _, sp := range s.spans {
		t, ok := traces[sp.TraceID]
		if !ok {
			t = &TraceSummary{
				TraceID:   sp.TraceID,
				StartTime: sp.StartTime,
				EndTime:   sp.EndTime,
			}
			traces[sp.TraceID] = t
		}
		t.SpanCount++
		if sp.ParentID == "" {
			t.RootName = sp.Name
		}
		if sp.StartTime.Before(t.StartTime) {
			t.StartTime = sp.StartTime
		}
		if sp.EndTime.After(t.EndTime) {
			t.EndTime = sp.EndTime
		}
	}

	result := make([]TraceSummary, 0, len(traces))
	for _, t := range traces {
		result = append(result, *t)
	}
	return result
}

// FindByTrackID returns trace summaries for traces containing the given track.id attribute.
func (s *Store) FindByTrackID(trackID string) []TraceSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	traceIDs, ok := s.trackIndex[trackID]
	if !ok {
		return nil
	}
	return s.traceSummariesForIDs(traceIDs)
}

// FindBySessionID returns trace summaries for traces containing the given session.id attribute.
func (s *Store) FindBySessionID(sessionID string) []TraceSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	traceIDs, ok := s.sessionIndex[sessionID]
	if !ok {
		return nil
	}
	return s.traceSummariesForIDs(traceIDs)
}

// traceSummariesForIDs builds TraceSummary objects for the given trace IDs. Must be called with mu held.
func (s *Store) traceSummariesForIDs(traceIDs map[string]struct{}) []TraceSummary {
	traces := make(map[string]*TraceSummary)
	for _, sp := range s.spans {
		if _, ok := traceIDs[sp.TraceID]; !ok {
			continue
		}
		t, ok := traces[sp.TraceID]
		if !ok {
			t = &TraceSummary{
				TraceID:   sp.TraceID,
				StartTime: sp.StartTime,
				EndTime:   sp.EndTime,
			}
			traces[sp.TraceID] = t
		}
		t.SpanCount++
		if sp.ParentID == "" {
			t.RootName = sp.Name
		}
		if sp.StartTime.Before(t.StartTime) {
			t.StartTime = sp.StartTime
		}
		if sp.EndTime.After(t.EndTime) {
			t.EndTime = sp.EndTime
		}
	}

	result := make([]TraceSummary, 0, len(traces))
	for _, t := range traces {
		result = append(result, *t)
	}
	return result
}

// GetTrace returns all spans for a given trace ID.
func (s *Store) GetTrace(traceID string) []SpanSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []SpanSummary
	for _, sp := range s.spans {
		if sp.TraceID == traceID {
			result = append(result, sp)
		}
	}
	return result
}
