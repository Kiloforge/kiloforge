package tracing

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestStore_RecordAndQuery(t *testing.T) {
	store := NewStore()
	proc := NewStoreProcessor(store)
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(proc))
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	tracer := tp.Tracer("test")

	// Create a parent and child span.
	ctx, parent := tracer.Start(context.Background(), "track/abc")
	_, child := tracer.Start(ctx, "phase/1")
	child.End()
	parent.End()

	traces := store.ListTraces()
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}
	if traces[0].SpanCount != 2 {
		t.Errorf("expected 2 spans in trace, got %d", traces[0].SpanCount)
	}
	if traces[0].RootName != "track/abc" {
		t.Errorf("expected root name 'track/abc', got %q", traces[0].RootName)
	}

	spans := store.GetTrace(traces[0].TraceID)
	if len(spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(spans))
	}
}

func TestStore_GetTrace_NotFound(t *testing.T) {
	store := NewStore()
	spans := store.GetTrace("nonexistent")
	if len(spans) != 0 {
		t.Errorf("expected 0 spans for nonexistent trace, got %d", len(spans))
	}
}

func TestStore_ListTraces_Empty(t *testing.T) {
	store := NewStore()
	traces := store.ListTraces()
	if len(traces) != 0 {
		t.Errorf("expected 0 traces, got %d", len(traces))
	}
}
