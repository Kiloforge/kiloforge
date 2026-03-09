package tracing

import (
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// TraceReader provides read access to trace and span data.
// Both the in-memory Store and sqlite.TraceStore implement this interface.
type TraceReader interface {
	ListTraces() []TraceSummary
	GetTrace(traceID string) []SpanSummary
	FindByTrackID(trackID string) []TraceSummary
	FindBySessionID(sessionID string) []TraceSummary
}

// SpanRecorder records completed spans. Used by StoreProcessor to decouple
// from the concrete Store type.
type SpanRecorder interface {
	Record(span sdktrace.ReadOnlySpan) error
}

// TraceWriter supports direct insertion of span summaries without going through OTel.
// Used for E2E test seeding.
type TraceWriter interface {
	SeedSpan(span SpanSummary) error
}
