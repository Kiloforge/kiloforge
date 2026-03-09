package tracing

import (
	"context"
	"log"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// StoreProcessor is an OTel SpanProcessor that records completed spans
// into a SpanRecorder (in-memory Store or SQLite TraceStore) for API queries.
type StoreProcessor struct {
	store SpanRecorder
}

// NewStoreProcessor creates a processor backed by the given span recorder.
func NewStoreProcessor(store SpanRecorder) *StoreProcessor {
	return &StoreProcessor{store: store}
}

func (p *StoreProcessor) OnStart(_ context.Context, _ sdktrace.ReadWriteSpan) {}

func (p *StoreProcessor) OnEnd(span sdktrace.ReadOnlySpan) {
	if err := p.store.Record(span); err != nil {
		log.Printf("trace store: record span: %v", err)
	}
}

func (p *StoreProcessor) Shutdown(_ context.Context) error { return nil }
func (p *StoreProcessor) ForceFlush(_ context.Context) error { return nil }
